# Authentication Architecture

## 1. Overview

Authentication is handled directly by the Go Management Service. The session
cookie issued at login authenticates all subsequent requests to the management
domain, including WebSocket connections to `/ws/terminal/*`. No external auth
proxy is involved.

**How it works:**

- Users log in on the management domain and receive a session cookie scoped
  to `management.example.com`.
- The Go service validates the session cookie directly on every request — API
  calls, static asset loads, and WebSocket upgrade handshakes.
- For terminal agents, the Go service validates the session during the
  WebSocket handshake, then opens its own internal connection to the agent.
  The agent never sees the session cookie.

## 2. Login flow

1. User sends `POST /api/auth/login` with email and password.
2. Go service verifies the credentials against SQLite.
3. If valid, a session record is created in SQLite with a random token and
   an expiration time of 24 hours.
4. Go service responds with `Set-Cookie: session=<token>`:
   - `Path=/`
   - `Domain=management.example.com`
   - `HttpOnly`, `Secure`, `SameSite=Lax`
   - `Max-Age=86400` (24 hours)

On subsequent requests, the browser sends the cookie to
`management.example.com`. The Go service validates it directly.

## 3. Agent access flow

Terminal-based agents are accessed through the management domain. The client
connects to `wss://management.example.com/ws/terminal/<agent-id>`, which is
served directly by the Go Management Service.

### Authentication

1. The browser sends the `Cookie: session=<token>` header during the WebSocket
   upgrade handshake to `management.example.com`.
2. The Go service validates the session token against SQLite.
3. If valid, the upgrade proceeds. The Go service then opens its own internal
   connection to the agent's terminal interface.
4. If invalid, the upgrade is rejected with a 401.

### Authorization

Terminal access follows the management access model:

- **Admin** — can connect to any agent's terminal stream.
- **Normal user** — can only connect to their own agents' terminal streams.

This is enforced by the Go service when the client requests
`/ws/terminal/<agent-id>`. The service looks up the agent, checks the user's
role, and either allows or rejects the connection.

## 4. Go service auth endpoints

| Endpoint                     | Method | Purpose                           |
|------------------------------|--------|-----------------------------------|
| `/api/auth/login`            | POST   | Validate credentials, issue cookie|
| `/api/auth/logout`           | POST   | Invalidate session                |

### CSRF protection

The `POST` endpoints (`/api/auth/login`, `/api/auth/logout`) should include
CSRF tokens (e.g., via a cookie-to-header pattern or a hidden form field in
the SPA) to prevent cross-site request forgery.

## 5. Session store

The session object is defined by the following Go structure:

```go
type Session struct {
    ID         int64  // INTEGER PRIMARY KEY
    TokenHash  []byte // BLOB — SHA-256 of the raw token
    UserID     int64  // INTEGER — references users.id
    IPAddress  []byte // BLOB — raw IP bytes (net.IP)
    LastUsedIP []byte // BLOB — raw IP bytes
    UserAgent  string // TEXT
    LastUsedAt int64  // INTEGER — Unix ms, updated if stale by >5 min
    CreatedAt  int64  // INTEGER — Unix ms
    ExpiresAt  int64  // INTEGER — Unix ms
}
```

Key behaviors:

- `LastUsedAt` is only written if the current time exceeds the stored value
  by more than 5 minutes, capping write frequency per session. `LastUsedIP`
  is updated alongside it when the client IP has changed.
- Expired sessions are ignored during validation.
- A periodic cleanup job removes expired rows.

## 6. Security considerations

- **Cookie theft**: The cookie is scoped to `management.example.com` and is not
  accessible from other domains. The Go service consumes the cookie during the
  WebSocket handshake and never forwards it to the agent.
- **HttpOnly**: The cookie is not accessible from JavaScript.
- **HTTPS only**: The `Secure` flag ensures the cookie is never sent over
  plain HTTP.
- **SameSite=Lax**: Prevents the cookie from being sent on cross-site
  requests.
- **Agent compromise**: If an agent is compromised, the attacker cannot
  obtain the session cookie. The cookie is consumed by the Go service and never
  reaches the agent. The agent only sees the user ID, which is low-value
  information.
