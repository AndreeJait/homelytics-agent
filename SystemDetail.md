# Homelytics Agent (`homelytics-agent`)

The `homelytics-agent` is a lightweight, secure edge daemon and CLI tool deployed on merchant-rented Linux and WSL machines. It acts as an isolated execution node that connects back to the central `homelytics-be` control plane, allowing the secure deployment and management of containerized workloads without exposing the host system or demanding manual infrastructure management from the merchant.

---

## Purpose

The primary purpose of the agent is to turn a standard consumer Linux or WSL machine into a secure, rented edge computing resource.

* **Zero Configuration for Merchants:** Simplifies the onboarding process via an automated shell script that handles dependencies and creates standalone system environments.
* **Tamper-Proof Runtime:** Hides the container orchestration layer behind strict OS permissions and custom sockets, ensuring merchants cannot view, manipulate, or accidentally destroy running workloads.
* **Invisible Networking:** Uses an in-memory VPN stack that eliminates the need for port forwarding, public IP allocation, or host-level firewall alterations.

---

## Tech Stack

* **Core Language:** Go (1.25+)
* **Container Engine:** Embedded `containerd` + `runc` (installed via the system package manager)
* **VPN Control Plane:** `tailscale.com/tsnet` (Tailscale embedded directly inside the Go binary as a userspace TCP/IP networking stack)
* **CLI Engine:** Standard POSIX subcommands using the Go `flag` package
* **Architecture Pattern:** Hexagonal Architecture (Ports & Adapters) for decoupling business domain logic from infrastructure runtimes.

---

## High-Level Design

The agent is decoupled into a background daemon service and a standard user-space CLI command utility.

```mermaid
flowchart TB
    subgraph Merchant["Merchant Computer (Linux / WSL)"]
        subgraph Daemon["Background Service (root or homelytics user)"]
            Core["Core State Domain\nHexagonal Orchestrator"]
            TSNet["Embedded tsnet Client\nIn-Memory Tailnet Connection"]
            CDClient["containerd Go SDK\nPrivate socket"]
        end

        subgraph Runtime["Runtime Layer"]
            CDDaemon["containerd Daemon"]
            Container["Isolated Namespaced Container"]
        end

        subgraph UI["User Interface Layer"]
            CLI["homelytics-agent CLI Client"]
        end
    end

    subgraph Remote["Remote Control"]
        BE["homelytics-be\nCentral Control"]
    end

    Core --"Container Lifecycle"--> CDClient
    Core --"Direct API Channel"--> TSNet
    CDClient --"Unix socket\n/run/containerd/containerd.sock"--> CDDaemon
    CDDaemon --> Container
    TSNet --"Encrypted Tunnel"--> BE
    CLI --"IPC Unix socket\n/opt/homelytics/run/ipc.sock"--> Core
```

### Core Architecture Components

#### 1. Command Line Interface (CLI)

The foreground interface used by the merchant. Running `homelytics-agent login` or `homelytics-agent status` does not run execution tasks locally. Instead, the CLI acts as a lightweight interface client, serialization engine, and validation proxy that formats instructions and forwards them via the internal Inter-Process Communication (IPC) domain socket.

#### 2. Local IPC Gateway (`/opt/homelytics/run/ipc.sock`)

A heavily locked-down Unix domain socket file with explicit file permissions. It handles high-speed local data streaming between user commands and the daemon without mapping local TCP ports.

The IPC protocol uses newline-delimited JSON command envelopes:

* `CommandRequest{ID, Method, Payload}`
* `CommandResponse{ID, OK, Data, Error}`

Current methods:

| Method | Payload | Response |
|--------|---------|----------|
| `login` | `LoginRequest` | `AuthSession` |
| `tsnet.auth` | - | `TSNetAuthKey` |
| `runtime.status` | - | `RuntimeStatus` |
| `status` | - | `AgentStatus` |
| `workload.run` | `RunWorkloadRequest` | `Workload` |
| `workload.stop` | `WorkloadIDRequest` | `Workload` |
| `workload.delete` | `WorkloadIDRequest` | `Workload` |
| `workload.list` | - | `WorkloadList` |
| `workload.status` | `WorkloadIDRequest` | `Workload` |

#### 3. Embedded Network Tunnel (`tsnet`)

The networking stack lives purely inside the application's RAM space. By embedding Tailscale inside the Go codebase, the application joins your private Tailnet control lane as an independent node. It listens for deployment instructions exclusively on its assigned Tailscale private IP address, completely bypassing public internet traffic, home routing tables, and local NAT setups.

