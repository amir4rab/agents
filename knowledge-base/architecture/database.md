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
  labels, provider names.
- **Enums** — status and categorical fields stored as `INTEGER`, mapped in
  application code.

### Agents table fields

The agents table includes a `provider TEXT NOT NULL` column that stores the
name of the provider used to create the agent (e.g., `"docker"`, `"firecracker"`).
This field is set at agent creation and is immutable. The orchestrator uses this
value to resolve the provider from the [Provider Registry](providers.md).

For detailed specifications (cursor-based pagination and data safety
practices), see [specs/database.md](specs/database.md).
