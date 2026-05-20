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

// keeperConfig holds the typed configuration for the Keeper backend.
// Currently a stub - will be expanded when full implementation is added.
type keeperConfig struct {
	// Config string // KSM_CONFIG or other configuration
}

// parseConfig converts the raw options map into a typed keeperConfig.
func parseConfig(opts map[string]string) (*keeperConfig, error) {
	return &keeperConfig{}, nil
}

func init() {
	backend.Register(backendName, func(opts map[string]string, isDefault bool) (backend.Backend, error) {
		if isDefault {
			return NewDefaultBackend()
		}
		_, err := parseConfig(opts)
		if err != nil {
			return nil, err
		}
		return &keeperBackend{}, nil
	})
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
