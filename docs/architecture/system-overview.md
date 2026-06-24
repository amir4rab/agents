# System Overview

## 1. Introduction

This project is a self-hosted web service that manages and exposes software agents
to the internet. It is designed for small-scale use — at most a handful of users
(friends, family).

**Key design choices:**

- Single Go binary (management service + embedded frontend + SQLite)
- Traefik as the reverse proxy for routing and TLS
- Agents run in isolated environments (Docker containers or Firecracker microVMs)
- No external dependencies beyond Traefik and a container runtime

## 2. Architecture

All traffic flows through Traefik, which terminates TLS and routes requests to the
appropriate backend:

- The **management domain** (`management.example.com`) targets the Go Management
  Service, which serves the Angular SPA, REST API, and WebSocket endpoints.
- Each **agent subdomain** (`<agent-id>.example.com`) targets the individual
  container or microVM hosting that agent.

The Go Management Service is a single binary composed of three logical layers:

- **HTTP Server** — serves the embedded Angular frontend (`/`), the REST API
  (`/api/*`), and WebSocket connections (`/ws/*`).
- **Agent Orchestrator** — manages agent lifecycles through a provider
  abstraction (Docker or Firecracker) and configures Traefik routes.
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
| `<agent-id>.example.com` | Individual agent        |

## 3. Communication

| Channel       | Path         | Purpose                        |
|---------------|--------------|--------------------------------|
| REST API      | `/api/*`     | Agent CRUD, auth, user mgmt    |
| WebSocket     | `/ws/*`      | Real-time agent logs, status   |
| Static assets | `/`          | Embedded Angular SPA           |

## 4. Agent lifecycle

1. User requests a new agent via the UI or REST API
2. The orchestrator selects a provider (Docker or Firecracker)
3. The agent is started inside its isolated environment
4. The orchestrator registers a subdomain route in Traefik
5. The user receives the agent URL and can access it
6. On deletion, the orchestrator stops the agent and removes the Traefik route

## 5. Configuration & deployment

- A single Go binary is all that is needed beyond Traefik
- Provider choice (docker / firecracker) is set at startup
- SQLite database file lives alongside the binary
- Traefik handles TLS via Let's Encrypt automatically
