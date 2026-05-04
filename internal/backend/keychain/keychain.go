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

func init() {
	// Register as explicit factory (with options from config)
	backend.Register(backendName, func(opts map[string]string) (backend.Backend, error) {
		return New(opts)
	})
	// Register as default (zero-config) factory - always available
	backend.DefaultRegistry.RegisterDefault(backendName, NewDefaultBackend)
}

// keychainBackend stores a keyring.Keyring opened per namespace on first use.
type keychainBackend struct {
	rings map[string]keyring.Keyring
}

// New creates a keychain backend. opts is unused for now but reserved for
// future options (e.g. custom service name prefix).
func New(_ map[string]string) (*keychainBackend, error) {
	return &keychainBackend{rings: make(map[string]keyring.Keyring)}, nil
}

// NewDefaultBackend creates a keychain backend with zero configuration.
// On macOS and Windows, this is always available.
// On Linux, availability depends on Secret Service (checked when first used).
func NewDefaultBackend() (backend.Backend, error) {
	// Always return the backend - actual availability is determined when used.
	// This avoids slow checks during status/commands.
	return &keychainBackend{rings: make(map[string]keyring.Keyring)}, nil
}

func (k *keychainBackend) ring(namespace string) (keyring.Keyring, error) {
	if r, ok := k.rings[namespace]; ok {
		return r, nil
	}
	r, err := keyring.Open(keyring.Config{
		ServiceName: "envoke/" + namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("open keychain namespace %q: %w", namespace, err)
	}
	k.rings[namespace] = r
	return r, nil
}

// Get retrieves a single key's value. Values are stored as JSON-encoded
// strings to allow future structured payloads without format changes.
func (k *keychainBackend) Get(namespace, key string) (string, error) {
	r, err := k.ring(namespace)
	if err != nil {
		return "", err
	}
	item, err := r.Get(key)
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
	r, err := k.ring(namespace)
	if err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("keychain marshal %s/%s: %w", namespace, key, err)
	}
	if err := r.Set(keyring.Item{
		Key:         key,
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
	r, err := k.ring(namespace)
	if err != nil {
		return nil, err
	}
	keys, err := r.Keys()
	if err != nil {
		return nil, fmt.Errorf("keychain list %s: %w", namespace, err)
	}
	return keys, nil
}
