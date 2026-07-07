# Terminal Space Reference

For a high-level overview of the terminal space architecture, see
[architecture/terminal-agents.md](../architecture/terminal-agents.md).

## WebSocket Protocol

The external WebSocket connection uses a simple framing protocol. Each frame
starts with a 1-byte type tag followed by a payload. Each connection represents
exactly one session — session routing is determined by the WebSocket endpoint
path, not the frame payload.

| Frame       | Payload        | Direction      | Description                      |
|-------------|----------------|----------------|----------------------------------|
| `output`    | Raw bytes      | Space -> Client| Terminal output (stdout/stderr)  |
| `input`     | Raw bytes      | Client -> Space| User input (stdin)               |
| `resize`    | JSON           | Client -> Space| Terminal resize (cols, rows)     |
| `close`     | Empty          | Either         | Close this session only          |
| `error`     | JSON           | Either         | Error message                    |
| `session_created` | JSON    | Server -> Client| Sent on new session creation     |
| `name_session`    | JSON    | Client -> Server| Assign a user-facing name        |

### Data flow — new session

1. Client connects to `/ws/terminal/<space-id>` — Go service authenticates via
   session cookie during the HTTP upgrade handshake.
2. Go service checks the space's active session count against its cap. If at
   capacity, the upgrade is rejected.
3. Go service generates a unique session ID, calls the provider to create a new
   shell process inside the space's environment, and registers the session in
   its in-memory session manager.
4. Go service sends a `session_created` frame to the client with the session ID
   and a default name.
5. Bidirectional streaming begins. Space output is tagged as `output` frames,
   user input as `input` frames.
6. On terminal resize, the client sends a `resize` frame. The dimensions are
   forwarded to the session's PTY.
7. Either side can send `close` to end **this** session only. Other sessions
   for the same space continue unaffected.

### Data flow — reconnect

1. Client connects to `/ws/terminal/<space-id>/<session-id>`.
2. Go service authenticates and looks up the session by ID in its in-memory
   session manager.
3. If the session is still active, the Go service sends the last N lines of
   buffered output to catch the client up, then resumes streaming.
4. If the session has expired, the Go service sends an `error` frame and closes
   the connection.

## Frontend Integration

The Angular frontend uses a tabbed terminal emulator interface, allowing users
to manage multiple concurrent sessions per space.

### Component structure

```
SpaceTerminalPage
+-- TabBar
|   +-- Tab (label, close button, active indicator)
|   +-- Tab (label, close button)
|   +-- [...]
|   +-- [+] button (create new session)
+-- TerminalComponent (active tab only)
```

Each `TerminalComponent` instance owns one WebSocket connection and one
terminal emulator instance:

```
TerminalComponent (per-tab)
+-- Terminal instance (terminal emulator library)
+-- Auto-resize handler
+-- WebSocket connection (one session)
+-- Session ID (from session_created frame)
+-- Name input (optional user-assigned name)
+-- Input/Output binding
```

### Tab management

- **Tab bar** is displayed at the top of the space terminal page, listing all
  active sessions for this space.
- **Tab label** shows the user-assigned name if set, otherwise the default
  name from the `session_created` frame.
- **[+] button** opens a new WebSocket to `/ws/terminal/<space-id>`, creating
  a new session. The new tab is automatically activated.
- **[x] button** on a tab sends a `close` frame for that session, then removes
  the tab.
- **Active tab** indicator highlights the currently visible terminal. Only the
  active tab's `TerminalComponent` is mounted in the DOM.

### TerminalComponent lifecycle

1. **Mount** — creates a terminal emulator instance and initiates the WebSocket
   connection.
2. **Connect** — opens a WebSocket to the terminal endpoint. On receiving the
   `session_created` frame, stores the session ID for reconnection and sends
   an initial `resize` frame.
3. **Render** — incoming `output` frames are written to the terminal. User
   keystrokes are sent as `input` frames.
4. **Resize** — on container resize, sends a `resize` frame with the new
   dimensions.
5. **Reconnect** — if the WebSocket drops, attempts to reconnect with
   exponential backoff (1s, 2s, 4s, 8s, max 30s). The scrollback buffer is
   preserved in memory.
6. **Destroy** — on tab close, sends a `close` frame and disposes the terminal
   emulator.

### Copy & Paste

- **Copy**: Terminal text selection is handled natively. Selected text can be
  copied via `Ctrl+Shift+C` or the browser's context menu.
- **Paste**: `Ctrl+Shift+V` pastes clipboard content into the terminal input
  stream.

## Session Buffering

### Per-session ring buffer

The Go service maintains an in-memory ring buffer (configurable size, default
1000 lines) for **each active terminal session**. This enables:

- Reconnecting clients to catch up on missed output.
- The admin panel to view a live or recent terminal session without connecting
  directly.

Each session gets its own ring buffer. Buffers are isolated — output from one
session is never visible in another.

### SessionManager

The Go service uses an in-memory session manager to track all active sessions.
Sessions are organized by space and by session ID.

### Session cleanup

- A session is removed when its process has exited and no WebSocket clients are
  connected to it.
- If the process exited while clients are still connected, the session remains
  in memory to serve the "Session ended" state.
- The session manager is periodically scanned for stale sessions.
- Session data is **not persisted to disk**. On Go service restart, all
  sessions are lost and clients must create new sessions.

### Visibility

- **Admin** — can list all active sessions across all spaces.
- **Space owner** — can list their own active sessions for their spaces.
