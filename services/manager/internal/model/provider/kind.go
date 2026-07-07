package provider

import (
	"errors"
)

// Possible kinds of available providers
type Kind uint16

const (
	_           Kind = iota
	Docker           // Docker
	Podman           // Podman
	Firecracker      // Firecracker
	QEMU             // QEMU
	Container        // Container (macOS only)
)

// Provider kind to string
var ptts = map[Kind][]byte{
	Docker:      []byte("\"docker\""),
	Podman:      []byte("\"podman\""),
	Firecracker: []byte("\"firecracker\""),
	QEMU:        []byte("\"qemu\""),
	Container:   []byte("\"container\""),
}

func (pt Kind) MarshalJSON() ([]byte, error) {
	b, ok := ptts[pt]
	if !ok {
		return nil, errors.New("invalid provider kind")
	}
	return b, nil
}

// Provider kind from string
var ptfs = map[string]Kind{
	"docker":      Docker,
	"podman":      Podman,
	"firecracker": Firecracker,
	"qemu":        QEMU,
	"container":   Container,
}

func (pt *Kind) UnmarshalJSON(b []byte) error {
	str := string(b)
	strLen := len(str)

	if strLen < 3 {
		return errors.New("provider value is too short")
	}

	str = str[1 : strLen-1]
	val, ok := ptfs[str]
	if !ok {
		return errors.New("invalid provider kind")
	}

	*pt = val
	return nil
}
