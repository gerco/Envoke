package keychain

import (
	"encoding/json"
	"errors"
	"sort"
	"testing"

	"github.com/99designs/keyring"

	"git.dries.info/gerco/envoke/internal/backend"
)

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
func newTestBackend(r keyring.Keyring) *keychainBackend {
	return &keychainBackend{ring: r}
}

// --- Get ---

func TestGet_ReturnsDecodedValue(t *testing.T) {
	r := newFakeRing()
	data, _ := json.Marshal("hello")
	// Key is now stored as "namespace/key"
	r.items["ns/KEY"] = keyring.Item{Key: "ns/KEY", Data: data}
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

func TestGet_RawBytesFallback(t *testing.T) {
	// Data that is not a valid JSON string falls back to raw bytes.
	r := newFakeRing()
	// Key is now stored as "namespace/key"
	r.items["ns/RAW"] = keyring.Item{Key: "ns/RAW", Data: []byte("not-json")}
	b := newTestBackend(r)

	got, err := b.Get("ns", "RAW")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "not-json" {
		t.Errorf("got %q, want %q", got, "not-json")
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

	// Key is now stored as "namespace/key"
	item := r.items["myns/MY_KEY"]
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
	// Keys are now stored as "namespace/key"
	r.items["ns/A"] = keyring.Item{Key: "ns/A"}
	r.items["ns/B"] = keyring.Item{Key: "ns/B"}
	r.items["ns/C"] = keyring.Item{Key: "ns/C"}
	b := newTestBackend(r)

	keys, err := b.List("ns")
	if err != nil {
		t.Fatalf("List: %v", err)
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
	// Keys from different namespaces in the same keyring
	r.items["ns1/A"] = keyring.Item{Key: "ns1/A"}
	r.items["ns1/B"] = keyring.Item{Key: "ns1/B"}
	r.items["ns2/C"] = keyring.Item{Key: "ns2/C"}
	r.items["ns2/D"] = keyring.Item{Key: "ns2/D"}
	b := newTestBackend(r)

	// List ns1 should only return keys from ns1
	keys1, err := b.List("ns1")
	if err != nil {
		t.Fatalf("List ns1: %v", err)
	}
	sort.Strings(keys1)
	want1 := []string{"A", "B"}
	if len(keys1) != len(want1) {
		t.Fatalf("ns1: got %v, want %v", keys1, want1)
	}
	for i := range want1 {
		if keys1[i] != want1[i] {
			t.Errorf("ns1[%d] = %q, want %q", i, keys1[i], want1[i])
		}
	}

	// List ns2 should only return keys from ns2
	keys2, err := b.List("ns2")
	if err != nil {
		t.Fatalf("List ns2: %v", err)
	}
	sort.Strings(keys2)
	want2 := []string{"C", "D"}
	if len(keys2) != len(want2) {
		t.Fatalf("ns2: got %v, want %v", keys2, want2)
	}
	for i := range want2 {
		if keys2[i] != want2[i] {
			t.Errorf("ns2[%d] = %q, want %q", i, keys2[i], want2[i])
		}
	}
}
