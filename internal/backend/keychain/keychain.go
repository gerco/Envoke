// Package keychain implements the OS keychain backend via 99designs/keyring.
// On Windows this uses Credential Manager, on macOS the system Keychain,
// and on Linux the Secret Service (e.g. GNOME Keyring / KWallet).
package keychain

import (
	"errors"
	"fmt"

	"github.com/99designs/keyring"

	"git.dries.info/gerco/envoke/internal/backend"
)

const backendName = "keychain"

// keychainConfig holds the typed configuration for the keychain backend.
type keychainConfig struct {
	// ServiceName is the optional service/collection name override.
	// If empty, platform defaults are used: "login" (Linux), "envoke" (macOS/Windows).
	ServiceName string
	// KeyPrefix is the optional prefix for all keys.
	// Default for default backend: platform-specific (e.g., "envoke/" on Linux).
	// Default for custom backend: empty string.
	KeyPrefix string
	// IsDefault indicates whether this is the default (zero-config) backend.
	IsDefault bool
}

// parseConfig converts the raw options map into a typed keychainConfig.
func parseConfig(opts map[string]string, isDefault bool) (*keychainConfig, error) {
	cfg := &keychainConfig{
		ServiceName: opts["service_name"],
		KeyPrefix:   opts["key_prefix"],
		IsDefault:   isDefault,
	}
	return cfg, nil
}

func init() {
	backend.Register(backendName, func(opts map[string]string, isDefault bool) (backend.Backend, error) {
		cfg, err := parseConfig(opts, isDefault)
		if err != nil {
			return nil, err
		}
		return New(cfg)
	})
}

// keychainBackend stores a single keyring.Keyring shared across all namespaces.
// Namespace separation is achieved by prefixing keys with "namespace/".
type keychainBackend struct {
	ring        keyring.Keyring
	serviceName string // Custom service name, or empty for platform default
	keyPrefix   string // Prefix for all keys (e.g., "envoke/")
}

// New creates a keychain backend with the given configuration.
func New(cfg *keychainConfig) (*keychainBackend, error) {
	// Determine key prefix
	prefix := cfg.KeyPrefix
	if prefix == "" && cfg.IsDefault {
		// Default backend uses platform-specific prefix
		prefix = keyPrefix()
	}
	// For custom backends, empty prefix stays empty unless explicitly set

	// Defer keyring opening until first use for default backend
	if cfg.IsDefault {
		return &keychainBackend{ring: nil, serviceName: cfg.ServiceName, keyPrefix: prefix}, nil
	}

	// For explicit backends, open immediately
	r, err := openKeyRing(cfg.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("open keychain: %w", err)
	}
	return &keychainBackend{ring: r, serviceName: cfg.ServiceName, keyPrefix: prefix}, nil
}

// ensureRing lazily initializes the keyring on first use.
// This is needed for NewDefaultBackend which defers opening the keyring.
func (k *keychainBackend) ensureRing() error {
	if k.ring != nil {
		return nil
	}
	r, err := openKeyRing(k.serviceName)
	if err != nil {
		return fmt.Errorf("open keychain: %w", err)
	}
	k.ring = r
	return nil
}

func openKeyRing(customServiceName string) (keyring.Keyring, error) {
	svcName := customServiceName
	if svcName == "" {
		svcName = serviceName()
	}
	return keyring.Open(keyring.Config{
		ServiceName:              svcName,
		AllowedBackends:          allowedBackends(),
		KeychainTrustApplication: true,
	})
}

// Get retrieves a single key's value.
func (k *keychainBackend) Get(namespace, key string) (string, error) {
	if err := k.ensureRing(); err != nil {
		return "", err
	}
	// Use "prefix/namespace/key" as the actual keyring key
	keychainKey := k.keyPrefix + namespace + "/" + key
	item, err := k.ring.Get(keychainKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", fmt.Errorf("%w: %s/%s", backend.ErrNotFound, namespace, key)
		}
		return "", fmt.Errorf("keychain get %s/%s: %w", namespace, key, err)
	}
	return string(item.Data), nil
}

// Set stores key=value in the OS keychain under namespace.
func (k *keychainBackend) Set(namespace, key, value string) error {
	if err := k.ensureRing(); err != nil {
		return err
	}
	// Use "prefix/namespace/key" as the actual keyring key
	keychainKey := k.keyPrefix + namespace + "/" + key
	if err := k.ring.Set(keyring.Item{
		Key:         keychainKey,
		Data:        []byte(value),
		Label:       fmt.Sprintf("envoke/%s/%s", namespace, key),
		Description: "envoke secret",
	}); err != nil {
		return fmt.Errorf("keychain set %s/%s: %w", namespace, key, err)
	}
	return nil
}

// List returns all keys stored in a namespace.
func (k *keychainBackend) List(namespace string) ([]string, error) {
	if err := k.ensureRing(); err != nil {
		return nil, err
	}
	allKeys, err := k.ring.Keys()
	if err != nil {
		return nil, fmt.Errorf("keychain list %s: %w", namespace, err)
	}
	// Filter keys that belong to this namespace
	prefix := k.keyPrefix + namespace + "/"
	var keys []string
	for _, key := range allKeys {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			// Strip the platform prefix and namespace prefix to return just the key name
			keys = append(keys, key[len(prefix):])
		}
	}
	return keys, nil
}
