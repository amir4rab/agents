# Authentication Architecture

## 1. Overview

Authentication is handled directly by the Go Management Service. The session
cookie issued at login authenticates all subsequent requests to the management
domain, including WebSocket connections. No external auth proxy is involved.

**How it works:**

- Users log in on the management domain and receive a session cookie scoped
  to the management domain.
- The Go service validates the session cookie on every request — API calls,
  static asset loads, and WebSocket upgrade handshakes.
- For terminal spaces, the Go service validates the session during the
  WebSocket handshake, then opens its own internal connection to the space.
  The space never sees the session cookie.

## 2. Login Flow

1. User sends `POST /api/auth/login` with email and password.
2. Go service verifies the credentials against SQLite.
3. If valid, a session record is created in SQLite with a random token and
   an expiration time of 24 hours.
4. Go service responds with `Set-Cookie: session=<token>`:
   - `Path=/`
   - `Domain=management.example.com`
   - `HttpOnly`, `Secure`, `SameSite=Lax`
   - `Max-Age=86400` (24 hours)

## 3. Session Model

Sessions are stored in SQLite and contain the token hash, user ID, IP address,
user agent, and timestamps for creation, last use, and update.

Key behaviors:
- `LastUsedAt` is only written if the current time exceeds the stored value
  by more than 5 minutes, capping write frequency per session.
- Expired sessions are ignored during validation.
- A periodic cleanup removes expired rows.

## 4. Space Access Flow

Terminal-based spaces are accessed through the management domain. The client
connects to `wss://management.example.com/ws/terminal/<space-id>`, which is
served directly by the Go Management Service.

### Authentication

1. The browser sends the `Cookie: session=<token>` header during the WebSocket
   upgrade handshake.
2. The Go service validates the session token against SQLite.
3. If valid, the upgrade proceeds. The Go service then opens its own internal
   connection to the space's terminal interface.
4. If invalid, the upgrade is rejected with a 401.

### Authorization

Terminal access follows the management access model:

- **Admin** — can connect to any space's terminal stream.
- **Normal user** — can only connect to their own spaces' terminal streams.

This is enforced by the Go service when the client requests the terminal
endpoint. The service looks up the space, checks the user's role, and either
allows or rejects the connection.