#### 4. Isolated Container Runtime (`containerd`)

The agent completely bypasses standard Docker Desktop installations. The installer uses the system package manager to install native `containerd` and `runc` binaries. The Core Domain instructs this engine via the official Go SDK to handle registry image acquisition, layer unpacking, and target execution within isolated Linux namespaces.

---

## Architecture

Strict inward dependency direction: **adapters → ports → domain**. Never the reverse.

```
cmd/
  daemon/              Background daemon entry point (wiring + DI)
  cli/                 User-space CLI client
  migrate/             Migration runner (kept for future use)
adapter/               Concrete implementations of ports
  inbound/ipc/         Unix-socket IPC server and command router
  outbound/            containerd runtime, tsnet VPN, backend client, session store
port/                  Interface contracts
  inbound/             Driving ports (use case interfaces)
  outbound/            Driven ports (repository/service interfaces)
usecase/               Use case implementations (root-level, separate from ports)
domain/                Core business logic (zero external dependencies)
  entity/              Domain models
  error/               Domain errors
config/                Configuration loading (Go code only)
files/                 Non-Go files
  config/              YAML configs (app.yaml + gitignored app.local.yaml)
  containerd/          containerd config.toml and Docker entrypoint
```

### Why `usecase/` is a root-level package

`port/inbound/` defines *what* the system can do (interfaces). `usecase/` implements *how* it does it (business logic). Separating them:

* **Clear separation** — contracts vs. implementations never mix in one package
* **No circular dependencies** — `usecase/` → `port/inbound/` → `domain/` is always one-directional
* **Generator-friendly** — future tooling can scaffold interface and implementation independently
* **Hex convention** — ports are the boundary, use cases are the application core

---

## Implementation Phases

## Phase 1 — Daemon skeleton, mocked control plane, and installer (complete)

Goal: establish the daemon + CLI shape and prove end-to-end command flow without requiring the real `homelytics-be` backend or a live Tailscale control server.

* Build `cmd/daemon` background service and `cmd/cli` user-space client communicating over a Unix domain socket (`ipc.sock`).
* Implement mocked `homelytics-be` login adapter and tsnet auth-key adapter.
* Wire containerd and tsnet via `go-utility/v2/containerdw` and `go-utility/v2/tailscalew/tsnetw` for status/version checks only.
* Extend config with `daemon`, `ipc`, `homelytics`, `tsnet`, `containerd`, `heartbeat`, and `graceful` blocks.
* Create `install.sh` that builds the binaries, installs containerd/runc via the system package manager, optionally creates a `homelytics` user, and installs a systemd/OpenRC service.
* Add Dockerfile and docker-compose setup for macOS / Linux local testing.

Verification: `homelytics-agent login`, `homelytics-agent tsnet auth`, `homelytics-agent status`, `homelytics-agent runtime status`, and `homelytics-agent workload run` all return expected responses while the daemon is running.

## Phase 2 — Workload execution, real tailnet auth, and backend triggers (current)

Goal: make the agent actually deploy containers, join a real Tailscale tailnet, and receive triggers from `homelytics-be`.

* Extend the containerd runtime adapter to pull images, create/start/stop/delete containers, and list container status.
* Add workload use cases (`Run`, `Stop`, `Delete`, `List`, `Status`) and inbound ports.
* Add `workload.*` IPC methods and CLI subcommands.
* Make the mock backend call the real Tailscale API to provision an auth key when `TAILSCALE_TAILNET` and `TAILSCALE_API_KEY` are set.
* Start the embedded tsnet node with the fetched auth key so the agent joins the tailnet.
* Add a heartbeat loop that reports agent state and executes commands returned by the control plane.
* Add a tsnet HTTP push listener (`POST /v1/commands`) so `homelytics-be` can trigger actions immediately over the tailnet.
* Use host networking for containers so deployed workloads are reachable on the agent's tailnet IP without additional port forwarding.

Verification: Deploy nginx in Docker, verify it is reachable from another PC on the same tailnet, and verify both heartbeat-polling and tsnet-push command paths work.

## Phase 3 — Real control-plane integration

* Replace the mock backend adapter with a real HTTPS client for `homelytics-be`.
* Implement secure token storage (encrypted at rest or via keyring) instead of the in-memory session store.
* Token refresh before expiry and retry/backoff for backend calls.
* Richer heartbeat telemetry (CPU, memory, network, container logs streaming).

## Phase 4 — Security hardening and merchant isolation

