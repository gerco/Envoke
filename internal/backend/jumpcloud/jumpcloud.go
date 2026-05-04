// Package jumpcloud will implement the JumpCloud Password Manager backend.
// This is a stub that returns ErrNotImplemented until the REST client is built.
package jumpcloud

import (
	"fmt"
	"os"

	"git.dries.info/gerco/envoke/internal/backend"
)

const backendName = "jumpcloud"

func init() {
	// Register as explicit factory (with options from config)
	backend.Register(backendName, func(opts map[string]string) (backend.Backend, error) {
		return &jumpcloudBackend{}, nil
	})
	// Register as default (zero-config) factory using JUMPCLOUD_API_KEY env var
	backend.DefaultRegistry.RegisterDefault(backendName, NewDefaultBackend)
}

type jumpcloudBackend struct{}

// NewDefaultBackend creates a JumpCloud backend with zero configuration.
// Requires JUMPCLOUD_API_KEY environment variable to be set.
func NewDefaultBackend() (backend.Backend, error) {
	if os.Getenv("JUMPCLOUD_API_KEY") == "" {
		return nil, fmt.Errorf("JUMPCLOUD_API_KEY not set")
	}
	return &jumpcloudBackend{}, nil
}

func (j *jumpcloudBackend) Get(_, _ string) (string, error) { return "", backend.ErrNotImplemented }
func (j *jumpcloudBackend) Set(_, _, _ string) error        { return backend.ErrNotImplemented }
func (j *jumpcloudBackend) List(_ string) ([]string, error) { return nil, backend.ErrNotImplemented }
