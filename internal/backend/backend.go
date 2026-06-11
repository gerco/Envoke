// Package backend defines the Backend interface and the global registry of named backends.
package backend

import (
	"errors"
	"fmt"
	"sync"
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

// Factory constructs a Backend from options and a flag indicating if this is a default (zero-config) backend.
// The isDefault flag allows backends to use environment-based configuration when true, or explicit config when false.
type Factory func(opts map[string]string, isDefault bool) (Backend, error)

var registry = map[string]Factory{}

// Register associates name with factory. Called from each backend's init().
// This is the legacy registration for explicitly configured backends.
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
	return f(opts, false)
}

// Names returns all registered backend names.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}

// ExplicitConfig holds configuration for an explicit backend that will be created lazily.
type ExplicitConfig struct {
	Type    string
	Options map[string]string
}

// Registry holds explicit backend configurations for lazy resolution.
// No backend caching - each Resolve() creates a fresh instance.
type Registry struct {
	mu             sync.RWMutex
	explicitConfig map[string]ExplicitConfig // explicit backend configs (for lazy creation)
	disabled       map[string]bool           // backends disabled in global config
}

// DefaultRegistry is the global registry instance.
var DefaultRegistry = &Registry{
	explicitConfig: make(map[string]ExplicitConfig),
	disabled:       make(map[string]bool),
}

// SetDisabled replaces the set of disabled backends from global config.
// Names that are not compiled in are silently ignored.
func (r *Registry) SetDisabled(names []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disabled = make(map[string]bool, len(names))
	for _, n := range names {
		r.disabled[n] = true
	}
}

// IsDisabled reports whether the named backend is disabled in config.
func (r *Registry) IsDisabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.disabled[name]
}

// RegisterExplicitConfig registers a backend configuration for lazy resolution.
// This is called during config loading before backends may be imported.
func (r *Registry) RegisterExplicitConfig(name string, backendType string, opts map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.explicitConfig[name] = ExplicitConfig{
		Type:    backendType,
		Options: opts,
	}
}

// Resolve returns a fresh backend instance by name.
// If explicit config exists, uses that. Otherwise uses zero-config (isDefault=true).
// Always creates a new instance - no caching.
// Disabled backends can be overridden by explicit configuration.
func (r *Registry) Resolve(name string) (Backend, error) {
	r.mu.RLock()
	cfg, hasConfig := r.explicitConfig[name]
	isDisabled := r.disabled[name]
	r.mu.RUnlock()

	// If explicit config exists, use it (disabling doesn't apply to explicit config)
	if hasConfig {
		return New(cfg.Type, cfg.Options)
	}

	// No explicit config - use zero-config (default) factory
	if isDisabled {
		return nil, fmt.Errorf("backend %q is disabled", name)
	}

	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("backend %q not found", name)
	}

	// Create fresh instance with isDefault=true
	return f(nil, true)
}

// HasExplicit checks if an explicit backend config is registered.
func (r *Registry) HasExplicit(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.explicitConfig[name]
	return ok
}

// HasBackend checks if a backend factory is registered.
func (r *Registry) HasBackend(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := registry[name]
	return ok
}

// ExplicitNames returns all registered explicit backend names.
func (r *Registry) ExplicitNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.explicitConfig))
	for n := range r.explicitConfig {
		names = append(names, n)
	}
	return names
}

// GetExplicitConfig returns the configuration for an explicit backend.
func (r *Registry) GetExplicitConfig(name string) (ExplicitConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cfg, ok := r.explicitConfig[name]
	return cfg, ok
}
