# Tenancy Architecture

## 1. Overview

The application supports two user roles: admin and normal users. Every agent
belongs to exactly one user.

Terminal streaming follows the management access model — the admin can view any
agent's terminal stream; normal users can only view their own.

**Core principle:** ownership controls access. Management operations may cross
ownership boundaries (admin). Terminal streaming is management-level access.

## 2. User roles

| Capability                        | Admin | Normal user |
|-----------------------------------|-------|-------------|
| Manage users (CRUD)               | Yes   | No          |
| Configure system (provider, cap)  | Yes   | No          |
| Manage all agents (start/stop)    | Yes   | No          |
| Manage own agents                 | Yes   | Yes         |
| View other users' agent terminal  | Yes   | No          |
| streams (via management WS)       |       |             |

The role is stored as an `INTEGER` enum on the users table (0 = normal,
1 = admin). The admin user is seeded at first startup — there is exactly one
admin account created during initialization.

## 3. Authorization model

### API endpoints

- **Admin endpoints** (`/api/admin/*`) — require the authenticated user to
  have the admin role. Returns 403 if the user is a normal user.
- **User endpoints** (`/api/agents/*`) — scoped to the authenticated user.
  Every query filters by `user_id` from the session. A user cannot see or
  modify agents they do not own.

### Terminal streaming access

Terminal streaming is management-level access. When a user connects to
`/ws/terminal/<agent-id>` on the management domain, the Go service applies
the management authorization model:

- **Admin** — can connect to any agent's terminal stream.
- **Normal user** — can only connect to their own agents' terminal streams.

For implementation details, see [terminal-agents.md](terminal-agents.md).

## 4. Agent limits

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

## 5. Storage limits

Each agent has a configurable storage cap. A user's total storage across all
agents is also capped.

### Per-agent cap

The admin sets a default per-agent storage limit in system settings. This is
enforced at the provider level during agent creation:

- **Docker** — `--storage-opt size=<limit>` on the container.
- **Firecracker** — rootfs size at VM creation.

The limit can be overridden per-agent if needed. The agent runner (the process
inside the container or VM) can query its own storage usage and cap via an
internal endpoint.

### Per-user cap

The total storage across all of a user's agents is capped. The effective limit
follows the same two-tier resolution as agent count: per-user override if set,
otherwise the system-wide default.

At agent creation, the service checks whether adding the agent would cause the
user's total storage to exceed the cap. If so, creation is rejected.

### Usage visibility

- **Admin** — can view per-agent usage, per-user total usage, and all caps.
- **Agent owner** — can view their own agents' usage and caps, and their total
  user storage cap.
- **Agent runner** — can query its own storage usage and cap for the agent it
  is running as.

## 6. Memory limits

Each agent has a configurable memory limit. A user's total memory allocation
across all agents is also capped.

### Per-agent cap

The admin sets a default per-agent memory limit in system settings. This is
enforced at the provider level during agent creation:

- **Docker** — `--memory` flag on the container.
- **Firecracker** — `--mem-size` flag at VM creation.

The limit can be overridden per-agent if needed. The agent runner can query its
own memory limit and current usage via an internal endpoint.

### Per-user cap

The total memory allocation across all of a user's agents is capped. The
effective limit follows the same two-tier resolution as agent count: per-user
override if set, otherwise the system-wide default.

At agent creation, the service checks whether the new agent's memory limit
would cause the user's total to exceed the cap. If so, creation is rejected.

### Usage visibility

- **Admin** — can view per-agent memory limits and usage, per-user total
  allocation, and all caps.
- **Agent owner** — can view their own agents' memory limits and usage, and
  their total user memory cap.
- **Agent runner** — can query its own memory limit and current usage (read
  from cgroups).

## 7. Session limits

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

## 8. Agent sharing (future)

Agent sharing is a planned feature. When implemented, it will allow users to
grant other users access to their agents without transferring ownership. The
ownership model described above remains the base layer — sharing adds a
secondary access path checked alongside ownership.
