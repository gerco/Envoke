//go:build aws
// +build aws

package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"testing"

	"git.dries.info/gerco/envoke/internal/backend"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// fakeClient is an in-memory implementation of secretsManagerClient.
// secrets maps secret name to JSON string (nil entry = secret does not exist).
type fakeClient struct {
	secrets map[string]*string
}

func newFake(initial map[string]map[string]string) *fakeClient {
	f := &fakeClient{secrets: make(map[string]*string)}
	for name, fields := range initial {
		b, _ := json.Marshal(fields)
		s := string(b)
		f.secrets[name] = &s
	}
	return f
}

func (f *fakeClient) notFound() error {
	msg := "secret not found"
	return &types.ResourceNotFoundException{Message: &msg}
}

func (f *fakeClient) GetSecretValue(_ context.Context, in *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	v, ok := f.secrets[*in.SecretId]
	if !ok || v == nil {
		return nil, f.notFound()
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: v}, nil
}

func (f *fakeClient) PutSecretValue(_ context.Context, in *secretsmanager.PutSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	if _, ok := f.secrets[*in.SecretId]; !ok {
		return nil, f.notFound()
	}
	s := *in.SecretString
	f.secrets[*in.SecretId] = &s
	return &secretsmanager.PutSecretValueOutput{}, nil
}

func (f *fakeClient) CreateSecret(_ context.Context, in *secretsmanager.CreateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	if _, exists := f.secrets[*in.Name]; exists {
		msg := "already exists"
		return nil, fmt.Errorf("%s", msg)
	}
	s := *in.SecretString
	f.secrets[*in.Name] = &s
	return &secretsmanager.CreateSecretOutput{}, nil
}

// stored unmarshals the current JSON for a secret name in the fake.
func (f *fakeClient) stored(name string) map[string]string {
	v := f.secrets[name]
	if v == nil {
		return nil
	}
	var m map[string]string
	_ = json.Unmarshal([]byte(*v), &m)
	return m
}

func newBackend(client *fakeClient, opts map[string]string) *awsBackend {
	cfg := &awsConfig{}
	if opts != nil {
		cfg.Region = opts["region"]
		cfg.Prefix = opts["prefix"]
	}
	return newWithConfig(client, cfg)
}

// --- Set ---

func TestSet_CreatesSecretWhenMissing(t *testing.T) {
	fake := newFake(nil)
	b := newBackend(fake, nil)

	if err := b.Set("myns", "KEY", "value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := fake.stored("myns")["KEY"]; got != "value" {
		t.Errorf("got %q, want %q", got, "value")
	}
}

func TestSet_UpdatesExistingSecret(t *testing.T) {
	fake := newFake(map[string]map[string]string{
		"myns": {"KEY": "old"},
	})
	b := newBackend(fake, nil)

	if err := b.Set("myns", "KEY", "new"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := fake.stored("myns")["KEY"]; got != "new" {
		t.Errorf("got %q, want %q", got, "new")
	}
}

func TestSet_PreservesOtherKeys(t *testing.T) {
	fake := newFake(map[string]map[string]string{
		"myns": {"EXISTING": "keep", "OTHER": "also keep"},
	})
	b := newBackend(fake, nil)

	if err := b.Set("myns", "NEW", "added"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored := fake.stored("myns")
	if stored["EXISTING"] != "keep" {
		t.Errorf("EXISTING overwritten: got %q", stored["EXISTING"])
	}
	if stored["OTHER"] != "also keep" {
		t.Errorf("OTHER overwritten: got %q", stored["OTHER"])
	}
	if stored["NEW"] != "added" {
		t.Errorf("NEW not stored: got %q", stored["NEW"])
	}
}

// --- Get ---

func TestGet_ReturnsValue(t *testing.T) {
	fake := newFake(map[string]map[string]string{
		"myns": {"KEY": "secret"},
	})
	b := newBackend(fake, nil)

	got, err := b.Get("myns", "KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "secret" {
		t.Errorf("got %q, want %q", got, "secret")
	}
}

func TestGet_MissingSecret_ReturnsErrNotFound(t *testing.T) {
	fake := newFake(nil)
	b := newBackend(fake, nil)

	_, err := b.Get("missing", "KEY")
	if !errors.Is(err, backend.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestGet_MissingKey_ReturnsErrNotFound(t *testing.T) {
	fake := newFake(map[string]map[string]string{
		"myns": {"OTHER": "x"},
	})
	b := newBackend(fake, nil)

	_, err := b.Get("myns", "MISSING")
	if !errors.Is(err, backend.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// --- List ---

func TestList_ReturnsKeys(t *testing.T) {
	fake := newFake(map[string]map[string]string{
		"myns": {"A": "1", "B": "2", "C": "3"},
	})
	b := newBackend(fake, nil)

	keys, err := b.List("myns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(keys)
	want := []string{"A", "B", "C"}
	if len(keys) != len(want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Errorf("keys[%d] = %q, want %q", i, keys[i], want[i])
		}
	}
}

func TestList_MissingSecret_ReturnsEmptySlice(t *testing.T) {
	fake := newFake(nil)
	b := newBackend(fake, nil)

	keys, err := b.List("missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected empty slice, got %v", keys)
	}
}

// --- Prefix ---

func TestPrefix_AppliedToSecretName(t *testing.T) {
	fake := newFake(map[string]map[string]string{
		"team/myns": {"KEY": "val"},
	})
	b := newBackend(fake, map[string]string{"prefix": "team"})

	got, err := b.Get("myns", "KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "val" {
		t.Errorf("got %q, want %q", got, "val")
	}
}

func TestPrefix_TrailingSlashNormalised(t *testing.T) {
	fake := newFake(nil)
	b1 := newBackend(fake, map[string]string{"prefix": "team"})
	b2 := newBackend(fake, map[string]string{"prefix": "team/"})
	if b1.prefix != b2.prefix {
		t.Errorf("prefix mismatch: %q vs %q", b1.prefix, b2.prefix)
	}
}
