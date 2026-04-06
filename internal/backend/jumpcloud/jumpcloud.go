// Package jumpcloud will implement the JumpCloud Password Manager backend.
// This is a stub that returns ErrNotImplemented until the REST client is built.
package jumpcloud

import (
	"git.dries.info/gerco/envoke/internal/backend"
)

const backendName = "jumpcloud"

func init() {
	backend.Register(backendName, func(opts map[string]string) (backend.Backend, error) {
		return &jumpcloudBackend{}, nil
	})
}

type jumpcloudBackend struct{}

func (j *jumpcloudBackend) Get(_, _ string) (string, error)  { return "", backend.ErrNotImplemented }
func (j *jumpcloudBackend) Set(_, _, _ string) error         { return backend.ErrNotImplemented }
func (j *jumpcloudBackend) List(_ string) ([]string, error)  { return nil, backend.ErrNotImplemented }
