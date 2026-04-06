// Package keeper will implement the Keeper Secrets Manager backend.
// This is a stub that returns ErrNotImplemented until the SDK integration is built.
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

func (k *keeperBackend) Get(_, _ string) (string, error)        { return "", backend.ErrNotImplemented }
func (k *keeperBackend) Set(_, _, _ string) error               { return backend.ErrNotImplemented }
func (k *keeperBackend) List(_ string) ([]string, error)        { return nil, backend.ErrNotImplemented }