* Lock down IPC socket permissions and directory ACLs.
* Run the daemon under a dedicated, unprivileged `homelytics` user where possible; use capabilities or sudo only for containerd operations that require it.
* Encrypt local state and credentials.
* Implement tamper-evident logging and remote attestation if required.

## Phase 5 — Distribution and operations

* Build release binaries for amd64/arm64 Linux and WSL.
* Produce signed `.deb`/`.rpm` packages and a one-line curl installer.
* Add upgrade mechanism via the control plane.
* Add monitoring, alerting, and remote diagnostics.

---

## Configuration

Config files in `files/config/`:

| File | Purpose |
|------|---------|
| `app.yaml` | Base config (committed) |
| `app.local.yaml` | Local overrides (gitignored) |
| `app.local.yaml.example` | Template — copy to `app.local.yaml` |

**Override priority** (highest wins): environment variables → `app.local.yaml` → `app.yaml`

Key blocks:

```yaml
app:
  name: homelytics-agent
  env: development

daemon:
  run_dir: /opt/homelytics/run
  etc_dir: /opt/homelytics/etc
  log_dir: /opt/homelytics/log

ipc:
  socket_path: /opt/homelytics/run/ipc.sock

homelytics:
  mock_mode: true          # use in-memory mock backend when true
  base_url: https://api.homelytics.internal
  mock_tsnet_auth_key: ""  # offline fallback key

tsnet:
  hostname: homelytics-agent
  control_url: ""
  advertise_tags: []
  dir: ""
  enable_command_listener: true
  command_listener_addr: ":7373"

containerd:
  address: /run/containerd/containerd.sock
  namespace: homelytics
  timeout: 10s

heartbeat:
  enabled: true
  interval: 30s

log:
  level: debug
  format: JSON
```

Environment variable overrides: `IPC_SOCKET_PATH`, `CONTAINERD_ADDRESS`, `TSNET_HOSTNAME`, `TAILSCALE_TAILNET`, `TAILSCALE_API_KEY`, etc.

---

## Backend Mocking

Because the real `homelytics-be` control plane does not exist yet, the daemon ships with an in-memory mock backend enabled by default (`homelytics.mock_mode: true`):

* **Login** succeeds for `merchant@example.com` / `password` and returns:

```json
{
  "access_token": "mock-token-12345",
  "refresh_token": "mock-token-12345",
  "token_type": "Bearer",
  "expires_in": 900,
  "expires_at": "2026-06-15T00:00:00Z"
}
```

* **`tsnet.auth`** fetches a real Tailscale auth key when `TAILSCALE_TAILNET` and `TAILSCALE_API_KEY` are set. If those variables are absent, it falls back to `homelytics.mock_tsnet_auth_key` when configured. The returned key looks like:

```json
{
  "auth_key": "tskey-auth-mock-abcde",
  "expires_at": "2026-06-15T00:00:00Z"
}
```

Set `homelytics.mock_mode: false` to wire the real HTTP backend stub, which currently returns an error until `homelytics-be` is implemented.

---

## Real homelytics-be API contract (Phase 3)

When `homelytics-be` is ready, the agent will switch from the mock adapter to a real HTTPS client. The agent needs exactly two backend calls to get online:

1. `POST /v1/auth/login` — authenticate the merchant and obtain an access token.
2. `GET /v1/agents/auth-key` — present the access token and receive a Tailscale auth key.

```mermaid
sequenceDiagram
    participant CLI as homelytics-agent CLI
    participant Daemon as homelytics-daemon
    participant BE as homelytics-be

    CLI->>+Daemon: login --email merchant@example.com --password password
    Daemon->>+BE: POST /v1/auth/login
    BE-->>-Daemon: access_token + refresh_token
    Daemon->>BE: GET /v1/agents/auth-key
    BE-->>-Daemon: tskey-auth-...
    Daemon-->>-CLI: AuthSession + TSNetAuthKey
```

### `POST /v1/auth/login`

**Request:**

```json
{
  "email": "merchant@example.com",
  "password": "password"
}
```

**Response `200 OK`:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4=...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### `POST /v1/auth/refresh`

**Request:**

```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2gtdG9rZW4=..."
}
```

**Response `200 OK`:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "bmV3LXJlZnJlc2gtdG9rZW4=...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### `GET /v1/agents/auth-key`

**Headers:**

```text
Authorization: Bearer <access_token>
```

**Response `200 OK`:**

```json
{
  "auth_key": "tskey-auth-abc123def456ghi789",
  "expires_at": "2026-06-15T00:00:00Z"
}
```

### `POST /v1/agents/heartbeat`

