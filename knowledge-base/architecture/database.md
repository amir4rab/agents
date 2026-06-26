# Database Architecture

## 1. Overview

The application uses an embedded SQLite database. The SQLite file lives
alongside the binary and is managed entirely by the Go process.

- Single binary deployment, no external database server
- No network round-trips — queries are in-process
- Sufficient for single-digit users and dozens of agents
- Stores users, sessions, agent configurations, and related metadata

## 2. SQLite configuration

The following pragmas are applied on every connection open:

| Pragma           | Value    | Purpose                                                  |
|------------------|----------|----------------------------------------------------------|
| `journal_mode`   | `WAL`    | Write-ahead logging for concurrent reads during writes   |
| `synchronous`    | `NORMAL` | Durable with WAL; faster than `FULL` with minimal risk   |
| `foreign_keys`   | `ON`     | Enforce FK constraints at the SQLite level               |
| `busy_timeout`   | `5000`   | Wait up to 5 seconds on locks instead of returning busy  |

Run `PRAGMA integrity_check` on startup and abort if it returns anything other
than `ok`. The service must open a single connection (`SetMaxOpenConns(1)`) —
SQLite allows one writer at a time, and all writes are serialized through it.

## 3. Schema design principles

- **Strict tables** — every table uses the `STRICT` option to enforce declared
  column types.
- **Primary keys** — `INTEGER PRIMARY KEY` (int64) on every table. No
  surrogate keys, no UUIDs, no composite PKs unless unavoidable.
- **Timestamps** — `INTEGER` storing Unix milliseconds, never text. The
  application generates values via `time.Now().UnixMilli()`.
- **Byte arrays** — `BLOB` for binary data (token hashes, IP addresses,
  serialized configs). Raw bytes, never hex or base64 encoded.
- **Text** — only for actual string content: emails, display names, subdomain
  labels.
- **Enums** — status and categorical fields stored as `INTEGER`, mapped in
  application code.

## 4. Cursor-based pagination

All list endpoints use cursor-based pagination. The cursor is the `id` of the
last item returned.

```sql
SELECT * FROM table WHERE id > :cursor ORDER BY id ASC LIMIT :page_size;
```

When `next_cursor` is `null`, there are no more pages.

- Consistent performance — always starts at an indexed position, no row scanning
- Stable under writes — no skipped or duplicated items between pages
- Minimal overhead — PK index provides direct access

The tradeoff is no arbitrary page jumping, which is acceptable for "load more"
UI patterns.

## 5. Data safety

**Passwords** — hashed with a standard password hashing function. Stored as
`BLOB` (raw hash output), never logged or stored in plaintext.

**Session tokens** — 32 random bytes from a CSPRNG, hashed (SHA-256 or similar)
before storage. The raw token is returned to the client in a cookie. On lookup,
hash the incoming token and compare against the stored hash. A database leak
never exposes usable tokens.

**IP addresses** — stored as raw bytes: IPv4 as 4-byte `BLOB`, IPv6 as
16-byte `BLOB`. More compact and efficient than string representation.

**Session cleanup** — a periodic job deletes expired rows. Expired sessions are
also ignored during validation, so stale rows are harmless until cleaned up.

**Write durability** — `synchronous = NORMAL` with `journal_mode = WAL` ensures
committed writes survive power loss or OS crash.
