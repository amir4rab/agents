# Terminal Agent Architecture

## 1. Overview

Terminal-based agents are agents that expose a command-line interface, REPL, or
terminal UI (TUI). Terminal agents stream their I/O through the Go Management
Service over WebSocket — the client never connects to the agent directly.

Each agent environment supports **multiple concurrent terminal sessions**,
each with its own independent shell process, PTY, and ring buffer. Sessions
are created dynamically at connection time — users open and close tabs like
a desktop terminal emulator.

(Web-based agents accessed via a dedicated subdomain may be supported in the
future.)

## 2. Architecture

Terminal streaming is a two-hop relay with per-session isolation. The client
never communicates directly with the agent — all data passes through the Go
service.

**Hop 1 (external):** The client opens a WebSocket to the Go service. Each
WebSocket connection represents one terminal session. The external endpoint
creates a new session on first connect and can reconnect to an existing session
by including the session ID in the path:

| Connection type         | Endpoint                                     |
|-------------------------|----------------------------------------------|
| Create new session      | `wss://management.example.com/ws/terminal/<agent-id>` |
| Reconnect to session    | `wss://management.example.com/ws/terminal/<agent-id>/<session-id>` |

Each Hop 1 connection is authenticated via the management session cookie sent
during the WebSocket handshake and carries terminal output (agent → client)
and user input (client → agent).

**Hop 2 (internal):** For each session, the Go service spawns an independent
process inside the agent's environment (e.g., a new `docker exec` or a new PTY
in the VM). The Go service relays bytes between the Hop 1 WebSocket and the
session's process bidirectionally. Each session is fully isolated from others —
they share the same filesystem and network namespace but have independent
shell processes, environment variables, and working directories.

### External WebSocket (Client ↔ Go Service)

- Create: `wss://management.example.com/ws/terminal/<agent-id>`
- Reconnect: `wss://management.example.com/ws/terminal/<agent-id>/<session-id>`
- Authenticated via the management session cookie sent during the WebSocket
  handshake
- Carries terminal output (agent → client) and user input (client → agent)
- Protocol: JSON control frames interspersed with raw binary data

### Internal Connection (Go Service ↔ Agent)

For each session, the Go service connects to the agent and spawns a new
process:

- **PTY exec mode**: The Go service uses the provider to exec into the agent
  and attach to a new PTY. Each session gets its own `docker exec` (Docker)
  or equivalent (Firecracker).
- **Internal WebSocket mode**: The Go service connects as a WebSocket client
  to the agent's internal WS server. The agent spawns a new PTY for each
  incoming connection.

The Go service relays bytes bidirectionally between each Hop 1 connection and
its corresponding Hop 2 process.

## 3. WebSocket Protocol

The external WebSocket connection uses a simple framing protocol. Each frame
starts with a 1-byte type tag followed by a payload. Each connection represents
exactly one session — session routing is determined by the WebSocket endpoint
path, not the frame payload.

| Type tag | Name      | Payload        | Direction      | Description                      |
|----------|-----------|----------------|----------------|----------------------------------|
| `0x01`   | `output`  | Raw bytes      | Agent → Client | Terminal output (stdout/stderr)  |
| `0x02`   | `input`   | Raw bytes      | Client → Agent | User input (stdin)               |
| `0x03`   | `resize`  | JSON           | Client → Agent | Terminal resize (cols, rows)     |
| `0x04`   | `close`   | Empty          | Either         | Close this session only          |
| `0x05`   | `error`   | JSON           | Either         | Error message                    |
| `0x06`   | `session_created` | JSON   | Server → Client | Sent on new session creation     |
| `0x07`   | `name_session` | JSON      | Client → Server | Assign a user-facing name        |

### Resize frame payload

```json
{
  "cols": 80,
  "rows": 24
}
```

### Error frame payload

```json
{
  "code": "SESSION_NOT_FOUND",
  "message": "The terminal session has expired."
}
```

### session_created frame payload

Sent from the server immediately after a new session is created on first
connect. The client uses the `session_id` for reconnection and the
`default_name` for the tab label.

```json
{
  "session_id": "a1b2c3d4",
  "default_name": "Session 1"
}
```

### name_session frame payload

Sent from the client to assign a user-friendly name to the session. The name
is displayed in the tab bar and returned in admin views.

```json
{
  "name": "Build server"
}
```

### Data flow — new session

1. Client connects to `/ws/terminal/<agent-id>` — Go service authenticates via
   session cookie during the HTTP upgrade handshake.
2. Go service checks the agent's active session count against its cap. If at
   capacity, the upgrade is rejected with an error.
3. Go service generates a unique `session_id`, spawns a new shell process
   inside the agent (e.g., `docker exec`), and registers the session in its
   in-memory `SessionManager`.
4. Go service sends a `session_created` frame to the client with the
   `session_id` and a `default_name`.
5. Bidirectional streaming begins. Agent output is tagged as `output` frames,
   user input as `input` frames.
6. On terminal resize, the client sends a `resize` frame. The Go service
   forwards it to the agent so the PTY dimensions are updated.
7. Either side can send `close` to end **this** session only. Other sessions
   for the same agent continue unaffected.

