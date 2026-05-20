// Package jumpcloud will implement the JumpCloud Password Manager backend.
// This is a stub that returns ErrNotImplemented until the REST client is built.
package jumpcloud

import (
	"fmt"
	"os"

	"git.dries.info/gerco/envoke/internal/backend"
)

const backendName = "jumpcloud"

// jumpcloudConfig holds the typed configuration for the JumpCloud backend.
// Currently a stub - will be expanded when full implementation is added.
type jumpcloudConfig struct {
	// APIKey string // JumpCloud API key
}

// parseConfig converts the raw options map into a typed jumpcloudConfig.
func parseConfig(opts map[string]string) (*jumpcloudConfig, error) {
	return &jumpcloudConfig{}, nil
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
		return &jumpcloudBackend{}, nil
	})
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
