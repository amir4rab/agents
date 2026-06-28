# Provider Reference

For a high-level overview of the provider architecture, see
[architecture/providers.md](../architecture/providers.md).

## Provider Interface

```go
// Provider manages the lifecycle of isolated environments.
type Provider interface {
    // Name returns a unique machine-readable identifier (e.g., "docker").
    Name() string

    // DisplayName returns a human-readable name (e.g., "Docker").
    DisplayName() string

    // Available checks whether this provider can be used on the current
    // system. Returns nil if available, or an error describing why not
    // (e.g., "Docker daemon not reachable", "KVM not supported").
    Available(ctx context.Context) error

    // CreateAgent provisions a new isolated environment and returns its
    // connection info. The returned AgentInfo includes the provider-specific
    // endpoint that StartSession will use.
    CreateAgent(ctx context.Context, opts CreateAgentOptions) (*AgentInfo, error)

    // DestroyAgent tears down the environment and releases all resources.
    DestroyAgent(ctx context.Context, agentID string) error

    // StartSession spawns a new terminal session inside the agent's
    // environment. Returns a SessionHandle for bidirectional I/O.
    StartSession(ctx context.Context, agentID string, opts SessionOptions) (*SessionHandle, error)

    // StopSession terminates a specific terminal session.
    StopSession(ctx context.Context, agentID, sessionID string) error

    // Capabilities returns this provider's resource limits and supported
    // features (max agents, storage support, memory support, etc.).
    Capabilities() ProviderCapabilities

    // HealthCheck verifies the provider is still operational. Used by the
    // registry for ongoing health monitoring.
    HealthCheck(ctx context.Context) error
}
```

### Supporting types

```go
type CreateAgentOptions struct {
    AgentID     string            // unique agent identifier
    UserID      int64             // owning user
    StorageMB   int64             // per-agent storage cap (0 = default)
    MemoryMB    int64             // per-agent memory limit (0 = default)
    Config      map[string]any    // provider-specific configuration
}

type AgentInfo struct {
    AgentID    string
    Endpoint   string            // internal address for session connections
    Config     map[string]any    // provider-specific metadata
}

type SessionOptions struct {
    SessionID  string
    Cols       int               // initial terminal width
    Rows       int               // initial terminal height
}

type SessionHandle struct {
    SessionID string
    Stdin     io.WriteCloser     // send input to the session
    Stdout    io.ReadCloser      // read output from the session
    Resize    func(cols, rows int) error  // resize the PTY
    Close     func() error       // terminate the session
}

type ProviderCapabilities struct {
    MaxAgents        int     // maximum concurrent agents (0 = unlimited)
    SupportsStorage  bool    // supports per-agent storage limits
    SupportsMemory   bool    // supports per-agent memory limits
    RequiresKVM      bool    // requires KVM virtualization support
    RequiresRoot     bool    // requires root or privileged access
}
```

### Contract

- `CreateAgent` must be idempotent from the caller's perspective — if the agent
  already exists, it should return the existing `AgentInfo`.
- `StartSession` must create an independent process with its own PTY. Multiple
  concurrent sessions per agent must be fully isolated.
- `SessionHandle.Stdout` must emit terminal output as raw bytes.
- `SessionHandle.Stdin` accepts raw bytes as user input.
- `Resize` updates the PTY dimensions. The provider is responsible for
  delivering `SIGWINCH` or the equivalent to the session process.
- `Close` sends `SIGTERM` to the session process (or equivalent), then
  `SIGKILL` after a grace period.
- `DestroyAgent` must terminate all active sessions before tearing down the
  environment.

## Provider Registry

The registry is an in-memory singleton that manages all registered providers.

### Startup flow

1. All known providers are registered at service startup.
2. The registry calls `Available()` on each provider.
3. Providers that return `nil` are marked **available**.
4. Providers that return an error are marked **unavailable** (reason recorded).
5. The available list is exposed via the admin API.

