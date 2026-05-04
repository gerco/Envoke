//go:build keeper
// +build keeper

package keeper

import (
	"errors"
	"testing"

	"git.dries.info/gerco/envoke/internal/backend"
)

func TestBackendReturnsNotImplemented(t *testing.T) {
	b := &keeperBackend{}

	_, err := b.Get("test", "key")
	if !errors.Is(err, backend.ErrNotImplemented) {
		t.Errorf("Get: want ErrNotImplemented, got %v", err)
	}

	err = b.Set("test", "key", "value")
	if !errors.Is(err, backend.ErrNotImplemented) {
		t.Errorf("Set: want ErrNotImplemented, got %v", err)
	}

	_, err = b.List("test")
	if !errors.Is(err, backend.ErrNotImplemented) {
		t.Errorf("List: want ErrNotImplemented, got %v", err)
	}
}
