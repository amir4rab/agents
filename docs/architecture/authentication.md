# Authentication Architecture

## 1. Overview

Authentication is centralized in the Go Management Service. Agents themselves
never handle credentials — Traefik strips all auth tokens before forwarding
requests to them. This keeps agents simple and avoids duplication of login
logic across every agent.

**How it works:**

- Users log in on the management domain and receive a session cookie scoped
  to `.example.com` (shared across all subdomains).
- Traefik intercepts requests to agent subdomains, validates the session
  with the Go service, strips the cookie, and forwards the request with
  only an identity header.

## 2. Login flow

1. User sends `POST /api/auth/login` with email and password.
2. Go service verifies the credentials against SQLite.
3. If valid, a session record is created in SQLite with a random token and
   an expiration time of 24 hours.
4. Go service responds with `Set-Cookie: session=<token>`:
   - `Path=/`
   - `Domain=.example.com`
   - `HttpOnly`, `Secure`, `SameSite=Lax`
   - `Max-Age=86400` (24 hours)

On subsequent requests, the browser sends the cookie to all subdomains of
`.example.com`. The Go service validates it; Traefik strips it before it
reaches any agent.

## 3. Agent access flow

When a user navigates to an agent subdomain, Traefik applies a middleware
chain that validates the session and removes credentials before the request
reaches the agent.

1. Browser sends `GET <agent-id>.example.com/` with `Cookie: session=<token>`.
2. Traefik matches the request against the agent route, which has a
   middleware chain attached.
3. **ForwardAuth** middleware sends a request to
   `http://go-service:8080/api/auth/forward`, including the original Cookie
   header. The original request details are forwarded as `X-Forwarded-*`
   headers (`X-Forwarded-Method`, `X-Forwarded-Proto`, `X-Forwarded-Host`,
   `X-Forwarded-Uri`, `X-Forwarded-For`). By default the outward method is
   GET; if the Go service needs the original method, set
   `preserveRequestMethod: true` in the middleware.
4. Go service looks up the session token in SQLite.
5. If valid, Go service returns `200 OK` with `X-Auth-User-Id: <id>`.
   If invalid, it returns `401` with a redirect to
   `management.example.com/login?return=<original-url>` — Traefik passes
   this response to the browser, sending the user to the login page.
6. **Headers** middleware strips the `Cookie` header (set to empty string)
   and adds `X-Auth-User-Id` from the ForwardAuth response.
7. Traefik forwards the cleaned request to the agent. The agent receives
   only `X-Auth-User-Id` — no credentials.

### Middleware chain (applied per agent route)

```yaml
http:
  middlewares:
    agent-auth-pipeline:
      chain:
        middlewares:
          - forward-auth
          - strip-cookies

    forward-auth:
      forwardAuth:
        address: "http://go-service:8080/api/auth/forward"
        authResponseHeaders:
          - "X-Auth-User-Id"

    strip-cookies:
      headers:
        customRequestHeaders:
          Cookie: ""
```

### What the agent receives

After the middleware chain processes the request, the agent sees only:

```
GET /
X-Auth-User-Id: 42
```

The session cookie is gone. No credentials are forwarded to the agent.

### WebSocket connections

WebSocket handshake requests include cookies, so Traefik's ForwardAuth
middleware works the same way. Apply the same `agent-auth-pipeline` chain
to WebSocket routes. The browser sends the cookie during the initial HTTP
handshake, Traefik validates and strips it, and the agent receives only
the `X-Auth-User-Id` header.

## 4. Go service auth endpoints

| Endpoint                     | Method | Purpose                           |
|------------------------------|--------|-----------------------------------|
| `/api/auth/login`            | POST   | Validate credentials, issue cookie|
| `/api/auth/logout`           | POST   | Invalidate session                |
| `/api/auth/forward`          | GET    | Called by Traefik ForwardAuth     |

### `/api/auth/forward` response

- **200 OK** — session is valid. Returns `X-Auth-User-Id` header.
- **401 Unauthorized** — session is missing or invalid. Returns a redirect
  to `management.example.com/login?return=<original-url>`, sending the
  user to the login page with a return path so they can be redirected
  back after logging in.

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

- **Cookie theft**: The `.example.com` cookie is sent to all subdomains, but
  Traefik strips it before it reaches any agent. Only the management API
  and the ForwardAuth endpoint ever see the raw token.
- **HttpOnly**: The cookie is not accessible from JavaScript.
- **HTTPS only**: The `Secure` flag ensures the cookie is never sent over
  plain HTTP.
- **SameSite=Lax**: Prevents the cookie from being sent on cross-site
  requests.
- **Agent compromise**: If an agent is compromised, the attacker cannot
  obtain the session cookie — it is stripped by Traefik. The agent only
  sees the user ID, which is low-value information.
