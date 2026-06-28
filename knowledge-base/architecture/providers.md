# Provider Architecture

## 1. Overview

The management service delegates all environment lifecycle management to
**providers**. A provider is a pluggable implementation that manages a specific
type of isolated environment (Docker containers, Firecracker microVMs, or any
future technology).

The management service remains **technology-agnostic** — it interacts with
providers exclusively through a well-defined Go interface. Providers are
registered at startup, discovered via a health check, and selected by the user
at agent creation time. Once an agent is created, its provider is immutable.

**Key design decisions:**

- Providers implement a common Go interface — the management service never
  imports provider-specific packages directly.
- Providers are registered in a central registry and scanned for availability
  at startup.
- The user selects the provider when creating an agent, choosing from the
  available options returned by the registry.
- Provider-specific capabilities (storage limits, memory limits, networking)
  are enforced inside the provider implementation, not in the orchestrator.
- Agent-to-provider binding is permanent — environments are never migrated
  between providers.

## 2. Provider Interface

The management service defines a `Provider` interface that all providers
implement. It includes methods for creating and destroying agents, starting
and stopping terminal sessions, checking availability, and reporting
capabilities.

For the full interface specification (Go code, supporting types, and contract
details), see [specs/providers.md](specs/providers.md).

## 3. Provider Registry

The registry is an in-memory singleton that manages all registered providers.
At startup, all known providers are registered and scanned for availability
via `Available()`. Only available providers are presented to users.

For the full registry API, provider status response schema, orchestrator
integration, and discovery implementation, see
[specs/providers.md](specs/providers.md).

## 4. Interface design rationale

The provider interface follows the codebase's abstraction rules:

> Interfaces are allowed when:
> - Defining domain boundaries
> - Supporting testing
> - Multiple implementations exist

The provider interface satisfies all three conditions:

1. **Domain boundary** — separates environment management from the orchestrator.
2. **Testing** — a mock provider can be used in unit tests without Docker or
   Firecracker.
3. **Multiple implementations** — Docker and Firecracker exist today; Podman,
   LXC, and others are expected.

No abstraction is introduced beyond what these requirements demand. Each
method maps directly to a concrete operation, and each provider implementation
is a self-contained package with its own logic.
