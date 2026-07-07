# System Overview

## 1. Introduction

This project is a self-hosted web service that manages terminal-based software
spaces (called **Spaces**) and provides browser-based access to their terminal
interfaces via WebSocket streaming. It is designed for small-scale use — at
most a handful of users (friends, family).

**Key design choices:**

- Single Go binary (management service + embedded frontend + SQLite)
- A reverse proxy (e.g., Caddy, nginx) handles TLS termination for the
  management domain
- Spaces run in isolated environments managed by **providers** — pluggable
  implementations that abstract the underlying technology (Docker, Firecracker,
  etc.). See [providers.md](providers.md).
- The management service is **provider-agnostic** — it interacts with
  environments exclusively through the Provider interface.
- Providers are plumbed at startup. Only available providers are presented for
  selection.
- WebSocket relay through the Go service is the primary access path — clients
  never connect to spaces directly

## 2. Architecture

All HTTP and WebSocket traffic for the management domain flows through a
reverse proxy which terminates TLS:

- The **management domain** (`management.example.com`) targets the Go Management
  Service, which serves the Angular SPA, REST API, and WebSocket endpoints.
- **Terminal-based spaces** are accessed through the management domain — their
  terminal I/O is streamed through the Go Management Service via WebSocket.

The Go Management Service is a single binary composed of the following:

- **HTTP Server** — serves the embedded Angular frontend (`/`), the REST API
  (`/api/*`), and WebSocket connections (`/ws/*`).
- **Provider Interface** — a Go interface in `internal/model/provider/` that
  abstracts environment lifecycle management.
- **SQLite** — embedded database storing users, spaces, and sessions.

Spaces run in isolated environments managed by **providers**:

- **Docker** — each space is a container; suitable for most environments.
- **Firecracker** — each space is a microVM; stronger isolation for advanced
  setups. A VPS without KVM support cannot run this provider.
- **Podman**, **QEMU**, **Container** — additional provider kinds.
  Docker and Firecracker are the primary targets.

### Domain layout

| Domain pattern           | Target                  |
|--------------------------|-------------------------|
| `management.example.com` | Go Service (UI + API)   |

## 3. Communication

| Channel       | Path               | Purpose                                  |
|---------------|--------------------|------------------------------------------|
| REST API      | `/api/*`           | Space CRUD, auth, user mgmt              |
| WebSocket     | `/ws/*`            | Real-time space status                   |
| WebSocket     | `/ws/terminal/*`   | Bidirectional terminal I/O for terminal- |
|               |                    | based spaces                             |
| Static assets | `/`                | Embedded Angular SPA                     |

## 4. Space Lifecycle

Terminal spaces are created, used, and destroyed through the orchestrator,
which delegates environment management to the selected provider.

1. User creates a new space, selecting a provider from the available options.
2. The orchestrator resolves the provider and delegates environment creation.
3. The provider provisions the environment and returns connection info.
4. The service records the space's provider and endpoint in the database.
5. The user accesses the space via the terminal UI.
6. On deletion, the orchestrator stops the space and cleans up the environment.

## 5. Configuration & Deployment

- A single Go binary is all that is needed to run the management service
- A reverse proxy (e.g., Caddy, nginx) handles TLS termination for the
  management domain
- Providers are registered at startup — availability checks determine which
  are usable
- SQLite database file lives alongside the binary
