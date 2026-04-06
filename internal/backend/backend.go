// Package backend defines the Backend interface and the global registry of named backends.
package backend

import (
	"errors"
	"fmt"
)

// ErrNotFound is returned when a key does not exist in a namespace.
var ErrNotFound = errors.New("key not found")

// ErrNotImplemented is returned by stub backends that have not been implemented yet.
var ErrNotImplemented = errors.New("backend not implemented")

// Backend is the single interface every secret backend must satisfy.
type Backend interface {
	// Get retrieves the value of key in namespace.
	Get(namespace, key string) (string, error)
	// Set stores value under key in namespace.
	Set(namespace, key, value string) error
	// List returns all keys in namespace.
	List(namespace string) ([]string, error)
}

// Factory constructs a Backend from an opaque options map supplied by the global config.
type Factory func(opts map[string]string) (Backend, error)

var registry = map[string]Factory{}

// Register associates name with factory. Called from each backend's init().
func Register(name string, f Factory) {
	registry[name] = f
}

// New creates a Backend by name using the provided options.
// Returns an error if name is unknown.
func New(name string, opts map[string]string) (Backend, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown backend %q", name)
	}
	return f(opts)
}

// Names returns all registered backend names.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
