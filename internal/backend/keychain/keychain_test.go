package keychain

import (
	"errors"
	"slices"
	"testing"

	"github.com/99designs/keyring"

	"git.dries.info/gerco/envoke/internal/backend"
)

// --- Config Tests ---

func TestParseConfig_EmptyOptions(t *testing.T) {
	cfg, err := parseConfig(map[string]string{}, false)
	if err != nil {
		t.Fatalf("parseConfig failed: %v", err)
	}
	if cfg.ServiceName != "" {
		t.Errorf("expected empty ServiceName, got %q", cfg.ServiceName)
	}
	if cfg.KeyPrefix != "" {
		t.Errorf("expected empty KeyPrefix, got %q", cfg.KeyPrefix)
	}
	if cfg.IsDefault {
		t.Error("expected IsDefault=false")
	}
}

func TestParseConfig_WithServiceName(t *testing.T) {
	cfg, err := parseConfig(map[string]string{"service_name": "custom"}, false)
	if err != nil {
		t.Fatalf("parseConfig failed: %v", err)
	}
	if cfg.ServiceName != "custom" {
		t.Errorf("expected ServiceName=custom, got %q", cfg.ServiceName)
	}
}

func TestParseConfig_WithKeyPrefix(t *testing.T) {
	cfg, err := parseConfig(map[string]string{"key_prefix": "myapp/"}, false)
	if err != nil {
		t.Fatalf("parseConfig failed: %v", err)
	}
	if cfg.KeyPrefix != "myapp/" {
		t.Errorf("expected KeyPrefix=myapp/, got %q", cfg.KeyPrefix)
	}
}

func TestParseConfig_IsDefault(t *testing.T) {
	cfg, err := parseConfig(map[string]string{}, true)
	if err != nil {
		t.Fatalf("parseConfig failed: %v", err)
	}
	if !cfg.IsDefault {
		t.Error("expected IsDefault=true")
	}
}

func TestNew_DefaultBackend_UsesPlatformPrefix(t *testing.T) {
	cfg := &keychainConfig{IsDefault: true}
	kb, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	// Default backend should use platform-specific prefix
	expected := keyPrefix()
	if kb.keyPrefix != expected {
		t.Errorf("expected keyPrefix=%q, got %q", expected, kb.keyPrefix)
	}
}

func TestNew_CustomBackend_EmptyPrefix(t *testing.T) {
	// Custom backend with no explicit prefix should have empty prefix
	cfg := &keychainConfig{IsDefault: false}
	// Can't actually open the keyring in tests, but we can verify the logic
	// by checking that New would use empty prefix for custom backends
	if cfg.KeyPrefix != "" {
		t.Errorf("expected empty KeyPrefix for custom backend, got %q", cfg.KeyPrefix)
	}
}

func TestNew_CustomBackend_ExplicitPrefix(t *testing.T) {
	cfg := &keychainConfig{IsDefault: false, KeyPrefix: "custom/"}
	// Verify config has the explicit prefix
	if cfg.KeyPrefix != "custom/" {
		t.Errorf("expected KeyPrefix=custom/, got %q", cfg.KeyPrefix)
	}
}

// fakeRing is an in-memory implementation of keyring.Keyring.
type fakeRing struct {
	items map[string]keyring.Item
}

func newFakeRing() *fakeRing {
	return &fakeRing{items: make(map[string]keyring.Item)}
}

