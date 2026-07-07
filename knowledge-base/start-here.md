# Start here

Before implementing any features:

1) Read ai-rules.md.
2) Read architecture/system-overview.md.
3) Read architecture/providers.md.
4) Read architecture/backend-patterns.md and architecture/frontend-patterns.md.
5) Find an existing implementation.
6) Follow existing patterns.

**Important: Spec files (`specs/`) are deep reference documents — read them
only on-demand when implementing or modifying the specific component they
describe.** For example, read `specs/providers.md` only when implementing a
new provider or modifying the provider interface. Reading spec files for
general orientation fills your context window with unnecessary detail — the
`architecture/` files are sufficient for understanding the system.

Projects main principles:

- Simplicity over cleverness
- Consistency over innovation
- Explicit code over magic
- Concrete implementations over abstractions
- Existing patterns over new patterns

**Never introduce new architectural approaches without explicit approval.**
