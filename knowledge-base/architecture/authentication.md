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

For detailed specifications (auth endpoints, session store structure, security
considerations), see [specs/authentication.md](specs/authentication.md).