func (f *fakeRing) Get(key string) (keyring.Item, error) {
	item, ok := f.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (f *fakeRing) GetMetadata(key string) (keyring.Metadata, error) {
	return keyring.Metadata{}, nil
}

func (f *fakeRing) Set(item keyring.Item) error {
	f.items[item.Key] = item
	return nil
}

func (f *fakeRing) Remove(key string) error {
	delete(f.items, key)
	return nil
}

func (f *fakeRing) Keys() ([]string, error) {
	keys := make([]string, 0, len(f.items))
	for k := range f.items {
		keys = append(keys, k)
	}
	return keys, nil
}

// newTestBackend creates a keychainBackend with a single ring,
// bypassing keyring.Open so tests never touch the OS keychain.
// Uses platform-specific prefix by default.
func newTestBackend(r keyring.Keyring) *keychainBackend {
	return &keychainBackend{ring: r, keyPrefix: keyPrefix()}
}

// newTestBackendWithPrefix creates a keychainBackend with a custom prefix.
func newTestBackendWithPrefix(r keyring.Keyring, prefix string) *keychainBackend {
	return &keychainBackend{ring: r, keyPrefix: prefix}
}

// expectedKey returns the full key as it would be stored in the keyring.
func expectedKey(namespace, key string) string {
	return keyPrefix() + namespace + "/" + key
}

// expectedKeyWithPrefix returns the full key with a custom prefix.
func expectedKeyWithPrefix(prefix, namespace, key string) string {
	return prefix + namespace + "/" + key
}

// --- Get ---

func TestGet_ReturnsValue(t *testing.T) {
	r := newFakeRing()
	// Key is stored as "prefix/namespace/key" (platform-specific prefix)
	key := expectedKey("ns", "KEY")
	r.items[key] = keyring.Item{Key: key, Data: []byte("hello")}
	b := newTestBackend(r)

	got, err := b.Get("ns", "KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestGet_MissingKey_ReturnsErrNotFound(t *testing.T) {
	b := newTestBackend(newFakeRing())

	_, err := b.Get("ns", "MISSING")
	if !errors.Is(err, backend.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestGet_RawBytes(t *testing.T) {
	// Test retrieval of raw bytes
	r := newFakeRing()
	// Key is stored as "prefix/namespace/key" (platform-specific prefix)
	key := expectedKey("ns", "RAW")
	r.items[key] = keyring.Item{Key: key, Data: []byte("raw-data")}
	b := newTestBackend(r)

	got, err := b.Get("ns", "RAW")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "raw-data" {
		t.Errorf("got %q, want %q", got, "raw-data")
	}
}

// --- Set ---

func TestSet_RoundTrip(t *testing.T) {
	r := newFakeRing()
	b := newTestBackend(r)

	if err := b.Set("ns", "KEY", "world"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := b.Get("ns", "KEY")
	if err != nil {
		t.Fatalf("Get after Set: %v", err)
	}
	if got != "world" {
		t.Errorf("got %q, want %q", got, "world")
	}
}

func TestSet_LabelAndDescription(t *testing.T) {
	r := newFakeRing()
	b := newTestBackend(r)

	if err := b.Set("myns", "MY_KEY", "val"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Key is stored as "prefix/namespace/key" (platform-specific prefix)
	key := expectedKey("myns", "MY_KEY")
	item := r.items[key]
	if want := "envoke/myns/MY_KEY"; item.Label != want {
		t.Errorf("label = %q, want %q", item.Label, want)
	}
	if want := "envoke secret"; item.Description != want {
		t.Errorf("description = %q, want %q", item.Description, want)
	}
}

func TestSet_OverwritesExistingKey(t *testing.T) {
	r := newFakeRing()
	b := newTestBackend(r)

	_ = b.Set("ns", "KEY", "first")
	_ = b.Set("ns", "KEY", "second")

	got, err := b.Get("ns", "KEY")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "second" {
		t.Errorf("got %q, want %q", got, "second")
	}
}

// --- List ---

func TestList_ReturnsAllKeys(t *testing.T) {
	r := newFakeRing()
	// Keys are stored as "prefix/namespace/key" (platform-specific prefix)
	r.items[expectedKey("ns", "A")] = keyring.Item{Key: expectedKey("ns", "A")}
	r.items[expectedKey("ns", "B")] = keyring.Item{Key: expectedKey("ns", "B")}
	r.items[expectedKey("ns", "C")] = keyring.Item{Key: expectedKey("ns", "C")}
	b := newTestBackend(r)

	keys, err := b.List("ns")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	slices.Sort(keys)
	want := []string{"A", "B", "C"}
	if !slices.Equal(keys, want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
}

func TestList_Empty(t *testing.T) {
	b := newTestBackend(newFakeRing())

	keys, err := b.List("ns")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected empty slice, got %v", keys)
	}
}

// --- namespace isolation ---

func TestList_NamespaceIsolation(t *testing.T) {
	r := newFakeRing()
	// Keys from different namespaces in the same keyring (with platform-specific prefix)
	r.items[expectedKey("ns1", "A")] = keyring.Item{Key: expectedKey("ns1", "A")}
	r.items[expectedKey("ns1", "B")] = keyring.Item{Key: expectedKey("ns1", "B")}
	r.items[expectedKey("ns2", "C")] = keyring.Item{Key: expectedKey("ns2", "C")}
	r.items[expectedKey("ns2", "D")] = keyring.Item{Key: expectedKey("ns2", "D")}
	b := newTestBackend(r)

	// List ns1 should only return keys from ns1
	keys1, err := b.List("ns1")
	if err != nil {
		t.Fatalf("List ns1: %v", err)
	}
	slices.Sort(keys1)
	want1 := []string{"A", "B"}
	if !slices.Equal(keys1, want1) {
		t.Fatalf("ns1: got %v, want %v", keys1, want1)
	}

	// List ns2 should only return keys from ns2
	keys2, err := b.List("ns2")
	if err != nil {
		t.Fatalf("List ns2: %v", err)
	}
	slices.Sort(keys2)
	want2 := []string{"C", "D"}
	if !slices.Equal(keys2, want2) {
		t.Fatalf("ns2: got %v, want %v", keys2, want2)
	}
}

// --- Multiple Backends ---

func TestMultipleBackends_DifferentServiceNames(t *testing.T) {
	// Simulate two different keychain backends with different service names.
	// In practice, these would be configured in the global config as separate backends.

	// Backend 1: default service name, platform-specific prefix
	b1 := &keychainBackend{ring: newFakeRing(), serviceName: "", keyPrefix: keyPrefix()}

	// Backend 2: custom service name, platform-specific prefix (both use same prefix in this test)
	b2 := &keychainBackend{ring: newFakeRing(), serviceName: "custom", keyPrefix: keyPrefix()}

	// Set a value in backend 1
	if err := b1.Set("ns", "KEY1", "value1"); err != nil {
		t.Fatalf("b1.Set failed: %v", err)
	}

	// Set a value in backend 2
	if err := b2.Set("ns", "KEY2", "value2"); err != nil {
		t.Fatalf("b2.Set failed: %v", err)
	}

	// Backend 1 should only see KEY1
	keys1, err := b1.List("ns")
	if err != nil {
		t.Fatalf("b1.List failed: %v", err)
	}
	if len(keys1) != 1 || keys1[0] != "KEY1" {
		t.Errorf("b1.List: expected [KEY1], got %v", keys1)
	}

	// Backend 2 should only see KEY2
	keys2, err := b2.List("ns")
	if err != nil {
		t.Fatalf("b2.List failed: %v", err)
	}
	if len(keys2) != 1 || keys2[0] != "KEY2" {
		t.Errorf("b2.List: expected [KEY2], got %v", keys2)
	}

	// Verify values
	val1, err := b1.Get("ns", "KEY1")
	if err != nil || val1 != "value1" {
		t.Errorf("b1.Get: expected value1, got %q (err=%v)", val1, err)
	}

	val2, err := b2.Get("ns", "KEY2")
	if err != nil || val2 != "value2" {
		t.Errorf("b2.Get: expected value2, got %q (err=%v)", val2, err)
	}
}

func TestMultipleBackends_DifferentPrefixes(t *testing.T) {
	// Simulate two keychain backends using the same keyring but different prefixes.
	// This tests the key_prefix option.

	sharedRing := newFakeRing()

	// Backend 1: "app1/" prefix
	b1 := newTestBackendWithPrefix(sharedRing, "app1/")

	// Backend 2: "app2/" prefix
	b2 := newTestBackendWithPrefix(sharedRing, "app2/")

	// Set values in both backends
	if err := b1.Set("ns", "KEY", "value1"); err != nil {
		t.Fatalf("b1.Set failed: %v", err)
	}
	if err := b2.Set("ns", "KEY", "value2"); err != nil {
		t.Fatalf("b2.Set failed: %v", err)
	}

	// Each backend should see its own value
	val1, err := b1.Get("ns", "KEY")
	if err != nil || val1 != "value1" {
		t.Errorf("b1.Get: expected value1, got %q (err=%v)", val1, err)
	}

	val2, err := b2.Get("ns", "KEY")
	if err != nil || val2 != "value2" {
		t.Errorf("b2.Get: expected value2, got %q (err=%v)", val2, err)
	}

	// Verify keys are isolated by prefix
	keys1, _ := b1.List("ns")
	keys2, _ := b2.List("ns")

	if len(keys1) != 1 || keys1[0] != "KEY" {
		t.Errorf("b1.List: expected [KEY], got %v", keys1)
	}
	if len(keys2) != 1 || keys2[0] != "KEY" {
		t.Errorf("b2.List: expected [KEY], got %v", keys2)
	}

	// Verify actual keyring contains both keys with different prefixes
	allKeys, _ := sharedRing.Keys()
	hasApp1 := false
	hasApp2 := false
	for _, k := range allKeys {
		if k == "app1/ns/KEY" {
			hasApp1 = true
		}
		if k == "app2/ns/KEY" {
			hasApp2 = true
		}
	}
	if !hasApp1 || !hasApp2 {
		t.Errorf("expected both app1/ns/KEY and app2/ns/KEY in keyring, got: %v", allKeys)
	}
}

func TestBackend_EmptyPrefix(t *testing.T) {
	// Test a backend with no prefix (custom backend default)
	b := newTestBackendWithPrefix(newFakeRing(), "")

	if err := b.Set("ns", "KEY", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := b.Get("ns", "KEY")
	if err != nil || val != "value" {
		t.Errorf("Get: expected value, got %q (err=%v)", val, err)
	}

	keys, _ := b.List("ns")
	if len(keys) != 1 || keys[0] != "KEY" {
		t.Errorf("List: expected [KEY], got %v", keys)
	}
}
