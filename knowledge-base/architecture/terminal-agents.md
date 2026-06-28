# Terminal Agent Architecture

## 1. Overview

Terminal-based agents are agents that expose a command-line interface, REPL, or
terminal UI (TUI). Terminal agents stream their I/O through the Go Management
Service over WebSocket — the client never connects to the agent directly.

Each agent environment supports **multiple concurrent terminal sessions**,
each with its own independent shell process, PTY, and ring buffer. Sessions
are created dynamically at connection time — users open and close tabs like
a desktop terminal emulator.

## 2. Architecture

Terminal streaming is a two-hop relay with per-session isolation. The client
never communicates directly with the agent — all data passes through the Go
service.

**Hop 1 (external):** The client opens a WebSocket to the Go service. Each
WebSocket connection represents one terminal session. The external endpoint
creates a new session on first connect and can reconnect to an existing session
by including the session ID in the path:

| Connection type | Endpoint |
|---|---|
| Create new session | `wss://management.example.com/ws/terminal/<agent-id>` |
| Reconnect to session | `wss://management.example.com/ws/terminal/<agent-id>/<session-id>` |

Each Hop 1 connection is authenticated via the management session cookie sent
during the WebSocket handshake and carries terminal output (agent → client)
and user input (client → agent).

**Hop 2 (internal):** For each session, the Go service calls
`provider.StartSession()` which returns a `SessionHandle` (providing `Stdin`,
`Stdout`, `Resize`, `Close`). The provider is responsible for spawning an
independent process with its own PTY inside the agent's environment. The Go
service relays bytes between the Hop 1 WebSocket and the `SessionHandle`
bidirectionally. The Go service has no knowledge of how the provider creates
the session — it may exec into a container, connect to an internal WebSocket
server, or use any other mechanism.

Each session is fully isolated from others — they share the same filesystem
and network namespace but have independent shell processes, environment
variables, and working directories.

### External WebSocket (Client ↔ Go Service)

- Create: `wss://management.example.com/ws/terminal/<agent-id>`
- Reconnect: `wss://management.example.com/ws/terminal/<agent-id>/<session-id>`
- Authenticated via the management session cookie sent during the WebSocket
  handshake
- Carries terminal output (agent → client) and user input (client → agent)
- Protocol: JSON control frames interspersed with raw binary data

### Internal Connection (Go Service ↔ Agent) — Provider abstraction

The Go service does not connect to agents directly. Instead, it delegates
session creation to the agent's registered provider via `StartSession()`:

1. The Go service resolves the provider from the [Provider Registry](providers.md)
   using the provider name stored in the agent's database record.
2. The Go service calls `provider.StartSession(ctx, agentID, opts)`.
3. The provider returns a `SessionHandle` with `Stdin`, `Stdout`, `Resize`,
   and `Close` methods.
4. The Go service relays bytes bidirectionally between the Hop 1 WebSocket
   and the `SessionHandle`.

The provider implementation determines how the session is created:

- **Docker provider** — uses `docker exec` to spawn a new shell process with a
  PTY inside the container. The provider wraps the process's stdin/stdout in a
  `SessionHandle`.
- **Firecracker provider** — spawns a new PTY inside the microVM, or connects
  to an internal WebSocket server running inside the VM that creates a new PTY
  per connection.
- **Future providers** — may use SSH, LXC exec, or any other mechanism. As
  long as the `Provider` interface is satisfied, the Go service handles the
  session identically.

See [providers.md](providers.md) for the complete interface specification and
integration contract.

## 3. Authentication & Authorization

### Authentication

The terminal WebSocket endpoint lives on the management domain, so it uses the
same session cookie as the Angular SPA. During the HTTP upgrade handshake:

1. The browser sends the `Cookie: session=<token>` header with the WS upgrade
   request.
2. The Go service validates the session token against SQLite.
3. If valid, the upgrade proceeds and the session is associated with the
   WebSocket connection.
4. If invalid, the upgrade is rejected with a 401 status.

### Authorization

Terminal access follows the management access model:

- **Admin** — can connect to any agent's terminal stream.
- **Normal user** — can only connect to their own agents' terminal streams.

This is enforced by the Go service when the client requests
`/ws/terminal/<agent-id>`. The service looks up the agent, checks the user's
role, and either allows or rejects the connection.

### Session cap enforcement

Before creating a new session, the Go service checks the agent's active session
count against its configured maximum:

1. Look up the agent's session cap (per-agent override, or system-wide default).
2. Count currently active sessions for this agent from the in-memory
   `SessionManager`.
3. If the count has reached the cap, the WebSocket upgrade is rejected with
   a 429 status and an error message indicating the limit has been reached.
4. On session close, the session is removed from the active count, freeing
   capacity for new sessions.

Session caps are not enforced during reconnection — only during initial
session creation.

For detailed specifications (WebSocket protocol frame types, frontend
component structure, lifecycle flows, session buffering), see
[specs/terminal-agents.md](specs/terminal-agents.md).
