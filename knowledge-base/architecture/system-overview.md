# System Overview

## 1. Introduction

This project is a self-hosted web service that manages terminal-based software
agents and provides browser-based access to their terminal interfaces via
WebSocket streaming. It is designed for small-scale use — at most a handful of
users (friends, family).

**Key design choices:**

- Single Go binary (management service + embedded frontend + SQLite)
- A reverse proxy (e.g., Caddy, nginx) handles TLS termination for the
  management domain
- Agents run in isolated environments managed by **providers** — pluggable
  implementations that abstract the underlying technology (Docker, Firecracker,
  etc.). See [providers.md](providers.md).
- The management service is **provider-agnostic** — it interacts with
  environments exclusively through the Provider interface.
- Providers are discovered at startup via health-check scan. Only available
  providers are presented to users for selection.
- WebSocket relay through the Go service is the primary agent access path —
  clients never connect to agents directly

## 2. Architecture

All HTTP and WebSocket traffic for the management domain flows through a
reverse proxy which terminates TLS:

- The **management domain** (`management.example.com`) targets the Go Management
  Service, which serves the Angular SPA, REST API, and WebSocket endpoints.
- **Terminal-based agents** are accessed through the management domain — their
  terminal I/O is streamed through the Go Management Service via WebSocket. See
  [terminal-agents.md](terminal-agents.md).

The Go Management Service is a single binary composed of three logical layers:

- **HTTP Server** — serves the embedded Angular frontend (`/`), the REST API
  (`/api/*`), and WebSocket connections (`/ws/*`).
- **Agent Orchestrator** — manages agent lifecycles through the
  [Provider interface](providers.md) and registers terminal streaming
  endpoints when agents are created. The orchestrator has no knowledge of
  the underlying environment technology.
- **Provider Registry** — an in-memory registry of all provider
  implementations, scanned at startup for availability. The orchestrator
  queries the registry to resolve provider instances by name.
- **SQLite** — embedded database storing users, agents, and sessions.

Agents run in isolated environments managed by **providers**:

- **Docker** (provider) — each agent is a container; suitable for most
  environments.
- **Firecracker** (provider) — each agent is a microVM; stronger isolation
  for advanced setups. A VPS without KVM support cannot run this provider.

Multiple agents can run simultaneously, potentially using different providers.
The provider is selected by the user at creation time from the list of
available providers. See [providers.md](providers.md) for details on the
provider interface, registry, and discovery.

### Domain layout

| Domain pattern           | Target                  |
|--------------------------|-------------------------|
| `management.example.com` | Go Service (UI + API)   |

## 3. Communication

| Channel       | Path               | Purpose                                  |
|---------------|--------------------|------------------------------------------|
| REST API      | `/api/*`           | Agent CRUD, auth, user mgmt              |
| WebSocket     | `/ws/*`            | Real-time agent logs, status             |
| WebSocket     | `/ws/terminal/*`   | Bidirectional terminal I/O for terminal- |
|               |                    | based agents                             |
| Static assets | `/`                | Embedded Angular SPA                     |

## 4. Agent lifecycle

Terminal agents are created, used, and destroyed through the orchestrator,
which delegates environment management to the selected provider.

For detailed lifecycle steps, including session creation, reconnection,
and agent termination, see [specs/system-overview.md](specs/system-overview.md).

## 5. Configuration & deployment

- A single Go binary is all that is needed to run the management service
- A reverse proxy (e.g., Caddy, nginx) handles TLS termination for the
  management domain
- Providers are auto-discovered at startup — no manual provider configuration
  is required. The registry scans all registered providers and reports their
  availability via the health-check API.
- SQLite database file lives alongside the binary
