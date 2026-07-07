# Tenancy Reference

For a high-level overview of the tenancy model, see
[architecture/tenancy.md](../architecture/tenancy.md).

## Space Limits

Each user has a maximum number of spaces they can run simultaneously. The
effective limit is resolved as follows:

1. If a per-user override is set for the user, use that value.
2. Otherwise, use the system-wide default.

The admin configures both through the system settings API:

- **System-wide default** — applies to all normal users unless overridden.
- **Per-user override** — set on a specific user to raise or lower their cap.

At space creation, the service checks the user's active space count against
the effective cap. If the count has reached the cap, creation is rejected.

## Storage Limits

Each space has a configurable storage cap. Storage caps are **resource-based**
and enforced by the provider — the orchestrator passes the limits to the
provider, and the provider is responsible for binding them to the environment
(e.g., filesystem size limit, block device quota).

### Per-space cap

The admin sets a default per-space storage limit in system settings. This is
passed to the provider during space creation.

Provider-specific enforcement:

- **Docker** — applied via `--storage-opt size=<limit>` on the container.
- **Firecracker** — enforced via rootfs image size at VM creation.

The limit can be overridden per-space if needed.

### Per-user cap

The total storage across all of a user's spaces is capped. The effective limit
follows the same two-tier resolution as space count: per-user override if set,
otherwise the system-wide default.

At space creation, the orchestrator checks whether adding the space would cause
the user's total storage to exceed the cap. If so, creation is rejected.

### Usage visibility

- **Admin** — can view per-space usage, per-user total usage, and all caps.
- **Space owner** — can view their own spaces' usage and caps, and their total
  user storage cap.

## Memory Limits

Each space has a configurable memory limit. Memory caps are **resource-based**
and enforced by the provider.

### Per-space cap

The admin sets a default per-space memory limit in system settings. This is
passed to the provider during space creation.

Provider-specific enforcement:

- **Docker** — applied via `--memory` flag on the container.
- **Firecracker** — applied via `--mem-size` flag at VM creation.

The limit can be overridden per-space if needed.

### Per-user cap

The total memory allocation across all of a user's spaces is capped. The
effective limit follows the same two-tier resolution as space count: per-user
override if set, otherwise the system-wide default.

### Usage visibility

- **Admin** — can view per-space memory limits and usage, per-user total
  allocation, and all caps.
- **Space owner** — can view their own spaces' memory limits and usage, and
  their total user memory cap.

## Session Limits

Each space has a configurable maximum number of concurrent terminal sessions.
The effective limit is resolved as follows:

1. If a per-space override is set, use that value.
2. Otherwise, use the system-wide default.

The admin configures both through the system settings API:

- **System-wide default** — applies to all spaces unless overridden (default:
  10).
- **Per-space override** — set on a specific space to raise or lower their cap.

At session creation, the Go service checks the space's active session count
against the effective cap. If the count has reached the cap, the session
creation is rejected.

### Usage visibility

- **Admin** — can view active session count and caps for any space.
- **Space owner** — can view their own spaces' session count and caps.

Sessions are tracked in-memory only and do not persist across restarts.
