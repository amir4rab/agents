# System Overview — Detailed Lifecycle

For a high-level overview of the system, see
[architecture/system-overview.md](../architecture/system-overview.md).

## Agent lifecycle (terminal-based agents)

1. User requests a new agent of type `terminal` via the UI or REST API,
   selecting a provider from the available options (e.g., Docker, Firecracker)
2. The orchestrator resolves the selected provider from the registry,
   validates count-based caps, then delegates environment creation to the
   provider via `CreateAgent()`
3. The provider provisions the environment and returns the agent's internal
   connection endpoint
4. The Go service records the agent's provider and endpoint in the database
5. The user is taken to the terminal UI in the Angular SPA, which shows the
   agent's terminal page with a tabbed interface. Each tab opens a new
   independent WebSocket session to `/ws/terminal/<agent-id>`. Multiple
   concurrent sessions per agent are supported.
6. On deletion, the orchestrator stops the agent and removes the terminal
   endpoint registration
