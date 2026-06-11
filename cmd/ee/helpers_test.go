package main

import (
	"reflect"
	"runtime"
	"testing"

	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

// ---------------------------------------------------------------------------
// Shared test types used across all cmd/ee test files.
// ---------------------------------------------------------------------------

// memBackend is an in-memory Backend for use in cmd tests.
type memBackend struct {
	data map[string]map[string]string // namespace -> key -> value
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

// errBackend always returns a fixed error from every method.
type errBackend struct{ err error }

func (e *errBackend) Get(_, _ string) (string, error) { return "", e.err }
func (e *errBackend) Set(_, _, _ string) error        { return e.err }
func (e *errBackend) List(_ string) ([]string, error) { return nil, e.err }

// registerTestBackend injects b into DefaultRegistry under name.
// The registration persists for the lifetime of the test process, so callers
// must use unique names to avoid cross-test interference.
func registerTestBackend(name string, b backend.Backend) {
	backend.Register(name, func(_ map[string]string, _ bool) (backend.Backend, error) {
		return b, nil
	})
	backend.DefaultRegistry.RegisterExplicitConfig(name, name, nil)
}

// loadedConfig returns a minimal *config.Loaded with one namespace entry.
func loadedConfig(nsName, backendName string) *config.Loaded {
	return &config.Loaded{
		Namespaces: []config.NamespaceEntry{{Name: nsName, Backend: backendName}},
	}
}

// setGlobalConfigDir redirects the global config lookup to dir for the test.
// Mirrors the helper in internal/config/loader_test.go.
func setGlobalConfigDir(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", dir)
	} else {
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
}

// ---------------------------------------------------------------------------
// Tests for getBackendName
// ---------------------------------------------------------------------------

func TestGetBackendName_FoundInConfig(t *testing.T) {
	cfg := loadedConfig("myns", "aws-prod")
	got := getBackendName(cfg, "myns")
	if got != "aws-prod" {
		t.Errorf("got %q, want %q", got, "aws-prod")
	}
}

func TestGetBackendName_NotInConfig_DefaultsToKeychain(t *testing.T) {
	cfg := &config.Loaded{} // empty namespace list
	got := getBackendName(cfg, "unknown-ns")
	if got != "keychain" {
		t.Errorf("got %q, want %q", got, "keychain")
	}
}

// ---------------------------------------------------------------------------
// Tests for mergeOpts
// ---------------------------------------------------------------------------

func TestMergeOpts_BothNil(t *testing.T) {
	if got := mergeOpts(nil, nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestMergeOpts_BaseOnly(t *testing.T) {
	base := map[string]string{"a": "1", "b": "2"}
	got := mergeOpts(base, nil)
	if !reflect.DeepEqual(got, base) {
		t.Errorf("got %v, want %v", got, base)
	}
}

func TestMergeOpts_OverrideOnly(t *testing.T) {
	override := map[string]string{"x": "9"}
	got := mergeOpts(nil, override)
	if !reflect.DeepEqual(got, override) {
		t.Errorf("got %v, want %v", got, override)
	}
}

func TestMergeOpts_OverrideWins(t *testing.T) {
	base := map[string]string{"a": "1", "b": "2"}
	override := map[string]string{"b": "99", "c": "3"}
	got := mergeOpts(base, override)
	want := map[string]string{"a": "1", "b": "99", "c": "3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
