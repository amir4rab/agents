# Terminal Space Architecture

## 1. Overview

Terminal-based spaces (called **Spaces**) are environments that expose a
command-line interface, REPL, or terminal UI (TUI). Terminal spaces stream
their I/O through the Go Management Service over WebSocket — the client never
connects to the space directly.

Each space environment supports **multiple concurrent terminal sessions**,
each with its own independent shell process, PTY, and ring buffer. Sessions
are created dynamically at connection time — users open and close tabs like
a desktop terminal emulator.

## 2. Architecture

Terminal streaming is a two-hop relay with per-session isolation. The client
never communicates directly with the space — all data passes through the Go
service.

**Hop 1 (external):** The client opens a WebSocket to the Go service. Each
WebSocket connection represents one terminal session. The external endpoint
creates a new session on first connect and can reconnect to an existing session
by including the session ID in the path:

| Connection type | Endpoint |
|---|---|
| Create new session | `wss://management.example.com/ws/terminal/<space-id>` |
| Reconnect to session | `wss://management.example.com/ws/terminal/<space-id>/<session-id>` |

Each Hop 1 connection is authenticated via the management session cookie sent
during the WebSocket handshake and carries terminal output (space -> client)
and user input (client -> space).

**Hop 2 (internal):** For each session, the Go service calls the provider to
spawn a new process with its own PTY inside the space's environment. The Go
service relays bytes between the Hop 1 WebSocket and the session handle
bidirectionally. The Go service has no knowledge of how the provider creates
the session — it may exec into a container, connect to an internal WebSocket
server, or use any other mechanism.

Each session is fully isolated from others — they share the same filesystem
and network namespace but have independent shell processes, environment
variables, and working directories.

### External WebSocket (Client -> Go Service)

- Create: `wss://management.example.com/ws/terminal/<space-id>`
- Reconnect: `wss://management.example.com/ws/terminal/<space-id>/<session-id>`
- Authenticated via the management session cookie sent during the WebSocket
  handshake
- Protocol: typed frames — output, input, resize, close, error, session info

### Internal Connection (Go Service -> Space) — Provider abstraction

The Go service does not connect to spaces directly. Instead, it delegates
session creation to the space's registered provider:

1. The Go service resolves the provider from the registry using the provider
   kind stored in the space's database record.
2. The Go service calls the provider to start a session.
3. The provider returns a session handle with stdin, stdout, resize, and close
   methods.
4. The Go service relays bytes bidirectionally between the Hop 1 WebSocket
   and the session handle.

The provider implementation determines how the session is created:

- **Docker provider** — uses `docker exec` to spawn a new shell process with a
  PTY inside the container.
- **Firecracker provider** — spawns a new PTY inside the microVM, or connects
  to an internal WebSocket server running inside the VM.
- **Future providers** — may use SSH, LXC exec, or any other mechanism.

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

- **Admin** — can connect to any space's terminal stream.
- **Normal user** — can only connect to their own spaces' terminal streams.

### Session cap enforcement

Before creating a new session, the Go service checks the space's active session
count against its configured maximum:

1. Look up the space's session cap.
2. Count currently active sessions from the in-memory session manager.
3. If the count has reached the cap, the WebSocket upgrade is rejected with
   a 429 status.
4. On session close, the session is removed from the active count.

Session caps are not enforced during reconnection — only during initial session
creation.
