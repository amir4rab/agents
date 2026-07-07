package provider

import (
	"context"
	"io"
)

type CreateSpaceOptions struct {
	SpaceID   int64
	UserID    int64
	StorageMB int64
	MemoryMB  int64
	Config    map[string]any
}

type SpaceInfo struct {
	SpaceID  int64
	Endpoint string
	Config   map[string]any
}

type SessionOptions struct {
	SessionID int64
	Cols      int
	Rows      int
}

type SessionHandle struct {
	SessionID int64
	Stdin     io.WriteCloser
	Stdout    io.ReadCloser
	Resize    func(cols, rows int) error
	Close     func() error
}

type ProviderCapabilities struct {
	MaxSpaces       int
	SupportsStorage bool
	SupportsMemory  bool
	RequiresKVM     bool
	RequiresRoot    bool
}

type Provider interface {
	Kind() Kind

	DisplayName() string

	Available(ctx context.Context) error

	CreateSpace(ctx context.Context, opts CreateSpaceOptions) (*SpaceInfo, error)

	DestroySpace(ctx context.Context, spaceID int64) error

	StartSession(ctx context.Context, spaceID int64, opts SessionOptions) (*SessionHandle, error)

	StopSession(ctx context.Context, spaceID, sessionID int64) error

	Capabilities() ProviderCapabilities

	HealthCheck(ctx context.Context) error
}
