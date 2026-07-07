# Database Architecture

## 1. Overview

The application uses an embedded SQLite database. The SQLite file lives
alongside the binary and is managed entirely by the Go process.

- Single binary deployment, no external database server
- No network round-trips — queries are in-process
- Sufficient for single-digit users and dozens of spaces
- Stores users, sessions, space configurations, and related metadata

## 2. SQLite Configuration

The following pragmas are applied on every connection open:

| Pragma           | Value    | Purpose                                                  |
|------------------|----------|----------------------------------------------------------|
| `journal_mode`   | `WAL`    | Write-ahead logging for concurrent reads during writes   |
| `synchronous`    | `NORMAL` | Durable with WAL; faster than `FULL` with minimal risk   |
| `foreign_keys`   | `ON`     | Enforce FK constraints at the SQLite level               |
| `busy_timeout`   | `5000`   | Wait up to 5 seconds on locks instead of returning busy  |

SQLite's WAL mode supports concurrent reads — multiple goroutines can read
simultaneously while a single writer is active.

## 3. Schema Design Principles

- **Strict tables** — every table uses the `STRICT` option to enforce declared
  column types.
- **Primary keys** — `INTEGER PRIMARY KEY` (int64) on every table. No
  surrogate keys, no UUIDs, no composite PKs unless unavoidable.
- **Timestamps** — `INTEGER` storing Unix milliseconds, never text. The
  application generates values via `time.Now().UnixMilli()`.
- **Byte arrays** — `BLOB` for binary data (token hashes, IP addresses).
  Raw bytes, never hex or base64 encoded.
- **Text** — only for actual string content: usernames, display names, etc.
- **Enums** — status and categorical fields stored as `INTEGER` or mapped in
  application code.

### Spaces table

The spaces table includes a provider column that stores the kind of provider
used to create the space (e.g., Docker, Firecracker). This field is set at
space creation and is immutable.

## 4. Cursor-Based Pagination

All list endpoints use cursor-based pagination for stable, efficient
navigation:

- Consistent performance — always starts at an indexed position, no row
  scanning
- Stable under writes — no skipped or duplicated items between pages
- Minimal overhead — PK index provides direct access

The cursor is the `id` of the last item returned. Results include metadata
(start cursor, end cursor, next/previous page presence) so clients know
when pagination is exhausted.

The tradeoff is no arbitrary page jumping, which is acceptable for "load more"
UI patterns.

## 5. Data Safety

**Passwords** — hashed with Argon2id. Never logged or stored in plaintext.

**Session tokens** — random bytes from a CSPRNG, hashed before storage.
The raw token is returned to the client in a cookie. On lookup, hash the
incoming token and compare against the stored hash. A database leak never
exposes usable tokens.

**IP addresses** — stored as raw bytes: IPv4 as 4-byte, IPv6 as 16-byte.

**Write durability** — `synchronous = NORMAL` with `journal_mode = WAL` ensures
committed writes survive power loss or OS crash.

**Session cleanup** — a periodic job deletes expired rows.