**Headers:**

```text
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request:**

```json
{
  "agent_id": "agent-uuid-or-hostname",
  "hostname": "homelytics-agent",
  "tsnet_ip": "100.64.0.5",
  "version": "v0.2.0",
  "timestamp": "2026-06-14T12:00:00Z",
  "runtime": {
    "connected": true,
    "version": "1.7.23",
    "revision": ""
  },
  "workloads": [
    {
      "id": "workload-1",
      "status": "running",
      "image": "docker.io/library/nginx:latest"
    }
  ]
}
```

**Response `202 Accepted`:**

```json
{
  "commands": [
    {
      "id": "cmd-uuid",
      "type": "DEPLOY",
      "payload": {
        "image": "nginx:latest",
        "ports": {"80": "8080"},
        "host_network": true
      }
    }
  ]
}
```

Supported command types:

| Type | Payload | Action |
|------|---------|--------|
| `DEPLOY` | `RunWorkloadRequest` | Pull image, create and start container |
| `STOP` | `WorkloadIDRequest` | Stop a running container |
| `DELETE` | `WorkloadIDRequest` | Stop and remove a container |

### tsnet push listener

When tsnet is enabled and `tsnet.enable_command_listener: true`, the agent also listens on the tailnet for immediate push commands:

```text
POST http://homelytics-agent:7373/v1/commands
Content-Type: application/json

{
  "id": "cmd-uuid",
  "type": "DEPLOY",
  "payload": {
    "image": "nginx:latest",
    "ports": {"80": "8080"},
    "host_network": true
  }
}
```

A `GET /health` endpoint is also exposed on the same listener for simple reachability checks.

### Error response format

```json
{
  "code": "INVALID_CREDENTIAL",
  "message": "Invalid email or password"
}
```

---

## Workload Execution

### Domain model

`domain/entity/workload.go`:

```go
type Workload struct {
    ID        string            `json:"id"`
    Image     string            `json:"image"`
    Status    string            `json:"status"`
    IPAddress string            `json:"ip_address,omitempty"`
    Ports     map[string]string `json:"ports,omitempty"`
    CreatedAt time.Time         `json:"created_at"`
}

type RunWorkloadRequest struct {
    ID          string            `json:"id"`
    Image       string            `json:"image"`
    Command     []string          `json:"command,omitempty"`
    Args        []string          `json:"args,omitempty"`
    Env         map[string]string `json:"env,omitempty"`
    Ports       map[string]string `json:"ports,omitempty"` // key=container port, value=host port
    HostNetwork bool              `json:"host_network"`
}
```

### Runtime behavior

* `workload run` pulls the image, creates a new containerd container, and starts the task.
* Containers run in the host network namespace (`oci.WithHostNamespace(specs.NetworkNamespace)`) so a workload bound to `:8080` is reachable on the agent's tailnet IP at port `8080`.
* If creation succeeds but start fails, the container is deleted automatically.
* `workload stop` terminates the task with `SIGTERM` and a 30-second timeout.
* `workload delete` removes the container and its snapshot.

### CLI examples

```bash
homelytics-agent workload run --image nginx:latest --port 8080:80 --host-network
homelytics-agent workload list
homelytics-agent workload status --id workload-1234567890
homelytics-agent workload stop --id workload-1234567890
homelytics-agent workload delete --id workload-1234567890
```

---

## Docker Local Testing (macOS / Linux)

The Docker setup runs containerd inside a privileged container. For the Tailscale auth key to be real, export your credentials first.

```bash
# Export Tailscale credentials so the mock backend can call the Tailscale API
export TAILSCALE_TAILNET=your-tailnet.ts.net
export TAILSCALE_API_KEY=tskey-api-xxxxxxxxxxxx

# Optional: set a hardcoded auth key for offline testing
# cp files/config/app.local.yaml.example files/config/app.local.yaml
# Edit files/config/app.local.yaml and set homelytics.mock_tsnet_auth_key.

# Build the image
make docker-build

# Start the daemon container
make docker-up

# In another terminal, run CLI commands against the same socket directory
make docker-cli-compose ARGS="login --email merchant@example.com --password password"
make docker-cli-compose ARGS="tsnet auth"
make docker-cli-compose ARGS="status"
make docker-cli-compose ARGS="runtime status"
make docker-cli-compose ARGS="workload run --image nginx:latest --port 8080:80 --host-network"
make docker-cli-compose ARGS="workload list"

