//go:build keeper
// +build keeper

// Package keeper is a stub for the Keeper Commander Service Mode backend.
// Build with -tags keeper to include this backend.
//
// TODO: Implement full Keeper Commander Service Mode integration.
// See: https://git.dries.info/gerco/Envoke/issues/8
package keeper

import (
	"git.dries.info/gerco/envoke/internal/backend"
)

const backendName = "keeper"

func init() {
	backend.Register(backendName, func(opts map[string]string) (backend.Backend, error) {
		return &keeperBackend{}, nil
	})
}

type keeperBackend struct{}

func (k *keeperBackend) Get(_, _ string) (string, error) { return "", backend.ErrNotImplemented }
func (k *keeperBackend) Set(_, _, _ string) error        { return backend.ErrNotImplemented }
func (k *keeperBackend) List(_ string) ([]string, error) { return nil, backend.ErrNotImplemented }
