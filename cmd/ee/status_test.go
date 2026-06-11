package main

import (
	"strings"
	"testing"

	"git.dries.info/gerco/envoke/internal/config"
)

// ---------------------------------------------------------------------------
// Tests for checkNamespaceStatus
// ---------------------------------------------------------------------------

func TestCheckNamespaceStatus_OK(t *testing.T) {
	fake := newMemBackend()
	const backendKey = "test-status-ok"
	registerTestBackend(backendKey, fake)

	ns := config.NamespaceEntry{Name: "myns", Backend: backendKey}
	got := checkNamespaceStatus(ns)
	if !strings.HasPrefix(got, "✓") {
		t.Errorf("expected ok status (✓...), got %q", got)
	}
}

func TestCheckNamespaceStatus_UnknownBackend(t *testing.T) {
	ns := config.NamespaceEntry{Name: "myns", Backend: "test-status-notregistered-xyz"}
	got := checkNamespaceStatus(ns)
	if !strings.HasPrefix(got, "✗") {
		t.Errorf("expected error status (✗...), got %q", got)
	}
}
