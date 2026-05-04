//go:build keeper
// +build keeper

// Package keeper is a stub for the Keeper Commander Service Mode backend.
// Build with -tags keeper to include this backend.
//
// TODO: Implement full Keeper Commander Service Mode integration.
// See: https://git.dries.info/gerco/Envoke/issues/8
package keeper

import (
	"fmt"
	"os"

	"git.dries.info/gerco/envoke/internal/backend"
)

const backendName = "keeper"

func init() {
	// Register as explicit factory (with options from config)
	backend.Register(backendName, func(opts map[string]string) (backend.Backend, error) {
		return &keeperBackend{}, nil
	})
	// Register as default (zero-config) factory using KSM_CONFIG env var
	backend.DefaultRegistry.RegisterDefault(backendName, NewDefaultBackend)
}

type keeperBackend struct{}

// NewDefaultBackend creates a Keeper backend with zero configuration.
// Requires KSM_CONFIG environment variable to be set.
func NewDefaultBackend() (backend.Backend, error) {
	if os.Getenv("KSM_CONFIG") == "" {
		return nil, fmt.Errorf("KSM_CONFIG not set")
	}
	return &keeperBackend{}, nil
}

func (k *keeperBackend) Get(_, _ string) (string, error) { return "", backend.ErrNotImplemented }
func (k *keeperBackend) Set(_, _, _ string) error        { return backend.ErrNotImplemented }
func (k *keeperBackend) List(_ string) ([]string, error) { return nil, backend.ErrNotImplemented }
