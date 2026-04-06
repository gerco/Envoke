package backend_test

import (
	"testing"

	"git.dries.info/gerco/envoke/internal/backend"
)

type fakeBackend struct{}

func (f *fakeBackend) Get(_, _ string) (string, error)  { return "val", nil }
func (f *fakeBackend) Set(_, _, _ string) error          { return nil }
func (f *fakeBackend) List(_ string) ([]string, error)   { return []string{"k"}, nil }

func TestRegisterAndNew(t *testing.T) {
	backend.Register("fake", func(_ map[string]string) (backend.Backend, error) {
		return &fakeBackend{}, nil
	})

	b, err := backend.New("fake", nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	val, err := b.Get("ns", "k")
	if err != nil || val != "val" {
		t.Errorf("Get: val=%q err=%v", val, err)
	}
}

func TestNew_UnknownBackend(t *testing.T) {
	_, err := backend.New("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}
