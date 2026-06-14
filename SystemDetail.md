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

* **Core Language:** Go (1.22+)
* **Container Engine:** Embedded `containerd` + `runc` (Native Linux runtime binaries bundled within the installer)
* **VPN Control Plane:** `tailscale.com/tsnet` (Tailscale embedded directly inside the Go binary as a userspace TCP/IP networking stack)
* **CLI Engine:** Standard POSIX-compliant subcommands (`cobra` or `urfave/cli`)
* **Architecture Pattern:** Hexagonal Architecture (Ports & Adapters) for decoupling business domain logic from infrastructure runtimes.

---

## High-Level Design

The agent is decoupled into a background daemon service (executing with privileges or under an isolated system account) and a standard user-space CLI command utility.

```
+-----------------------------------------------------------------------------------+
|                              MERCHANT COMPUTER (LINUX / WSL)                      |
|                                                                                   |
|  +-----------------------------------------------------------------------------+  |
|  | BACKGROUND SERVICE (Root/Privileged or Dedicated 'homelytics' User Daemon)  |  |
|  |                                                                             |  |
|  |   +--------------------------+       +-----------------------------------+  |  |
|  |   |    Core State Domain     |       |       Embedded tsnet Client       |  |  |
|  |   |  (Hexagonal Orchestrator) <-------> (In-Memory Tailnet Connection)   |  |  |
|  |   +------------+-------------+       +-----------------+-----------------+  |  |
|  |                |                                       |                    |  |
|  |                | Handles Container Lifecycle           | Direct API Channel |  |
|  |                v                                       v                    |  |
|  |   +--------------------------+                         |                    |  |
|  |   |    containerd Go SDK     |                         |                    |  |
|  |   | (Pipes to private socket) |                         |                    |  |
|  |   +------------+-------------+                         |                    |  |
|  +----------------|---------------------------------------|--------------------+  |
|                   |                                       |                     |
|                   | /var/run/homelytics-engine.sock       | Encrypted Tunnel    |
|                   v                                       v                     |
|  +-------------------------------------+       +----------+--------------------+  |
|  | RUNTIME LAYER                       |       | REMOTE CONTROL                 |  |
|  |                                     |       |                                |  |
|  |  +-------------------------------+  |       |     homelytics-be              |  |
|  |  | Bundled containerd Daemon     |  |       |    (Central Control)           |  |
|  |  +---------------+---------------+  |       +--------------------------------+  |
|  |                  |                  |                                          |
|  |                  v                  |                                          |
|  |  +-------------------------------+  |                                          |
|  |  | Isolated Namespaced Container |  |                                          |
|  |  +-------------------------------+  |                                          |
|  +-------------------------------------+                                          |
|                   ^                                                               |
|                   | IPC (Local Unix Domain Socket: /opt/homelytics/run/ipc.sock)   |
|                   v                                                               |
|  +-----------------------------------------------------------------------------+  |
|  | USER INTERFACE LAYER (Standard User Space)                               |  |
|  |                                                                             |  |
|  |   +---------------------------------------------------------------------+   |  |
|  |   |                    homelytics-agent CLI Client                      |   |  |
|  |   +---------------------------------------------------------------------+   |  |
|  +-----------------------------------------------------------------------------+  |
+-----------------------------------------------------------------------------------+

```

### Core Architecture Components

#### 1. Command Line Interface (CLI)

The foreground interface used by the merchant. Running `homelytics-agent login` or `homelytics-agent status` does not run execution tasks locally. Instead, the CLI acts as a lightweight interface client, serialization engine, and validation proxy that formats instructions and forwards them via an internal Inter-Process Communication (IPC) domain socket.

#### 2. Local IPC Gateway (`/opt/homelytics/run/ipc.sock`)

A heavily locked-down Unix domain socket file with explicit file permissions (`0666` for message passing, inside a directory restricted to the application run user). It handles high-speed local data streaming between user commands and the daemon without mapping local TCP ports.

#### 3. Embedded Network Tunnel (`tsnet`)

The networking stack lives purely inside the application's RAM space. By embedding Tailscale inside the Go codebase, the application joins your private Tailnet control lane as an independent node. It listens for deployment instructions exclusively on its assigned Tailscale private IP address, completely bypassing public internet traffic, home routing tables, and local NAT setups.

#### 4. Isolated Container Runtime (`containerd`)

The agent completely bypasses standard Docker Desktop installations. The installation bundles raw, native `containerd` and `runc` binaries, mounting them to an internal, hidden Unix socket (`/opt/homelytics/run/homelytics-engine.sock`). The Core Domain instructs this engine via the official Go SDK to handle registry image acquisition, layer unpacking, and target execution within isolated Linux namespaces.


# Implementation Phases

## Phase 1 — Daemon skeleton, mocked control plane, and installer (current)

Goal: establish the daemon + CLI shape and prove end-to-end command flow without requiring the real `homelytics-be` backend or a live Tailscale control server.

* Build `cmd/daemon` background service and `cmd/cli` user-space client communicating over a Unix domain socket (`ipc.sock`).
* Implement mocked `homelytics-be` login adapter and tsnet auth-key adapter.
* Wire containerd and tsnet via `go-utility/v2/containerdw` and `go-utility/v2/tailscalew/tsnetw` for status/version checks only.
* Extend config with `daemon`, `ipc`, `homelytics`, `tsnet`, and `containerd` blocks.
* Create `install.sh` that builds the binaries, installs containerd/runc via the system package manager, optionally creates a `homelytics` user, and installs a systemd/OpenRC service.

Verification: `homelytics-agent login`, `homelytics-agent tsnet auth`, `homelytics-agent status`, and `homelytics-agent runtime status` all return expected responses while the daemon is running.

## Phase 2 — Real control-plane integration

* Replace the mock backend adapter with a real HTTP/gRPC client for `homelytics-be`.
* Implement secure token storage (encrypted at rest or via keyring) instead of the in-memory session store.
* Fetch the tsnet auth key from `homelytics-be` after login and use it to join the tailnet.
* Add heartbeat/telemetry so the control plane can see agent health and assigned workloads.

## Phase 3 — Workload execution

* Implement container lifecycle use cases: pull image, create container, start task, stop, delete.
* Add CLI commands: `homelytics-agent workload list`, `workload run`, `workload stop`, `workload logs`.
* Route workload commands from `homelytics-be` to the daemon over the tailnet (tsnet listener).
* Add log and metric collection from running containers.

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

# Reference Document
1. https://containerd.io/docs/
2. https://tailscale.com/docs/

# Repository Reference
1. Go Utility: https://github.com/AndreeJait/go-utility: refer this repository and use the V2. The utility is already cloned locally at `/Users/andreepanjaitan/go/src/github.com/AndreeJait/go-utility`. When a needed wrapper exists in `go-utility/v2`, import it (e.g. `github.com/AndreeJait/go-utility/v2/containerdw`, `github.com/AndreeJait/go-utility/v2/tailscalew/tsnetw`) and run `go mod tidy`.
