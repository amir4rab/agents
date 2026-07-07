# Tenancy Architecture

## 1. Overview

The application supports two user roles: admin and normal users. Every space
belongs to exactly one user.

Terminal streaming follows the management access model — the admin can view
any space's terminal stream; normal users can only view their own.

**Core principle:** ownership controls access. Management operations may cross
ownership boundaries (admin). Terminal streaming is management-level access.

### Cap enforcement split

Cap enforcement is split across the orchestrator and providers:

| Cap type | Enforced by | Details |
|---|---|---|
| Space count (per-user) | Orchestrator | Count-based, checked at space creation |
| Session count (per-space) | Orchestrator | Count-based, checked at session creation |
| Storage (per-space, per-user) | Provider | Resource-based, enforced via provider capabilities |
| Memory (per-space, per-user) | Provider | Resource-based, enforced via provider capabilities |

The orchestrator checks count-based caps before delegating to the provider.
Resource caps are configured per-space and enforced by the provider
implementation.

## 2. User Roles

| Capability | Admin | Normal user |
|---|---|---|
| Manage users (CRUD) | Yes | No |
| Configure system (provider, cap) | Yes | No |
| Manage all spaces | Yes | No |
| Manage own spaces | Yes | Yes |
| View other users' terminal streams | Yes | No |

The role is stored as an integer enum on the users table. The admin user is
seeded at first startup — there is exactly one admin account created during
initialization.

## 3. Authorization Model

### API endpoints

- **Admin endpoints** (`/api/admin/*`) — require the authenticated user to
  have the admin role. Returns 403 if the user is a normal user.
- **User endpoints** (`/api/spaces/*`) — scoped to the authenticated user.
  Every query filters by `user_id` from the session. A user cannot see or
  modify spaces they do not own.

### Terminal streaming access

Terminal streaming is management-level access. When a user connects to
`/ws/terminal/<space-id>` on the management domain, the Go service applies
the management authorization model:

- **Admin** — can connect to any space's terminal stream.
- **Normal user** — can only connect to their own spaces' terminal streams.

## 4. Space Sharing (Future)

Space sharing is a planned feature. When implemented, it will allow users to
grant other users access to their spaces without transferring ownership. The
ownership model described above remains the base layer — sharing adds a
secondary access path checked alongside ownership.
