package runner

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

type memBackend struct {
	data map[string]map[string]string
}

func newMemBackend() *memBackend {
	return &memBackend{data: make(map[string]map[string]string)}
}

func (m *memBackend) Get(ns, key string) (string, error) {
	v, ok := m.data[ns][key]
	if !ok {
		return "", backend.ErrNotFound
	}
	return v, nil
}

func (m *memBackend) Set(ns, key, value string) error {
	if m.data[ns] == nil {
		m.data[ns] = make(map[string]string)
	}
	m.data[ns][key] = value
	return nil
}

func (m *memBackend) List(ns string) ([]string, error) {
	var keys []string
	for k := range m.data[ns] {
		keys = append(keys, k)
	}
	return keys, nil
}

// listErrBackend returns a fixed error from List.
type listErrBackend struct{ err error }

func (e *listErrBackend) Get(_, _ string) (string, error) { return "", nil }
func (e *listErrBackend) Set(_, _, _ string) error        { return nil }
func (e *listErrBackend) List(_ string) ([]string, error) { return nil, e.err }

// getErrBackend lists one key but returns a non-ErrNotFound error from Get.
type getErrBackend struct{ err error }

func (g *getErrBackend) Get(_, _ string) (string, error) { return "", g.err }
func (g *getErrBackend) Set(_, _, _ string) error        { return nil }
func (g *getErrBackend) List(_ string) ([]string, error) { return []string{"KEY"}, nil }

// registerFixedBackend injects b into the global registry and DefaultRegistry under name.
// Uses name as both the type key and the explicit config name.
// Registrations persist for the lifetime of the test process — use unique names per test.
func registerFixedBackend(name string, b backend.Backend) {
	backend.Register(name, func(_ map[string]string, _ bool) (backend.Backend, error) {
		return b, nil
	})
	backend.DefaultRegistry.RegisterExplicitConfig(name, name, nil)
}

// envSliceToMap converts a KEY=VALUE slice to a map.
func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				m[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return m
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestBuildEnv_SingleNamespace(t *testing.T) {
	b := newMemBackend()
	b.data["myns"] = map[string]string{"TEST_SECRET": "hello"}
	registerFixedBackend("runner-test-single", b)
	t.Cleanup(func() { os.Unsetenv("TEST_SECRET") })

	cfg := &config.Loaded{
		Namespaces: []config.NamespaceEntry{
			{Name: "myns", Backend: "runner-test-single"},
		},
	}

	env, err := buildEnv(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := envSliceToMap(env)
	if got["TEST_SECRET"] != "hello" {
		t.Errorf("TEST_SECRET = %q, want %q", got["TEST_SECRET"], "hello")
	}
}

// TestBuildEnv_Chaining verifies sequential namespace activation: secrets written
// to the process environment by namespace N are visible to namespace N+1's backend
// factory at resolution time.
func TestBuildEnv_Chaining(t *testing.T) {
	// Namespace 1 provides a credential key.
	b1 := newMemBackend()
	b1.data["cred-ns"] = map[string]string{"RUNNER_CHAIN_CRED": "chained-value"}
	registerFixedBackend("runner-test-chain-b1", b1)

	// Namespace 2's factory captures the env var injected by namespace 1.
	var capturedCred string
	backend.Register("runner-test-chain-b2-type", func(_ map[string]string, _ bool) (backend.Backend, error) {
		capturedCred = os.Getenv("RUNNER_CHAIN_CRED")
		return newMemBackend(), nil
	})
	backend.DefaultRegistry.RegisterExplicitConfig("runner-test-chain-b2", "runner-test-chain-b2-type", nil)

	os.Unsetenv("RUNNER_CHAIN_CRED")
	t.Cleanup(func() { os.Unsetenv("RUNNER_CHAIN_CRED") })

	cfg := &config.Loaded{
		Namespaces: []config.NamespaceEntry{
			{Name: "cred-ns", Backend: "runner-test-chain-b1"},
			{Name: "secret-ns", Backend: "runner-test-chain-b2"},
		},
	}

	if _, err := buildEnv(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedCred != "chained-value" {
		t.Errorf("backend 2 saw RUNNER_CHAIN_CRED=%q, want %q", capturedCred, "chained-value")
	}
}

func TestBuildEnv_UnknownBackend(t *testing.T) {
	cfg := &config.Loaded{
		Namespaces: []config.NamespaceEntry{
			{Name: "ns", Backend: "runner-test-does-not-exist"},
		},
	}

	_, err := buildEnv(cfg)
	if err == nil {
		t.Fatal("expected error for unknown backend, got nil")
	}
}

func TestBuildEnv_ListError(t *testing.T) {
	listErr := fmt.Errorf("list unavailable")
	registerFixedBackend("runner-test-list-err", &listErrBackend{err: listErr})

	cfg := &config.Loaded{
		Namespaces: []config.NamespaceEntry{
			{Name: "ns", Backend: "runner-test-list-err"},
		},
	}

	_, err := buildEnv(cfg)
	if !errors.Is(err, listErr) {
		t.Errorf("got %v, want error wrapping %v", err, listErr)
	}
}

func TestBuildEnv_GetError(t *testing.T) {
	getErr := fmt.Errorf("get unavailable")
	registerFixedBackend("runner-test-get-err", &getErrBackend{err: getErr})

	cfg := &config.Loaded{
		Namespaces: []config.NamespaceEntry{
			{Name: "ns", Backend: "runner-test-get-err"},
		},
	}

	_, err := buildEnv(cfg)
	if !errors.Is(err, getErr) {
		t.Errorf("got %v, want error wrapping %v", err, getErr)
	}
}
