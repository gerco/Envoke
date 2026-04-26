//go:build aws
// +build aws

// Package aws implements the AWS Secrets Manager backend.
// To minimize AWS costs, all keys in a namespace are stored as fields
// within a single secret (JSON format), not as separate secrets.
// Build with -tags aws to include this backend.
package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"git.dries.info/gerco/envoke/internal/backend"
)

const backendName = "aws"

func init() {
	backend.Register(backendName, func(opts map[string]string) (backend.Backend, error) {
		return New(opts)
	})
}

// awsBackend implements the envoke backend interface for AWS Secrets Manager.
type awsBackend struct {
	client *secretsmanager.Client
	prefix string // optional prefix for secret names (e.g., "envoke/")
}

// New creates an AWS Secrets Manager backend with the given options.
func New(opts map[string]string) (*awsBackend, error) {
	ctx := context.Background()
	
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	// Optional region override
	if region, ok := opts["region"]; ok {
		cfg.Region = region
	}

	client := secretsmanager.NewFromConfig(cfg)

	prefix := ""
	if p, ok := opts["prefix"]; ok {
		prefix = p
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
	}

	return &awsBackend{
		client: client,
		prefix: prefix,
	}, nil
}

// secretName returns the full AWS secret name for a namespace.
func (a *awsBackend) secretName(namespace string) string {
	return a.prefix + namespace
}

// Get retrieves a key from the namespace secret.
// namespace -> AWS secret name, key -> JSON field name.
func (a *awsBackend) Get(namespace, key string) (string, error) {
	ctx := context.Background()
	secretName := a.secretName(namespace)

	result, err := a.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if ok := isErrorType(err, notFound); ok {
			return "", fmt.Errorf("%w: %s/%s", backend.ErrNotFound, namespace, key)
		}
		return "", fmt.Errorf("aws get secret %s: %w", secretName, err)
	}

	// Parse JSON secret value
	var data map[string]string
	if result.SecretString != nil {
		if err := json.Unmarshal([]byte(*result.SecretString), &data); err != nil {
			return "", fmt.Errorf("aws secret %s: invalid JSON: %w", secretName, err)
		}
	} else {
		// Binary secret - not supported
		return "", fmt.Errorf("aws secret %s: binary secrets not supported", secretName)
	}

	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", backend.ErrNotFound, namespace, key)
	}

	return value, nil
}

// Set stores a key in the namespace secret.
// Creates the secret if it doesn't exist, updates if it does.
func (a *awsBackend) Set(namespace, key, value string) error {
	ctx := context.Background()
	secretName := a.secretName(namespace)

	// Try to get existing secret first
	existing := make(map[string]string)
	result, err := a.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err == nil && result.SecretString != nil {
		// Parse existing values
		if err := json.Unmarshal([]byte(*result.SecretString), &existing); err != nil {
			// Invalid JSON - start fresh
			existing = make(map[string]string)
		}
	}

	// Add/update the key
	existing[key] = value

	// Marshal updated data
	jsonData, err := json.Marshal(existing)
	if err != nil {
		return fmt.Errorf("aws marshal secret %s: %w", secretName, err)
	}
	jsonStr := string(jsonData)

	// Update existing or create new
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if isErrorType(err, notFound) {
			// Create new secret
			_, err = a.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
				Name:         &secretName,
				SecretString: &jsonStr,
			})
			if err != nil {
				return fmt.Errorf("aws create secret %s: %w", secretName, err)
			}
			return nil
		}
		return fmt.Errorf("aws get secret %s: %w", secretName, err)
	}

	// Update existing secret
	_, err = a.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     &secretName,
		SecretString: &jsonStr,
	})
	if err != nil {
		return fmt.Errorf("aws put secret %s: %w", secretName, err)
	}

	return nil
}

// List returns all keys (JSON field names) in the namespace secret.
func (a *awsBackend) List(namespace string) ([]string, error) {
	ctx := context.Background()
	secretName := a.secretName(namespace)

	result, err := a.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if isErrorType(err, notFound) {
			return []string{}, nil // Empty list for non-existent secret
		}
		return nil, fmt.Errorf("aws get secret %s: %w", secretName, err)
	}

	// Parse JSON
	var data map[string]string
	if result.SecretString != nil {
		if err := json.Unmarshal([]byte(*result.SecretString), &data); err != nil {
			// Invalid JSON - return empty list
			return []string{}, nil
		}
	}

	// Return keys
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	return keys, nil
}

// isErrorType checks if an error is of a specific AWS error type.
func isErrorType(err error, target error) bool {
	return strings.Contains(err.Error(), "ResourceNotFoundException")
}
