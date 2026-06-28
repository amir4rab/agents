# Tenancy Architecture

## 1. Overview

The application supports two user roles: admin and normal users. Every agent
belongs to exactly one user.

Terminal streaming follows the management access model — the admin can view any
agent's terminal stream; normal users can only view their own.

**Core principle:** ownership controls access. Management operations may cross
ownership boundaries (admin). Terminal streaming is management-level access.

### Cap enforcement split

Cap enforcement is split across the orchestrator and providers:

| Cap type | Enforced by | Details |
|---|---|---|
| Agent count (per-user) | Orchestrator | Count-based, checked at agent creation |
| Session count (per-agent) | Orchestrator | Count-based, checked at session creation |
| Storage (per-agent, per-user) | Provider | Resource-based, enforced via provider capabilities |
| Memory (per-agent, per-user) | Provider | Resource-based, enforced via provider capabilities |

The orchestrator checks count-based caps before delegating to the provider.
Resource caps are passed to the provider via `CreateAgentOptions` and
enforced by the provider implementation.

## 2. User roles

| Capability | Admin | Normal user |
|---|---|---|
| Manage users (CRUD) | Yes | No |
| Configure system (provider, cap) | Yes | No |
| Manage all agents (start/stop) | Yes | No |
| Manage own agents | Yes | Yes |
| View other users' agent terminal streams | Yes | No |

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

## 4. Agent sharing (future)

Agent sharing is a planned feature. When implemented, it will allow users to
grant other users access to their agents without transferring ownership. The
ownership model described above remains the base layer — sharing adds a
secondary access path checked alongside ownership.

For detailed specifications on agent, session, storage, and memory limits
(resolution logic, per-user overrides, provider enforcement), see
[specs/tenancy.md](specs/tenancy.md).
