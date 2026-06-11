// Package keychain implements the OS keychain backend via 99designs/keyring.
// On Windows this uses Credential Manager, on macOS the system Keychain,
// and on Linux the Secret Service (e.g. GNOME Keyring / KWallet).
package keychain

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/99designs/keyring"

	"git.dries.info/gerco/envoke/internal/backend"
)

const backendName = "keychain"

// keychainConfig holds the typed configuration for the keychain backend.
// Currently unused but reserved for future options (e.g. custom service name prefix).
type keychainConfig struct {
	// ServicePrefix string // Optional prefix for service names
}

// parseConfig converts the raw options map into a typed keychainConfig.
func parseConfig(opts map[string]string) (*keychainConfig, error) {
	return &keychainConfig{}, nil
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
		return New(opts)
	})
}

// keychainBackend stores a single keyring.Keyring shared across all namespaces.
// Namespace separation is achieved by prefixing keys with "namespace/".
type keychainBackend struct {
	ring keyring.Keyring
}

// New creates a keychain backend. opts is unused for now but reserved for
// future options (e.g. custom service name prefix).
func New(_ map[string]string) (*keychainBackend, error) {
	r, err := keyring.Open(keyring.Config{
		ServiceName:              "login",
		AllowedBackends:          allowedBackends(),
		KeychainTrustApplication: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open keychain: %w", err)
	}
	return &keychainBackend{ring: r}, nil
}

// NewDefaultBackend creates a keychain backend with zero configuration.
// On macOS and Windows, this is always available.
// On Linux, availability depends on Secret Service (checked when first used).
func NewDefaultBackend() (backend.Backend, error) {
	// Defer keyring opening until first use to avoid slow checks during status/commands.
	return &keychainBackend{ring: nil}, nil
}

// ensureRing lazily initializes the keyring on first use.
// This is needed for NewDefaultBackend which defers opening the keyring.
func (k *keychainBackend) ensureRing() error {
	if k.ring != nil {
		return nil
	}
	r, err := keyring.Open(keyring.Config{
		ServiceName:              "login",
		AllowedBackends:          allowedBackends(),
		KeychainTrustApplication: true,
	})
	if err != nil {
		return fmt.Errorf("open keychain: %w", err)
	}
	k.ring = r
	return nil
}

// Get retrieves a single key's value. Values are stored as JSON-encoded
// strings to allow future structured payloads without format changes.
func (k *keychainBackend) Get(namespace, key string) (string, error) {
	if err := k.ensureRing(); err != nil {
		return "", err
	}
	// Use "namespace/key" as the actual keyring key
	keychainKey := namespace + "/" + key
	item, err := k.ring.Get(keychainKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", fmt.Errorf("%w: %s/%s", backend.ErrNotFound, namespace, key)
		}
		return "", fmt.Errorf("keychain get %s/%s: %w", namespace, key, err)
	}
	var value string
	if err := json.Unmarshal(item.Data, &value); err != nil {
		// Fall back to raw bytes for values stored by other tools.
		return string(item.Data), nil
	}
	return value, nil
}

// Set stores key=value in the OS keychain under namespace.
func (k *keychainBackend) Set(namespace, key, value string) error {
	if err := k.ensureRing(); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("keychain marshal %s/%s: %w", namespace, key, err)
	}
	// Use "namespace/key" as the actual keyring key
	keychainKey := namespace + "/" + key
	if err := k.ring.Set(keyring.Item{
		Key:         keychainKey,
		Data:        data,
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
	prefix := namespace + "/"
	var keys []string
	for _, key := range allKeys {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			// Strip the namespace prefix to return just the key name
			keys = append(keys, key[len(prefix):])
		}
	}
	return keys, nil
}
