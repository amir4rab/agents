# Tenancy Reference

For a high-level overview of the tenancy model, see
[architecture/tenancy.md](../architecture/tenancy.md).

## Agent limits

Each user has a maximum number of agents they can run simultaneously. The
effective limit is resolved as follows:

1. If a per-user override is set for the user, use that value.
2. Otherwise, use the system-wide default.

The admin configures both through the system settings API:

- **System-wide default** — applies to all normal users unless overridden.
- **Per-user override** — set on a specific user to raise or lower their cap.

At agent creation, the service checks the user's active agent count against
the effective cap. If the count has reached the cap, creation is rejected with
an appropriate error.

## Storage limits

Each agent has a configurable storage cap. A user's total storage across all
agents is also capped. Storage caps are **resource-based** and enforced by the
provider — the orchestrator passes the limits to the provider via
`CreateAgentOptions`, and the provider is responsible for binding them to the
environment (e.g., filesystem size limit, block device quota).

### Per-agent cap

The admin sets a default per-agent storage limit in system settings. This is
passed to the provider as `CreateAgentOptions.StorageMB` during agent creation.

Provider-specific enforcement:

- **Docker** — applied via `--storage-opt size=<limit>` on the container.
- **Firecracker** — enforced via rootfs image size at VM creation.

The limit can be overridden per-agent if needed.

### Per-user cap

The total storage across all of a user's agents is capped. The effective limit
follows the same two-tier resolution as agent count: per-user override if set,
otherwise the system-wide default.

At agent creation, the orchestrator checks whether adding the agent would cause
the user's total storage to exceed the cap. If so, creation is rejected.

### Usage visibility

- **Admin** — can view per-agent usage, per-user total usage, and all caps.
- **Agent owner** — can view their own agents' usage and caps, and their total
  user storage cap.

## Memory limits

Each agent has a configurable memory limit. A user's total memory allocation
across all agents is also capped. Memory caps are **resource-based** and
enforced by the provider — the orchestrator passes the limits to the provider
via `CreateAgentOptions`, and the provider is responsible for applying them
(e.g., cgroup limit, VM memory size).

### Per-agent cap

The admin sets a default per-agent memory limit in system settings. This is
passed to the provider as `CreateAgentOptions.MemoryMB` during agent creation.

Provider-specific enforcement:

- **Docker** — applied via `--memory` flag on the container.
- **Firecracker** — applied via `--mem-size` flag at VM creation.

The limit can be overridden per-agent if needed.

### Per-user cap

The total memory allocation across all of a user's agents is capped. The
effective limit follows the same two-tier resolution as agent count: per-user
override if set, otherwise the system-wide default.

At agent creation, the orchestrator checks whether the new agent's memory limit
would cause the user's total to exceed the cap. If so, creation is rejected.

### Usage visibility

- **Admin** — can view per-agent memory limits and usage, per-user total
  allocation, and all caps.
- **Agent owner** — can view their own agents' memory limits and usage, and
  their total user memory cap.

## Session limits

Each agent has a configurable maximum number of concurrent terminal sessions.
The effective limit is resolved as follows:

1. If a per-agent override is set for the agent, use that value.
2. Otherwise, use the system-wide default.

The admin configures both through the system settings API:

- **System-wide default** — applies to all agents unless overridden (default: 10).
- **Per-agent override** — set on a specific agent to raise or lower their cap.

At session creation, the Go service checks the agent's active session count
against the effective cap. If the count has reached the cap, the session
creation is rejected with an appropriate error.

### Usage visibility

- **Admin** — can view active session count and caps for any agent.
- **Agent owner** — can view their own agents' session count and caps.

Sessions are tracked in-memory only by the Go service and do not persist
across restarts.