### Data flow — reconnect

1. Client connects to `/ws/terminal/<agent-id>/<session-id>`.
2. Go service authenticates and looks up the session by ID in its in-memory
   `SessionManager`.
3. If the session is still active (process running), the Go service sends the
   last N lines of buffered output from the session's ring buffer to catch the
   client up, then resumes streaming.
4. If the session has expired (process exited, no clients connected), the Go
   service sends an `error` frame with code `SESSION_NOT_FOUND` and closes the
   connection.

## 4. Authentication & Authorization

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

## 5. Agent Requirements

For an agent to support terminal streaming, it must:

- Run a process that reads from stdin and writes to stdout/stderr (a PTY is
  recommended for proper terminal emulation support).
- Support **multiple concurrent PTY connections** — the agent must allow
  multiple independent processes to run simultaneously, each with its own PTY.
- Accept connections from the Go service on a known internal address/port
  (configured at agent creation time).
- Handle SIGWINCH (or equivalent resize signal) when the terminal dimensions
  change.
- Optionally expose a health-check endpoint on the same internal address so
  the Go service can verify the terminal interface is ready.

### Agent terminal interface

The Go service supports two modes for connecting to the agent's terminal:

#### PTY exec mode

The Go service uses the provider (Docker/Firecracker) to exec into the agent
and attach to a new PTY for **each session**:

- **Docker**: `docker exec -it <container> <shell>` — the Go service runs a
  separate `docker exec` for each session, creating an independent shell
  process inside the container.
- **Firecracker**: The agent's init process must expose a mechanism for
  spawning new PTY sessions (e.g., an internal exec endpoint), or the VM
  runs an internal WebSocket server that creates a new PTY per connection
  (see Internal WebSocket mode).

Multiple concurrent `docker exec` invocations are supported natively by the
Docker daemon and provide natural per-session isolation.

#### Internal WebSocket mode

The agent runs a small WebSocket server internally. The Go service connects to
it as a client, creating one connection per session:

```
Go Service ──WS──► Agent (:9090/terminal)
                    ├── PTY session 1 (shell)
                    ├── PTY session 2 (shell)
                    └── PTY session 3 (shell)
```

The agent must accept **multiple concurrent WebSocket connections**, each
spawning its own PTY. The agent manages each PTY independently and translates
between WebSocket frames (using the same protocol described in section 3) and
each PTY's file descriptors.

This mode gives the agent full control over terminal setup (environment,
shell, resize handling) and is the recommended approach, especially for
Firecracker-based agents where `docker exec` is not available.

## 6. Frontend Integration

The Angular frontend uses a tabbed terminal emulator interface, allowing users
to manage multiple concurrent sessions per agent.

### Component structure

```
AgentTerminalPage
├── TabBar
│   ├── Tab (label, close button, active indicator)
│   ├── Tab (label, close button)
│   ├── [...]
│   └── [+] button (create new session)
└── TerminalComponent (active tab only)
```

Each `TerminalComponent` instance owns one WebSocket connection and one
terminal emulator instance:

```
TerminalComponent (per-tab)
├── Terminal instance (terminal emulator library)
├── Auto-resize handler
├── WebSocket connection (one session)
├── Session ID (from session_created frame)
├── Name input (optional user-assigned name)
└── Input/Output binding
```

### Tab management

- **Tab bar** is displayed at the top of the agent terminal page, listing all
  active sessions for this agent.
- **Tab label** shows the user-assigned name if set, otherwise the
  `default_name` from the `session_created` frame.
- **[+] button** opens a new WebSocket to `/ws/terminal/<agent-id>`, creating
  a new session. The new tab is automatically activated.
- **[×] button** on a tab sends a `close` frame for that session, then removes
  the tab. The underlying shell process in the agent is terminated.
- **Active tab** indicator highlights the currently visible terminal. Only the
  active tab's `TerminalComponent` is mounted in the DOM. Inactive tabs remain
  connected in the background but are not rendered (the WebSocket stays open).

### TerminalComponent lifecycle

1. **Mount** — `TerminalComponent` creates a terminal emulator instance, opens
   it in the DOM, and initiates the WebSocket connection.
2. **Connect** — The component opens a WebSocket to
   `wss://management.example.com/ws/terminal/<agent-id>`. On receiving the
   `session_created` frame, it stores the `session_id` for reconnection. It
   sends an initial `resize` frame with the current terminal dimensions.
3. **Render** — Incoming `output` frames are written to the terminal. User
   keystrokes are captured via the terminal emulator's input event and sent as
   `input` frames.
4. **Resize** — On container resize, the component recalculates dimensions and
   sends a `resize` frame with the new cols/rows.
5. **Session naming** — The user can optionally name the session by clicking
   the tab label and entering a name. The component sends a `name_session`
   frame with the entered name.
6. **Reconnect** — If the WebSocket drops, the component attempts to reconnect
   to `/ws/terminal/<agent-id>/<session-id>` with exponential backoff (1s, 2s,
   4s, 8s, max 30s). The scrollback buffer is preserved in memory.
7. **Destroy** — On tab close or component destroy, the component sends a
   `close` frame and disposes the terminal emulator.

