package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"git.dries.info/gerco/envoke/internal/config"
)

// captureStdout runs fn while capturing everything written to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	r.Close()
	return buf.String()
}

// ---------------------------------------------------------------------------
// Tests for listOne
// ---------------------------------------------------------------------------

func TestListOne_PrintsSortedKeys(t *testing.T) {
	fake := newMemBackend()
	fake.data["myns"] = map[string]string{"ZEBRA": "z", "ALPHA": "a", "MIDDLE": "m"}
	registerTestBackend("test-listOne-sorted", fake)

	cfg := loadedConfig("myns", "test-listOne-sorted")

	out := captureStdout(t, func() {
		if err := listOne(cfg, "myns"); err != nil {
			t.Errorf("listOne: %v", err)
		}
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	want := []string{"ALPHA", "MIDDLE", "ZEBRA"}
	if len(lines) != len(want) {
		t.Fatalf("got lines %v, want %v", lines, want)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d: got %q, want %q", i, lines[i], w)
		}
	}
}

func TestListOne_Empty(t *testing.T) {
	registerTestBackend("test-listOne-empty", newMemBackend())
	cfg := loadedConfig("myns", "test-listOne-empty")

	out := captureStdout(t, func() {
		if err := listOne(cfg, "myns"); err != nil {
			t.Errorf("listOne: %v", err)
		}
	})
	if out != "" {
		t.Errorf("expected no output, got %q", out)
	}
}

func TestListOne_BackendError_ReturnsError(t *testing.T) {
	// Namespace resolves to a backend that doesn't exist → Resolve returns an error.
	cfg := loadedConfig("myns", "test-listOne-nonexistent-xyz")

	err := listOne(cfg, "myns")
	if err == nil {
		t.Error("expected error when backend not found, got nil")
	}
}

func TestListOne_NoEnvoke_DefaultsToKeychain(t *testing.T) {
	// getBackendName returns "keychain" for any namespace not in cfg.
	cfg := &config.Loaded{}
	got := getBackendName(cfg, "any-ns")
	if got != "keychain" {
		t.Errorf("expected default backend %q, got %q", "keychain", got)
	}
}

// ---------------------------------------------------------------------------
// Tests for listNamespace (explicit backend override path)
// ---------------------------------------------------------------------------

func TestListNamespace_PrintsSortedKeys(t *testing.T) {
	fake := newMemBackend()
	fake.data["myns"] = map[string]string{"ZEBRA": "z", "ALPHA": "a"}
	registerTestBackend("test-listNamespace-sorted", fake)

	out := captureStdout(t, func() {
		if err := listNamespace(fake, "myns"); err != nil {
			t.Errorf("listNamespace: %v", err)
		}
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	want := []string{"ALPHA", "ZEBRA"}
	if len(lines) != len(want) {
		t.Fatalf("got lines %v, want %v", lines, want)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d: got %q, want %q", i, lines[i], w)
		}
	}
}

func TestListNamespace_OverridesConfiguredBackend(t *testing.T) {
	overrideFake := newMemBackend()
	overrideFake.data["myns"] = map[string]string{"OVERRIDE_KEY": "v"}
	registerTestBackend("test-listNS-override", overrideFake)

	// Passing the override backend directly bypasses whatever .envoke says.
	out := captureStdout(t, func() {
		if err := listNamespace(overrideFake, "myns"); err != nil {
			t.Errorf("listNamespace: %v", err)
		}
	})

	if !strings.Contains(out, "OVERRIDE_KEY") {
		t.Errorf("expected OVERRIDE_KEY in output; got:\n%s", out)
	}
}

func TestListNamespace_Empty(t *testing.T) {
	out := captureStdout(t, func() {
		if err := listNamespace(newMemBackend(), "myns"); err != nil {
			t.Errorf("listNamespace: %v", err)
		}
	})
	if out != "" {
		t.Errorf("expected no output, got %q", out)
	}
}

// ---------------------------------------------------------------------------
// Tests for listAll
// ---------------------------------------------------------------------------

func TestListAll_NoNamespaces(t *testing.T) {
	cfg := &config.Loaded{} // no namespaces
	// Should return nil without panicking (prints to stderr).
	if err := listAll(cfg); err != nil {
		t.Errorf("listAll: %v", err)
	}
}

func TestListAll_PrintsKeysByNamespace(t *testing.T) {
	fake := newMemBackend()
	fake.data["nsA"] = map[string]string{"KEY1": "v1", "KEY2": "v2"}
	registerTestBackend("test-listAll-basic", fake)

	cfg := &config.Loaded{
		Namespaces: []config.NamespaceEntry{
			{Name: "nsA", Backend: "test-listAll-basic"},
		},
	}

	out := captureStdout(t, func() {
		if err := listAll(cfg); err != nil {
			t.Errorf("listAll: %v", err)
		}
	})

	if !strings.Contains(out, "nsA") {
		t.Errorf("output missing namespace name; got:\n%s", out)
	}
	if !strings.Contains(out, "KEY1") || !strings.Contains(out, "KEY2") {
		t.Errorf("output missing keys; got:\n%s", out)
	}
}

func TestListAll_EmptyNamespace_PrintsEmptyMarker(t *testing.T) {
	registerTestBackend("test-listAll-emptyns", newMemBackend())

	cfg := &config.Loaded{
		Namespaces: []config.NamespaceEntry{
			{Name: "emptyns", Backend: "test-listAll-emptyns"},
		},
	}

	out := captureStdout(t, func() {
		if err := listAll(cfg); err != nil {
			t.Errorf("listAll: %v", err)
		}
	})

	if !strings.Contains(out, "(empty)") {
		t.Errorf("expected (empty) marker; got:\n%s", out)
	}
}

func TestListAll_BackendError_PrintsErrorInline(t *testing.T) {
	// Namespace points to an unregistered backend → openBackend returns error.
	cfg := &config.Loaded{
		Namespaces: []config.NamespaceEntry{
			{Name: "badns", Backend: fmt.Sprintf("test-listAll-unregistered-%d", os.Getpid())},
		},
	}

	out := captureStdout(t, func() {
		// listAll prints backend errors inline rather than returning them.
		if err := listAll(cfg); err != nil {
			t.Errorf("listAll returned unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "error") {
		t.Errorf("expected inline error in output; got:\n%s", out)
	}
}
