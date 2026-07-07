# Authentication Reference

For a high-level overview of the authentication architecture, see
[architecture/authentication.md](../architecture/authentication.md).

## Go Service Auth Endpoints

| Endpoint           | Method | Purpose                           |
|--------------------|--------|-----------------------------------|
| `/api/auth/login`  | POST   | Validate credentials, issue cookie|
| `/api/auth/logout` | POST   | Invalidate session                |

### CSRF Protection

The `POST` endpoints should include CSRF tokens (e.g., via a cookie-to-header
pattern) to prevent cross-site request forgery.

## Session Store

Sessions are stored in SQLite. Each session record contains:

- A hash of the session token
- The associated user ID
- IP address and user agent from the login request
- Timestamps for creation, last use, and last update
- An expiration time

Key behaviors:

- `LastUsedAt` is only written if the current time exceeds the stored value
  by more than 5 minutes, capping write frequency per session.
- Expired sessions are ignored during validation.
- A periodic cleanup job removes expired rows.

## Security Considerations

- **Cookie theft**: The cookie is scoped to the management domain and is not
  accessible from other domains. The Go service consumes the cookie during the
  WebSocket handshake and never forwards it to the space.
- **HttpOnly**: The cookie is not accessible from JavaScript.
- **HTTPS only**: The `Secure` flag ensures the cookie is never sent over
  plain HTTP.
- **SameSite=Lax**: Prevents the cookie from being sent on cross-site requests.
- **Space compromise**: If a space is compromised, the attacker cannot obtain
  the session cookie. The cookie is consumed by the Go service and never reaches
  the space.
