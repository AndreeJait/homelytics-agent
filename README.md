# Homelytics Agent (`homelytics-agent`)

A lightweight, secure edge daemon and CLI tool deployed on merchant-rented Linux and WSL machines. It acts as an isolated execution node that connects back to the central `homelytics-be` control plane, allowing the secure deployment and management of containerized workloads without exposing the host system or demanding manual infrastructure management from the merchant.

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
```

### Why `usecase/` is a root-level package

`port/inbound/` defines *what* the system can do (interfaces). `usecase/` implements *how* it does it (business logic). Separating them:

* **Clear separation** — contracts vs. implementations never mix in one package
* **No circular dependencies** — `usecase/` → `port/inbound/` → `domain/` is always one-directional
* **Generator-friendly** — future tooling can scaffold interface and implementation independently
* **Hex convention** — ports are the boundary, use cases are the application core

---

## Getting Started

### Prerequisites

* Go 1.25+
* containerd and runc (installed automatically by `install.sh`, or manually via your package manager)
* Linux or WSL environment (macOS users can use the provided Dockerfile)

### Quick local run (native Linux/WSL)

```bash
# Build both binaries
make build

# Create a local socket directory so you don't need /opt permissions
mkdir -p var/run
export IPC_SOCKET_PATH=$PWD/var/run/ipc.sock

# Start the daemon
./bin/homelytics-daemon --config files/config/app.yaml &

# Login with the mocked credentials
./bin/homelytics-agent login --email merchant@example.com --password password

# Get a mocked tsnet auth key
./bin/homelytics-agent tsnet auth

# Check overall agent status
./bin/homelytics-agent status
```

### Run with Docker (macOS / Linux)

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

# Run the daemon container
make docker-up

# In another terminal, run CLI commands against the same socket directory
make docker-cli-compose ARGS="login --email merchant@example.com --password password"
make docker-cli-compose ARGS="tsnet auth"
make docker-cli-compose ARGS="status"
make docker-cli-compose ARGS="workload run --image nginx:latest --port 8080:80 --host-network"
make docker-cli-compose ARGS="workload list"

# Or run a self-contained test
make docker-test
```

The Dockerfile uses an Alpine runtime stage with containerd and runc installed. The daemon entrypoint starts `containerd` in the background before launching `homelytics-daemon`. The container runs as `root` inside Docker so it can manage cgroups and namespaces. `/opt/homelytics/run` is mounted from `./var/run` so the host CLI (or another container) can reach the IPC socket. The daemon container uses `network_mode: host` so containers with `--host-network` are reachable on the Docker host's interfaces (Linux only; Docker Desktop on macOS does not support host networking).

### Install on a target machine

```bash
sudo ./install.sh --create-user --start
```

This installs the binaries, containerd/runc, the default config, and the systemd service. The `--create-user` flag creates a dedicated `homelytics` system account; without it the daemon runs as root.

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

tsnet:
  hostname: homelytics-agent
  control_url: ""
  advertise_tags: []
  dir: ""

containerd:
  address: /run/containerd/containerd.sock
  namespace: homelytics
  timeout: 10s

log:
  level: debug
  format: JSON
```

Environment variable overrides: `IPC_SOCKET_PATH`, `CONTAINERD_ADDRESS`, `TSNET_HOSTNAME`, etc.

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

* **`tsnet.auth`** succeeds for that access token. It first tries to call the real Tailscale API using `TAILSCALE_TAILNET` and `TAILSCALE_API_KEY` from the environment. If those variables are not set, it falls back to `homelytics.mock_tsnet_auth_key` when configured. Otherwise it returns:

```json
{
  "auth_key": "tskey-auth-mock-abcde",
  "expires_at": "2026-06-15T00:00:00Z"
}
```

Set `homelytics.mock_mode: false` to wire the real HTTP backend stub, which currently returns an error until `homelytics-be` is implemented.

## Real homelytics-be API contract (Phase 2)

When `homelytics-be` is ready, the agent will switch from the mock adapter to a real HTTPS client. The contract is documented in `docs/homelytics-be-api.md`. Quick reference with example JSON:

### Required endpoints for tsnet auth-key flow

The agent needs exactly two backend calls to get online:

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
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4=..."
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
  "workloads": []
}
```

**Response `202 Accepted`:**

```json
{
  "commands": []
}
```

### Error response format

```json
{
  "code": "INVALID_CREDENTIAL",
  "message": "Invalid email or password"
}
```

---

## Backend Trigger Mechanism

`homelytics-be` can drive the agent through two complementary paths:

1. **Heartbeat polling** — the daemon sends a `POST /v1/agents/heartbeat` request on a configurable interval (default 30s). The backend replies with a list of commands to execute, such as `DEPLOY`, `STOP`, or `DELETE`.

2. **tsnet push listener** — once the agent joins the tailnet, it starts an HTTP server on `tsnet.command_listener_addr` (default `:7373`). `homelytics-be` or any authorized tailnet device can POST a single command to `http://homelytics-agent:7373/v1/commands` for immediate execution.

Both paths use the same command executor, so behavior is identical whether a command arrives via polling or push.

Supported command types:

| Type | Payload | Action |
|------|---------|--------|
| `DEPLOY` | `RunWorkloadRequest` | Pull image, create and start container |
| `STOP` | `WorkloadIDRequest` | Stop a running container |
| `DELETE` | `WorkloadIDRequest` | Stop and remove a container |

The listener and heartbeat are enabled by default and can be toggled via config:

```yaml
heartbeat:
  enabled: true
  interval: 30s

tsnet:
  enable_command_listener: true
  command_listener_addr: ":7373"
```

---

## Current Status

This repository contains the Phase 2 vertical slice: daemon skeleton, mocked control-plane auth with real Tailscale API key provisioning, containerd workload execution, tsnet tailnet joining, heartbeat polling, tsnet push command listener, and the auto-installer. See `SystemDetail.md` for the full five-phase roadmap.

---

## License

MIT
