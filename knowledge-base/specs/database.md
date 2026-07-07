# Database Reference

For a high-level overview of the database architecture, see
[architecture/database.md](../architecture/database.md).

## Cursor-Based Pagination

All list endpoints use cursor-based pagination. The cursor is the `id` of the
last item returned.

```sql
SELECT * FROM table WHERE id > :cursor ORDER BY id ASC LIMIT :page_size;
```

When `next_cursor` is null, there are no more pages.

- Consistent performance — always starts at an indexed position, no row scanning
- Stable under writes — no skipped or duplicated items between pages
- Minimal overhead — PK index provides direct access

The tradeoff is no arbitrary page jumping, which is acceptable for "load more"
UI patterns.

## Data Safety

**Passwords** — hashed with Argon2id. Never logged or stored in plaintext.

**Session tokens** — random bytes from a CSPRNG, hashed before storage.
The raw token is returned to the client in a cookie. On lookup, hash the
incoming token and compare against the stored hash. A database leak never
exposes usable tokens.

**IP addresses** — stored as raw bytes: IPv4 as 4-byte, IPv6 as 16-byte.

**Write durability** — `synchronous = NORMAL` with `journal_mode = WAL` ensures
committed writes survive power loss or OS crash.

**Session cleanup** — a periodic job deletes expired rows.
