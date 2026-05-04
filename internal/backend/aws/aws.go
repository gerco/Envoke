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

	"git.dries.info/gerco/envoke/internal/backend"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

const backendName = "aws"

func init() {
	// Register as explicit factory (with options from config)
	backend.Register(backendName, func(opts map[string]string) (backend.Backend, error) {
		return New(opts)
	})
	// Register as default (zero-config) factory using SDK default credential chain
	backend.DefaultRegistry.RegisterDefault(backendName, NewDefaultBackend)
}

// secretsManagerClient is the subset of secretsmanager.Client used by awsBackend.
type secretsManagerClient interface {
	GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	PutSecretValue(context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	CreateSecret(context.Context, *secretsmanager.CreateSecretInput, ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
}

// awsBackend implements the envoke backend interface for AWS Secrets Manager.
type awsBackend struct {
	client secretsManagerClient
	prefix string // optional prefix for secret names (e.g., "envoke/")
	region string // AWS region (for display/info purposes)
}

// New creates an AWS Secrets Manager backend with the given options.
func New(opts map[string]string) (*awsBackend, error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	if region, ok := opts["region"]; ok {
		cfg.Region = region
	}

	return newWithClient(secretsmanager.NewFromConfig(cfg), opts), nil
}

// NewDefaultBackend creates an AWS Secrets Manager backend with SDK default credential chain.
// Uses: env vars → ~/.aws/credentials → ~/.aws/config → IAM role.
// Returns (nil, error) if credentials are unavailable (checked without making API calls).
func NewDefaultBackend() (backend.Backend, error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	// Check credentials exist without making an API call
	// This retrieves credentials from the chain without validating them remotely
	_, err = cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws credentials not available: %w", err)
	}

	client := secretsmanager.NewFromConfig(cfg)
	return &awsBackend{client: client, prefix: "", region: cfg.Region}, nil
}

// newWithClient constructs an awsBackend with an injected client (used in tests).
func newWithClient(client secretsManagerClient, opts map[string]string) *awsBackend {
	prefix := ""
	if p, ok := opts["prefix"]; ok {
		prefix = p
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
	}
	return &awsBackend{client: client, prefix: prefix}
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

	existing := make(map[string]string)
	secretExists := true

	result, getErr := a.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if getErr != nil {
		var notFound *types.ResourceNotFoundException
		if !isErrorType(getErr, notFound) {
			return fmt.Errorf("aws get secret %s: %w", secretName, getErr)
		}
		secretExists = false
	} else if result.SecretString != nil {
		if err := json.Unmarshal([]byte(*result.SecretString), &existing); err != nil {
			existing = make(map[string]string)
		}
	}

	existing[key] = value

	jsonData, err := json.Marshal(existing)
	if err != nil {
		return fmt.Errorf("aws marshal secret %s: %w", secretName, err)
	}
	jsonStr := string(jsonData)

	if !secretExists {
		_, err = a.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         &secretName,
			SecretString: &jsonStr,
		})
		if err != nil {
			return fmt.Errorf("aws create secret %s: %w", secretName, err)
		}
		return nil
	}

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
