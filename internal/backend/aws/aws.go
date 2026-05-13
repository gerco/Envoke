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
	"os"
	"strings"

	"git.dries.info/gerco/envoke/internal/backend"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

const backendName = "aws"

// awsConfig holds the typed configuration for the AWS backend.
// Parsed once during initialization from the raw map.
type awsConfig struct {
	Region string
	Prefix string
}

// parseConfig converts the raw options map into a typed awsConfig.
// This is called once during backend initialization - business code never touches the raw map.
func parseConfig(opts map[string]string) (*awsConfig, error) {
	return &awsConfig{
		Region: opts["region"],
		Prefix: opts["prefix"],
	}, nil
}

func init() {
	// Register as explicit factory (with options from config)
	backend.Register(backendName, func(opts map[string]string) (backend.Backend, error) {
		return New(opts)
	})
	// Register as default (zero-config) factory using SDK default credential chain
	// Provide fast check function that just looks at env vars (no network calls)
	backend.DefaultRegistry.RegisterDefault(backendName, NewDefaultBackend, checkAWSAvailable)
}

// checkAWSAvailable does a fast check for AWS credentials without network calls.
func checkAWSAvailable() (bool, string) {
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true, ""
	}
	if os.Getenv("AWS_PROFILE") != "" {
		return true, ""
	}
	if os.Getenv("AWS_SDK_LOAD_CONFIG") != "" {
		return true, ""
	}
	if credentialsFileExists() {
		return true, ""
	}
	return false, "AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY, AWS_PROFILE, AWS_SDK_LOAD_CONFIG, or ~/.aws/credentials"
}

func credentialsFileExists() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(home + "/.aws/credentials")
	return err == nil && !info.IsDir()
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
	// Parse raw map into typed config - this is the ONLY place we access opts
	cfg, err := parseConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	ctx := context.Background()

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	if cfg.Region != "" {
		awsCfg.Region = cfg.Region
	}

	return newWithConfig(secretsmanager.NewFromConfig(awsCfg), cfg), nil
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

// newWithConfig constructs an awsBackend with an injected client and typed config (used in tests).
func newWithConfig(client secretsManagerClient, cfg *awsConfig) *awsBackend {
	prefix := cfg.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &awsBackend{client: client, prefix: prefix, region: cfg.Region}
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
