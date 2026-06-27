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
- Agents run in isolated environments (Docker containers or Firecracker microVMs)
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
- **Agent Orchestrator** — manages agent lifecycles through a provider
  abstraction (Docker or Firecracker) and registers terminal streaming
  endpoints when agents are created.
- **SQLite** — embedded database storing users, agents, and sessions.

Agents run in isolation using one of two providers:

- **Docker** — each agent is a container; suitable for most environments.
- **Firecracker** — each agent is a microVM; stronger isolation for advanced
  setups. A VPS without KVM support cannot run this provider.

Multiple agents can run simultaneously under either provider.

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

### Terminal-based agents

1. User requests a new agent of type `terminal` via the UI or REST API
2. The orchestrator selects a provider (Docker or Firecracker)
3. The agent is started and exposes a terminal interface on its internal address
4. The Go service records the agent's terminal endpoint
5. The user is taken to the terminal UI in the Angular SPA, which shows the
   agent's terminal page with a tabbed interface. Each tab opens a new
   independent WebSocket session to `/ws/terminal/<agent-id>`. Multiple
   concurrent sessions per agent are supported.
6. On deletion, the orchestrator stops the agent and removes the terminal
   endpoint registration

## 5. Configuration & deployment

- A single Go binary is all that is needed to run the management service
- A reverse proxy (e.g., Caddy, nginx) handles TLS termination for the
  management domain
- Provider choice (docker / firecracker) is set at startup
- SQLite database file lives alongside the binary