# Or run a self-contained test
make docker-test
```

The Dockerfile uses an Alpine runtime stage with containerd and runc installed. The daemon entrypoint starts `containerd` in the background before launching `homelytics-daemon`. The container runs as `root` inside Docker so it can manage cgroups and namespaces. `/opt/homelytics/run` is mounted from `./var/run` so the host CLI (or another container) can reach the IPC socket. The daemon container uses `network_mode: host` so containers with `--host-network` are reachable on the Docker host's interfaces (Linux only; Docker Desktop on macOS does not support host networking).

### End-to-end verification checklist

1. Build and start the daemon container:
   ```bash
   export TAILSCALE_TAILNET=your-tailnet.ts.net
   export TAILSCALE_API_KEY=tskey-api-...
   make docker-build
   make docker-up
   ```

2. Login and join the tailnet:
   ```bash
   make docker-cli-compose ARGS="login --email merchant@example.com --password password"
   make docker-cli-compose ARGS="tsnet auth"
   ```
   Verify the agent appears in the Tailscale admin console as `homelytics-agent`.

3. Deploy nginx:
   ```bash
   make docker-cli-compose ARGS="workload run --image nginx:latest --port 8080:80 --host-network"
   make docker-cli-compose ARGS="workload list"
   ```

4. From another PC on the same tailnet, access:
   ```
   http://homelytics-agent:8080
   ```
   or using the agent's Tailscale IP:
   ```
   http://<tailscale-ip>:8080
   ```

5. Test backend trigger via heartbeat: the mock backend currently returns an empty command list, but the code path is wired; once `homelytics-be` returns `DEPLOY` commands the agent will execute them automatically.

6. Test backend trigger via tsnet push: from another tailnet device, POST a command to `http://homelytics-agent:7373/v1/commands` and verify the container starts.

---

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build `bin/homelytics-daemon` and `bin/homelytics-agent` |
| `make build-daemon` | Build only the daemon binary |
| `make build-cli` | Build only the CLI binary |
| `make run-daemon` | Run the daemon with the default config |
| `make run-cli ARGS="status"` | Run the CLI via `go run` |
| `make test` | Run all tests |
| `make vet` | Run static analysis |
| `make tidy` | Clean up dependencies |
| `make install` | Run the installer (requires root) |
| `make docker-build` | Build the Docker image |
| `make docker-up` | Start the daemon container in the background |
| `make docker-down` | Stop the daemon container |
| `make docker-cli-compose ARGS="status"` | Run a one-off CLI command in a container |
| `make docker-test` | Build image and run login/tsnet/workload test |
| `make docker-workload-run IMAGE=nginx:latest PORT=8080:80` | Deploy a workload from a container |
| `make migrate-new name=foo` | Create a new migration |
| `make migrate-up` | Run pending migrations |
| `make migrate-down` | Roll back last migration |
| `make migrate-fresh` | Drop all + re-run all migrations |

---

## CLI Commands

```bash
homelytics-agent login --email=<email> --password=<password>
homelytics-agent tsnet auth
homelytics-agent runtime status
homelytics-agent status
homelytics-agent workload run --image=<image> [--id=<id>] [--port=<host>:<container>] [--host-network]
homelytics-agent workload list
homelytics-agent workload status --id=<id>
homelytics-agent workload stop --id=<id>
homelytics-agent workload delete --id=<id>
```

All client commands accept `--socket-path=<path>` to override the IPC socket location.

---

## Dependency Injection

DI uses [uber-go/dig](https://github.com/uber-go/dig). Provider functions live in `cmd/daemon/`:

* `infra.go` — containerd runtime, tsnet VPN, backend client, session store.
* `service.go` — login, tsnet-auth, runtime-status, agent-status, and workload use cases.
* `ipc.go` — IPC server provider.
* `background.go` — command executor, heartbeat loop, tsnet command listener.
* `wire.go` — container setup and `CleanupCollector` for graceful shutdown.

To add a new dependency, write a provider function with its dependencies as parameters and call `c.Provide(...)` in the appropriate `provide*` function. Dig resolves the graph automatically.

---

## Reference Document

1. https://containerd.io/docs/
2. https://tailscale.com/docs/

## Repository Reference

1. Go Utility: https://github.com/AndreeJait/go-utility: refer this repository and use the V2. The utility is already cloned locally at `/Users/andreepanjaitan/go/src/github.com/AndreeJait/go-utility`. When a needed wrapper exists in `go-utility/v2`, import it (e.g. `github.com/AndreeJait/go-utility/v2/containerdw`, `github.com/AndreeJait/go-utility/v2/tailscalew/tsnetw`) and run `go mod tidy`.
