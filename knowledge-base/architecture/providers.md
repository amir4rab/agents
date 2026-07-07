# Provider Architecture

## 1. Overview

The management service delegates all environment lifecycle management to
**providers**. A provider is a pluggable implementation that manages a specific
type of isolated environment (Docker containers, Firecracker microVMs, or any
future technology).

The management service remains **technology-agnostic** — it interacts with
providers exclusively through a well-defined Go interface with concrete types.
Providers are resolved at runtime by their registered kind. Once a space is
created, its provider is immutable.

**Key design decisions:**

- Providers implement a common Go interface — the management service never
  imports provider-specific packages directly.
- Provider implementations are identified by a typed kind enum.
- Provider-specific capabilities (storage limits, memory limits, networking)
  are enforced inside the provider implementation, not in the orchestrator.
- Space-to-provider binding is permanent — environments are never migrated
  between providers.
- The interface uses concrete types for all parameters and return values,
  establishing a clear contract for the orchestrator.

## 2. Provider Interface

The management service defines a `Provider` interface that all providers
implement:

- `Kind()` — returns the provider's type identifier (e.g., Docker, Firecracker)
- `DisplayName()` — returns a human-readable name
- `Available()` — checks whether this provider can be used on the current system
- `CreateSpace()` — provisions a new isolated environment, taking
  `CreateSpaceOptions` and returning `SpaceInfo`
- `DestroySpace()` — tears down the environment and releases all resources
- `StartSession()` — spawns a new terminal session inside the environment,
  taking `SessionOptions` and returning a `SessionHandle`
- `StopSession()` — terminates a specific terminal session
- `Capabilities()` — returns a populated `ProviderCapabilities` struct
- `HealthCheck()` — verifies the provider is still operational

### Supporting types

**CreateSpaceOptions** — carries the parameters for provisioning a new space:
unique space identifier, owning user ID, per-space resource caps (storage,
memory), and an opaque config map for provider-specific settings.

**SpaceInfo** — returned after successful space creation, containing the
space ID, internal connection endpoint, and provider-specific metadata.

**SessionOptions** — carries parameters for starting a new terminal session:
unique session ID and initial terminal dimensions (columns, rows).

**SessionHandle** — the runtime handle for an active terminal session,
providing stdin (write user input), stdout (read terminal output), resize
(update PTY dimensions), and close (terminate the session).

**ProviderCapabilities** — declares the provider's resource limits and
supported features: maximum concurrent spaces, whether it supports per-space
storage and memory limits, and whether it requires KVM or root access.

### Provider kinds

| Kind         | Description                        |
|--------------|------------------------------------|
| Docker       | Docker containers                  |
| Podman       | Podman containers                  |
| Firecracker  | Firecracker microVMs               |
| QEMU         | QEMU virtual machines              |
| Container    | macOS containers                   |

## 3. Provider Registry

The registry is an in-memory singleton that manages all registered providers.
At startup, all known providers are registered and scanned for availability.
Only available providers are presented to users via the admin API.

The orchestrator resolves providers by name at space-creation time and caches
the binding in the database.

## 4. Interface Design Rationale

The provider interface follows the codebase's abstraction rules:

> Interfaces are allowed when:
> - Defining domain boundaries
> - Supporting testing
> - Multiple implementations exist

The provider interface satisfies all three conditions:

1. **Domain boundary** — separates environment management from the rest of
   the service.
2. **Testing** — a mock provider can be used in unit tests without Docker or
   Firecracker.
3. **Multiple implementations** — Docker, Podman, Firecracker, QEMU, and
   Container are all expected provider kinds.

No abstraction is introduced beyond what these requirements demand. Each method
maps directly to a concrete operation, and each provider implementation is a
self-contained package with its own logic. The concrete types define a precise
contract without leaking implementation details.
