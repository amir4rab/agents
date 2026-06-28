# Authentication Reference

For a high-level overview of the authentication architecture, see
[architecture/authentication.md](../architecture/authentication.md).

## Go service auth endpoints

| Endpoint                     | Method | Purpose                           |
|------------------------------|--------|-----------------------------------|
| `/api/auth/login`            | POST   | Validate credentials, issue cookie|
| `/api/auth/logout`           | POST   | Invalidate session                |

### CSRF protection

The `POST` endpoints (`/api/auth/login`, `/api/auth/logout`) should include
CSRF tokens (e.g., via a cookie-to-header pattern or a hidden form field in
the SPA) to prevent cross-site request forgery.

## Session store

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

## Security considerations

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
