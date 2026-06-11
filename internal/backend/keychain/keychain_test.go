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

// newTestBackend creates a keychainBackend with ring pre-seeded for namespace,
// bypassing keyring.Open so tests never touch the OS keychain.
func newTestBackend(namespace string, r keyring.Keyring) *keychainBackend {
	b := &keychainBackend{rings: make(map[string]keyring.Keyring)}
	if r != nil {
		b.rings[namespace] = r
	}
	return b
}

// --- Get ---

func TestGet_ReturnsDecodedValue(t *testing.T) {
	r := newFakeRing()
	data, _ := json.Marshal("hello")
	r.items["KEY"] = keyring.Item{Key: "KEY", Data: data}
	b := newTestBackend("ns", r)

	got, err := b.Get("ns", "KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestGet_MissingKey_ReturnsErrNotFound(t *testing.T) {
	b := newTestBackend("ns", newFakeRing())

	_, err := b.Get("ns", "MISSING")
	if !errors.Is(err, backend.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestGet_RawBytesFallback(t *testing.T) {
	// Data that is not a valid JSON string falls back to raw bytes.
	r := newFakeRing()
	r.items["RAW"] = keyring.Item{Key: "RAW", Data: []byte("not-json")}
	b := newTestBackend("ns", r)

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
	b := newTestBackend("ns", r)

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
	b := newTestBackend("myns", r)

	if err := b.Set("myns", "MY_KEY", "val"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	item := r.items["MY_KEY"]
	if want := "envoke/myns/MY_KEY"; item.Label != want {
		t.Errorf("label = %q, want %q", item.Label, want)
	}
	if want := "envoke secret"; item.Description != want {
		t.Errorf("description = %q, want %q", item.Description, want)
	}
}

func TestSet_OverwritesExistingKey(t *testing.T) {
	r := newFakeRing()
	b := newTestBackend("ns", r)

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
	r.items["A"] = keyring.Item{Key: "A"}
	r.items["B"] = keyring.Item{Key: "B"}
	r.items["C"] = keyring.Item{Key: "C"}
	b := newTestBackend("ns", r)

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
	b := newTestBackend("ns", newFakeRing())

	keys, err := b.List("ns")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected empty slice, got %v", keys)
	}
}

// --- ring caching ---

func TestRing_Caches(t *testing.T) {
	r := newFakeRing()
	b := newTestBackend("ns", r)

	r1, err := b.ring("ns")
	if err != nil {
		t.Fatalf("first ring(): %v", err)
	}
	r2, err := b.ring("ns")
	if err != nil {
		t.Fatalf("second ring(): %v", err)
	}
	if r1 != r2 {
		t.Error("ring() returned different instances: not cached")
	}
}
