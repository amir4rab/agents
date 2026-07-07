# Provider Reference

For a high-level overview of the provider architecture, see
[architecture/providers.md](../architecture/providers.md).

## Provider Interface

```go
// Provider manages the lifecycle of isolated environments.
type Provider interface {
    // Kind returns a unique machine-readable identifier (e.g., "docker").
    Kind() Kind

    // DisplayName returns a human-readable name (e.g., "Docker").
    DisplayName() string

    // Available checks whether this provider can be used on the current
    // system. Returns nil if available, or an error describing why not
    // (e.g., "Docker daemon not reachable", "KVM not supported").
    Available(ctx context.Context) error

    // CreateSpace provisions a new isolated environment and returns its
    // connection info. The returned SpaceInfo includes the provider-specific
    // endpoint that StartSession will use.
    CreateSpace(ctx context.Context, opts CreateSpaceOptions) (*SpaceInfo, error)

    // DestroySpace tears down the environment and releases all resources.
    DestroySpace(ctx context.Context, spaceID int64) error

    // StartSession spawns a new terminal session inside the space's
    // environment. Returns a SessionHandle for bidirectional I/O.
    StartSession(ctx context.Context, spaceID int64, opts SessionOptions) (*SessionHandle, error)

    // StopSession terminates a specific terminal session.
    StopSession(ctx context.Context, spaceID, sessionID int64) error

    // Capabilities returns this provider's resource limits and supported
    // features.
    Capabilities() ProviderCapabilities

    // HealthCheck verifies the provider is still operational.
    HealthCheck(ctx context.Context) error
}
```

### Supporting types

```go
type CreateSpaceOptions struct {
    SpaceID   int64             // unique space identifier
    UserID    int64             // owning user
    StorageMB int64             // per-space storage cap (0 = default)
    MemoryMB  int64             // per-space memory limit (0 = default)
    Config    map[string]any    // provider-specific configuration
}

type SpaceInfo struct {
    SpaceID  int64
    Endpoint string            // internal address for session connections
    Config   map[string]any    // provider-specific metadata
}

type SessionOptions struct {
    SessionID int64
    Cols      int               // initial terminal width
    Rows      int               // initial terminal height
}

type SessionHandle struct {
    SessionID int64
    Stdin     io.WriteCloser    // send input to the session
    Stdout    io.ReadCloser     // read output from the session
    Resize    func(cols, rows int) error  // resize the PTY
    Close     func() error      // terminate the session
}

type ProviderCapabilities struct {
    MaxSpaces       int     // maximum concurrent spaces (0 = unlimited)
    SupportsStorage bool    // supports per-space storage limits
    SupportsMemory  bool    // supports per-space memory limits
    RequiresKVM     bool    // requires KVM virtualization support
    RequiresRoot    bool    // requires root or privileged access
}
```

### Contract

- `CreateSpace` must be idempotent from the caller's perspective — if the space
  already exists, it should return the existing `SpaceInfo`.
- `StartSession` must create an independent process with its own PTY. Multiple
  concurrent sessions per space must be fully isolated.
- `SessionHandle.Stdout` must emit terminal output as raw bytes.
- `SessionHandle.Stdin` accepts raw bytes as user input.
- `Resize` updates the PTY dimensions. The provider is responsible for
  delivering SIGWINCH or the equivalent to the session process.
- `Close` sends SIGTERM to the session process (or equivalent), then SIGKILL
  after a grace period.
- `DestroySpace` must terminate all active sessions before tearing down the
  environment.

## Provider Kinds

| Kind         | Description                        | Availability check                     |
|--------------|------------------------------------|----------------------------------------|
| Docker       | Docker containers                  | Docker socket reachable                |
| Podman       | Podman containers                  | Podman socket reachable                |
| Firecracker  | Firecracker microVMs               | `/dev/kvm` accessible, binaries in PATH|
| QEMU         | QEMU virtual machines              | QEMU binary in PATH                    |
| Container    | macOS containers                   | macOS host detected                    |

## Provider Registry

The registry is an in-memory singleton that manages all registered providers.

### Startup flow

1. All known providers are registered at service startup.
2. The registry calls `Available()` on each provider.
3. Providers that return nil are marked **available**.
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
        "max_spaces": 0,
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
        "max_spaces": 20,
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

## Adding a New Provider

To add a new provider type (e.g., Podman, LXC, SSH):

1. Add a new entry to the provider kind enum.
2. Implement the `Provider` interface.
3. Register the provider in the registry at startup.
4. No changes to the orchestrator, API handlers, or frontend are required —
   the provider is automatically discoverable.

### Provider checklist

When implementing a new provider, verify:

- [ ] `Available()` detects the required binaries, daemons, and hardware features.
- [ ] `CreateSpace()` provisions an isolated environment with the specified
      resource limits.
- [ ] `DestroySpace()` tears down the environment and frees all resources.
- [ ] `StartSession()` spawns a new PTY-bound process, isolated from other
      sessions.
- [ ] `StopSession()` sends SIGTERM, then SIGKILL after a grace period.
- [ ] Multiple concurrent sessions are fully isolated.
- [ ] `SessionHandle.Resize` propagates dimension changes to the PTY.
- [ ] Provider-specific configuration is passed via `Config` in
      `CreateSpaceOptions` (e.g., container image, VM kernel image).
- [ ] The provider does not expose implementation details to the management
      service — all communication is through `SessionHandle`.
