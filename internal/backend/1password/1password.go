//go:build 1password
// +build 1password

// Package onepassword implements the 1Password Secrets Manager backend.
// This backend uses the official 1Password Go SDK and requires a service account token.
// Build with -tags 1password to include this backend.
package onepassword

import (
	"context"
	"fmt"
	"os"
	"strings"

	"git.dries.info/gerco/envoke/internal/backend"
	"github.com/1password/onepassword-sdk-go"
)

const backendName = "1password"

// opConfig holds the typed configuration for the 1Password backend.
type opConfig struct {
	Token string
}

// parseConfig converts the raw options map into a typed opConfig.
func parseConfig(opts map[string]string) (*opConfig, error) {
	token, ok := opts["token"]
	if !ok {
		return nil, fmt.Errorf("1password backend requires 'token' option (service account token)")
	}
	return &opConfig{Token: token}, nil
}

func init() {
	backend.Register(backendName, func(opts map[string]string, isDefault bool) (backend.Backend, error) {
		if isDefault {
			return NewDefaultBackend()
		}
		cfg, err := parseConfig(opts)
		if err != nil {
			return nil, err
		}
		return New(cfg.Token)
	})
}

// opBackend implements the envoke backend interface for 1Password.
type opBackend struct {
	client *onepassword.Client
}

// New creates a 1Password backend with the given service account token.
func New(token string) (*opBackend, error) {
	client, err := onepassword.NewClient(
		context.Background(),
		onepassword.WithServiceAccountToken(token),
	)
	if err != nil {
		return nil, fmt.Errorf("1password client init: %w", err)
	}
	return &opBackend{client: client}, nil
}

// NewDefaultBackend creates a 1Password backend with zero configuration.
// Requires OP_SERVICE_ACCOUNT_TOKEN environment variable to be set.
func NewDefaultBackend() (backend.Backend, error) {
	token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("OP_SERVICE_ACCOUNT_TOKEN not set")
	}
	return New(token)
}

// Get retrieves a secret from 1Password.
// namespace = vault name, key = item name.
// Optional: key can include field via "item/field" syntax (default field is "password").
func (o *opBackend) Get(namespace, key string) (string, error) {
	itemName, field := parseKey(key)

	reference := fmt.Sprintf("op://%s/%s/%s", namespace, itemName, field)

	secret, err := o.client.Secrets().Resolve(context.Background(), reference)
	if err != nil {
		if isNotFound(err) {
			return "", fmt.Errorf("%w: %s/%s", backend.ErrNotFound, namespace, key)
		}
		return "", fmt.Errorf("1password resolve %s: %w", reference, err)
	}

	return secret, nil
}

// Set is not supported by 1Password SDK (read-only for service accounts).
func (o *opBackend) Set(namespace, key, value string) error {
	return fmt.Errorf("1password backend does not support Set (service accounts are read-only)")
}

// List returns all item names in a vault (namespace).
func (o *opBackend) List(namespace string) ([]string, error) {
	// 1Password SDK v1 doesn't have a direct vault list API.
	// We can iterate items if we have vault UUID, but this requires additional lookup.
	// For now, return empty list - users reference items by name directly.
	// Future: use client.Vaults().Get(ctx, vaultName) + client.Items().List(ctx, vaultUUID)
	return []string{}, nil
}

// parseKey splits "item" or "item/field" into components.
// Default field is "password".
func parseKey(key string) (item, field string) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], "password"
}

// isNotFound checks if error indicates item/vault not found.
// 1Password SDK wraps HTTP errors; we check for common not-found patterns.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// The SDK returns errors with message containing "not found" for missing items
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}
