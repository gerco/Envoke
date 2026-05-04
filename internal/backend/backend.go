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

// Factory constructs a Backend from an opaque options map supplied by the global config.
type Factory func(opts map[string]string) (Backend, error)

// DefaultFactory creates a Backend with zero configuration.
// Returns an error if the backend cannot be initialized (e.g., missing credentials).
type DefaultFactory func() (Backend, error)

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

// ExplicitConfig holds configuration for an explicit backend that will be created lazily.
type ExplicitConfig struct {
	Type    string
	Options map[string]string
}

// Registry manages both default (zero-config) and explicit (configured) backends.
type Registry struct {
	mu             sync.RWMutex
	defaults       map[string]DefaultFactory // provider name -> zero-config factory
	explicit       map[string]Backend        // created explicit backends (cached)
	explicitConfig map[string]ExplicitConfig // explicit backend configs (for lazy creation)
}

// DefaultRegistry is the global registry instance.
var DefaultRegistry = &Registry{
	defaults:       make(map[string]DefaultFactory),
	explicit:       make(map[string]Backend),
	explicitConfig: make(map[string]ExplicitConfig),
}

// RegisterDefault allows backends to self-register as zero-config providers.
// The factory should return (nil, error) if zero-config is not possible.
func (r *Registry) RegisterDefault(providerName string, factory DefaultFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaults[providerName] = factory
}

// RegisterExplicit registers a pre-created configured backend from config file.
// Explicit backends take precedence over defaults.
func (r *Registry) RegisterExplicit(name string, b Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.explicit[name] = b
}

// RegisterExplicitConfig registers a backend configuration for lazy creation.
// This is called during config loading before backends may be imported.
func (r *Registry) RegisterExplicitConfig(name string, backendType string, opts map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.explicitConfig[name] = ExplicitConfig{
		Type:    backendType,
		Options: opts,
	}
}

// createExplicitBackend creates a backend from stored config.
func (r *Registry) createExplicitBackend(name string, cfg ExplicitConfig) (Backend, error) {
	return New(cfg.Type, cfg.Options)
}

// Resolve returns a backend by name.
// Priority: explicit config (lazy-created) > explicit config (pre-created) > default factory > error.
func (r *Registry) Resolve(name string) (Backend, error) {
	r.mu.RLock()

	// 1. Check for already-created explicit backend
	if b, ok := r.explicit[name]; ok {
		r.mu.RUnlock()
		return b, nil
	}

	// 2. Check for explicit config that needs lazy creation
	cfg, hasConfig := r.explicitConfig[name]

	// 3. Check for default factory
	factory, hasDefault := r.defaults[name]

	r.mu.RUnlock()

	// Create from config if available (explicit wins over default)
	if hasConfig {
		b, err := r.createExplicitBackend(name, cfg)
		if err != nil {
			// If creation fails, try default as fallback if available
			if hasDefault {
				return factory()
			}
			return nil, fmt.Errorf("explicit backend %q (%s): %w", name, cfg.Type, err)
		}
		// Cache the created backend
		r.mu.Lock()
		r.explicit[name] = b
		r.mu.Unlock()
		return b, nil
	}

	// 4. Try default factory (creates on demand)
	if hasDefault {
		return factory()
	}

	return nil, fmt.Errorf("backend %q not found", name)
}

// HasDefault checks if a default (zero-config) factory is registered.
func (r *Registry) HasDefault(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.defaults[name]
	return ok
}

// HasExplicit checks if an explicit backend config is registered (lazy or created).
func (r *Registry) HasExplicit(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.explicit[name]
	if ok {
		return true
	}
	_, ok = r.explicitConfig[name]
	return ok
}

// DefaultNames returns all registered default backend names.
func (r *Registry) DefaultNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.defaults))
	for n := range r.defaults {
		names = append(names, n)
	}
	return names
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

// CheckExplicit tests if an explicit backend can be created.
// Returns (true, nil) if available, (false, error) with details if not.
func (r *Registry) CheckExplicit(name string) (bool, error) {
	r.mu.RLock()

	// Already created
	if _, ok := r.explicit[name]; ok {
		r.mu.RUnlock()
		return true, nil
	}

	// Has config but not created yet
	cfg, ok := r.explicitConfig[name]
	r.mu.RUnlock()

	if !ok {
		return false, fmt.Errorf("backend %q has no explicit configuration", name)
	}

	// Try to create it
	_, err := r.createExplicitBackend(name, cfg)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetExplicitConfig returns the configuration for an explicit backend.
func (r *Registry) GetExplicitConfig(name string) (ExplicitConfig, bool) {
	r.mu.RLock()
	defer r.mu.Unlock()
	cfg, ok := r.explicitConfig[name]
	return cfg, ok
}
