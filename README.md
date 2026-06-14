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

```bash
# Build the image
make docker-build

# Optional: create a local override file
# (the example below uses a local socket path so the CLI can reach it easily)
cp files/config/app.local.yaml.example files/config/app.local.yaml
# Then edit files/config/app.local.yaml as needed.

# Run the daemon container
make docker-run

# In another terminal, run CLI commands against the same socket directory
make docker-cli ARGS="login --email merchant@example.com --password password"
make docker-cli ARGS="tsnet auth"
make docker-cli ARGS="status"

# Or run a self-contained test
make docker-test
```

The Dockerfile uses an Alpine runtime stage with containerd and runc installed. The daemon runs as a `homelytics` system user; `/opt/homelytics/run` is mounted from `./var/run` so the host CLI (or another container) can reach the IPC socket. If `files/config/app.local.yaml` exists, it is mounted into the container as `/opt/homelytics/etc/app.local.yaml` and merged on top of the base config.

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
| `make docker-run` | Run the daemon container in the foreground |
| `make docker-cli ARGS="status"` | Run a one-off CLI command in a container |
| `make docker-test` | Build image and run a self-contained login/status test |
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
```

All client commands accept `--socket-path=<path>` to override the IPC socket location.

---

## Backend Mocking

Because the real `homelytics-be` control plane does not exist yet, the daemon ships with an in-memory mock backend enabled by default (`homelytics.mock_mode: true`):

* **Login** succeeds for `merchant@example.com` / `password` and returns `mock-token-12345`.
* **`tsnet.auth`** succeeds for that token and returns `tskey-auth-mock-abcde`.

Set `homelytics.mock_mode: false` to wire the real HTTP backend stub, which currently returns an error until `homelytics-be` is implemented.

---

## Current Status

This repository contains the Phase 1 vertical slice: daemon skeleton, mocked control-plane auth, mocked tsnet auth-key retrieval, containerd/tsnet scaffolding, and the auto-installer. See `SystemDetail.md` for the full five-phase roadmap.

---

## License

MIT