### Copy & Paste

- **Copy**: Terminal text selection is handled natively. Selected text can be
  copied via `Ctrl+Shift+C` or the browser's context menu.
- **Paste**: `Ctrl+Shift+V` pastes clipboard content into the terminal input
  stream.

## 7. Lifecycle

### Creating a terminal agent

1. User creates a new agent with `type: terminal` via the UI or API.
2. The orchestrator starts the agent in its isolated environment (Docker/Firecracker).
3. The agent's terminal interface becomes available on its internal address.
4. The Go service records the agent's terminal endpoint in the database.
5. The user is taken to the terminal UI, which shows the tabbed interface
   with an initial session tab.

### Creating a new session

1. User clicks the **[+] button** in the tab bar, or the agent terminal page
   auto-creates the first session on load.
2. A new `TerminalComponent` mounts and opens a WebSocket to
   `/ws/terminal/<agent-id>`.
3. The Go service authenticates the session cookie, authorizes the user
   (owner or admin), and checks the agent's session cap.
4. If the cap is reached, the upgrade is rejected with a 429 status. The user
   sees an error message indicating the limit has been reached.
5. If allowed, the Go service generates a unique `session_id`, spawns a new
   shell process inside the agent (e.g., `docker exec`), and sends a
   `session_created` frame back to the client.
6. A new tab appears in the tab bar with the `default_name`. The tab becomes
   active and bidirectional streaming begins.

### Connecting to an existing session (reconnect)

1. The `TerminalComponent` opens a WebSocket to
   `/ws/terminal/<agent-id>/<session-id>`.
2. The Go service authenticates and looks up the session in its in-memory
   `SessionManager`.
3. If the session is active, the Go service sends the last N lines of buffered
   output to catch the client up, then resumes streaming.
4. If the session has expired, the Go service sends an `error` frame
   (`SESSION_NOT_FOUND`) and closes the connection. The component removes the
   tab and shows a "Session expired" message.

### Reconnecting after WebSocket drop

If the WebSocket drops (network blip, browser tab backgrounded):

1. The terminal's scrollback buffer is preserved in memory.
2. The component attempts to reconnect with exponential backoff (1s, 2s, 4s,
   8s, max 30s) to `/ws/terminal/<agent-id>/<session-id>`.
3. On reconnect, the Go service sends the last N lines of buffered output to
   catch the client up.
4. The terminal display resumes seamlessly.

### Closing a session

- User clicks the **[×] button** on a tab — the `TerminalComponent` sends a
  `close` frame.
- The Go service terminates the session's shell process (sends SIGTERM or
  equivalent) and removes the session from the in-memory `SessionManager`.
- The internal connection to the agent is cleaned up.
- The external WebSocket is closed cleanly.
- Other sessions for the same agent continue unaffected.
- If the shell process exits on its own (e.g., the user types `exit`), the
  Go service sends a `close` frame to the client and cleans up. The tab
  shows a "Session ended" state.

### Agent termination

When the agent is deleted or stopped:

1. The orchestrator stops the agent environment (container or microVM).
2. The Go service terminates all active sessions for that agent.
3. All connected WebSocket clients receive a `close` frame and the connection
   is dropped.
4. The in-memory `SessionManager` removes all sessions for the agent.

## 8. Session buffering

### Per-session ring buffer

The Go service maintains an in-memory ring buffer (configurable size, default
1000 lines) for **each active terminal session**. This enables:

- Reconnecting clients to catch up on missed output.
- The admin panel to view a live or recent terminal session without connecting
  directly (read-only snapshot).

Each session gets its own ring buffer. Buffers are isolated — output from one
session is never visible in another.

### SessionManager

The Go service uses an in-memory `SessionManager` to track all active sessions:

```go
type Session struct {
    ID         string       // unique session identifier
    AgentID    int64        // owning agent
    UserID     int64        // owning user (who created the session)
    Name       string       // user-assigned name, optional
    CreatedAt  int64        // Unix ms
    RingBuffer *RingBuffer  // per-session ring buffer
    Process    interface{}  // provider-specific process handle
}

type SessionManager struct {
    mu           sync.Mutex
    sessions     map[int64]map[string]*Session // agentID -> sessionID -> Session
    maxSessions  int                         // per-agent cap, configurable
}
```

### Session cleanup

- A session is removed from the `SessionManager` when its shell process exits
  **and** no WebSocket clients are connected to it.
- If the process exits while clients are still connected, the session remains
  in memory to serve the "Session ended" state and allow clients to read the
  final buffered output. The session is removed once all clients disconnect.
- The `SessionManager` is periodically scanned for stale sessions (e.g.,
  processes that exited more than 5 minutes ago with no clients) and removes
  them.
- Session data is **not persisted to disk**. On Go service restart, all
  sessions are lost and clients must create new sessions.

### Visibility

- **Admin** — can list all active sessions across all agents, including
  session IDs, assigned names, owner, and creation time.
- **Agent owner** — can list their own active sessions for their agents.
- **Agent runner** — sessions are opaque to the agent; the agent sees
  independent processes and does not know about session IDs.