### API

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/api/admin/providers` | List all providers with availability status, capabilities, and error info |

Response shape:

```json
{
  "providers": [
    {
      "name": "docker",
      "display_name": "Docker",
      "available": true,
      "capabilities": {
        "max_agents": 0,
        "supports_storage": true,
        "supports_memory": true,
        "requires_kvm": false,
        "requires_root": false
      }
    },
    {
      "name": "firecracker",
      "display_name": "Firecracker",
      "available": false,
      "error": "KVM not supported on this system",
      "capabilities": {
        "max_agents": 20,
        "supports_storage": true,
        "supports_memory": true,
        "requires_kvm": true,
        "requires_root": true
      }
    }
  ]
}
```

### Registry operations

| Operation | Description |
|---|---|
| `Register(p Provider)` | Add a provider to the registry |
| `Scan()` | Update availability of all registered providers |
| `GetAvailable()` | Return all currently available providers |
| `Get(name string)` | Return a specific provider by name (available or not) |
| `IsAvailable(name string)` | Check if a specific provider is available |

### Consumer: orchestrator

The orchestrator uses the registry when creating agents:

1. User requests agent creation with a `provider` field.
2. Orchestrator calls `registry.Get(req.Provider)` to resolve the provider.
3. If not found or unavailable, agent creation is rejected with an appropriate
   error.
4. Orchestrator enforces count-based caps (agent limit, session limit).
5. Orchestrator calls `provider.CreateAgent(ctx, opts)` to provision the
   environment.
6. Provider-specific resource caps (storage, memory) are passed in
   `CreateAgentOptions` and enforced by the provider.

## Adding a new provider

To add a new provider type (e.g., `podman`, `LXC`, `SSH`):

1. Create a new package under `internal/provider/<name>/`.
2. Implement the `Provider` interface.
3. Register the provider in the registry at startup.
4. No changes to the orchestrator, API handlers, or frontend are required —
   the provider is automatically discoverable.

### Provider checklist

When implementing a new provider, verify:

- [ ] `Available()` detects the required binaries, daemons, and hardware features.
- [ ] `CreateAgent()` provisions an isolated environment with the specified
      resource limits.
- [ ] `DestroyAgent()` tears down the environment and frees all resources.
- [ ] `StartSession()` spawns a new PTY-bound process, isolated from other
      sessions.
- [ ] `StopSession()` sends SIGTERM, then SIGKILL after a grace period.
- [ ] Multiple concurrent sessions are fully isolated.
- [ ] `SessionHandle.Resize` propagates dimension changes to the PTY.
- [ ] Provider-specific configuration is passed via `Config` in
      `CreateAgentOptions` (e.g., container image, VM kernel image).
- [ ] The provider does not expose implementation details to the management
      service — all communication is through `SessionHandle`.

## Provider lifecycle in agent operations

### Agent creation

```
User → POST /api/agents { type: "terminal", provider: "docker", ... }
  → Orchestrator: validate caps (count, session)
  → Orchestrator: provider = registry.Get("docker")
  → Orchestrator: provider.CreateAgent(ctx, opts)
  → Provider: provisions container/VM, returns AgentInfo
  → Orchestrator: stores agent record with provider + endpoint in DB
  → Response: agent created
```

### Session creation

```
User → WS connect to /ws/terminal/<agent-id>
  → Orchestrator: authenticate, authorize, check session cap
  → Orchestrator: provider = registry.Get(agent.Provider)
  → Orchestrator: provider.StartSession(ctx, agentID, opts)
  → Provider: docker exec / spawn PTY in VM, returns SessionHandle
  → Orchestrator: relays bytes between WS and SessionHandle
  → Session is active
```

### Session reconnect

```
User → WS connect to /ws/terminal/<agent-id>/<session-id>
  → Orchestrator: look up session in SessionManager
  → Orchestrator: send buffered output, resume relay
  → (Provider is not involved — reconnect is a management service concern)
```

### Agent destruction

```
User → DELETE /api/agents/<id>
  → Orchestrator: terminate all active sessions
  → Orchestrator: provider = registry.Get(agent.Provider)
  → Orchestrator: provider.DestroyAgent(ctx, agentID)
  → Provider: tears down container/VM
  → Orchestrator: removes agent record from DB
```

## Discovery system

Provider discovery ensures that only usable providers are shown to users.

### Implementation

```go
type ProviderDiscovery struct {
    registry *Registry
}

func (pd *ProviderDiscovery) Scan() []ProviderStatus {
    var results []ProviderStatus
    for _, p := range pd.registry.List() {
        err := p.Available(context.Background())
        results = append(results, ProviderStatus{
            Name:        p.Name(),
            DisplayName: p.DisplayName(),
            Available:   err == nil,
            Error:       errString(err),
            Caps:        p.Capabilities(),
        })
    }
    return results
}
```

### What each provider checks

| Provider | Availability check |
|---|---|
| Docker | Docker socket reachable, daemon responds to ping |
| Firecracker | `/dev/kvm` exists and is r/w, `jailer` and `firecracker` binaries in PATH, KVM support confirmed |
| (future) Podman | Podman socket reachable, compatible version |
| (future) SSH | SSH binary in PATH, reachable remote host |

### Schedule

- **Full scan** at service startup — determines which providers are listed
  in the admin panel and agent creation form.
- **On-demand** via `GET /api/admin/providers` — admin can refresh status.
- Providers are **not** periodically polled. If a provider goes down after
  startup, agents using it continue running. The admin must check status and
  act accordingly.
