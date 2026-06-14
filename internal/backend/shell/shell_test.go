package shell

import (
	"testing"
)

func defaultBackend(vars map[string]string) *shellBackend {
	return &shellBackend{shellArgs: defaultShellArgs(), vars: vars}
}

func TestShellBackend_GetAndList(t *testing.T) {
	b := defaultBackend(map[string]string{
		"FOO": "echo hello",
		"BAR": "printf world",
	})

	keys, err := b.List("ns")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("List: expected 2 keys, got %d", len(keys))
	}

	val, err := b.Get("ns", "FOO")
	if err != nil {
		t.Fatalf("Get FOO: %v", err)
	}
	if val != "hello" {
		t.Errorf("Get FOO: expected %q, got %q", "hello", val)
	}

	val, err = b.Get("ns", "BAR")
	if err != nil {
		t.Fatalf("Get BAR: %v", err)
	}
	if val != "world" {
		t.Errorf("Get BAR: expected %q, got %q", "world", val)
	}
}

func TestShellBackend_GetNotFound(t *testing.T) {
	b := defaultBackend(map[string]string{})
	_, err := b.Get("ns", "MISSING")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestShellBackend_GetTrimsWhitespace(t *testing.T) {
	b := defaultBackend(map[string]string{
		"KEY": "printf '  trimmed  '",
	})
	val, err := b.Get("ns", "KEY")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "trimmed" {
		t.Errorf("expected trimmed value, got %q", val)
	}
}

func TestShellBackend_GetCommandFailure(t *testing.T) {
	b := defaultBackend(map[string]string{
		"KEY": "exit 1",
	})
	_, err := b.Get("ns", "KEY")
	if err == nil {
		t.Fatal("expected error for failing command")
	}
}

func TestShellBackend_SetReadOnly(t *testing.T) {
	b := defaultBackend(map[string]string{})
	if err := b.Set("ns", "KEY", "val"); err == nil {
		t.Fatal("expected error: shell backend is read-only")
	}
}

func TestShellBackend_CustomShell(t *testing.T) {
	b := &shellBackend{
		shellArgs: []string{"/bin/bash", "-c"},
		vars:      map[string]string{"KEY": "echo bash"},
	}
	val, err := b.Get("ns", "KEY")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "bash" {
		t.Errorf("expected %q, got %q", "bash", val)
	}
}

func TestShellBackend_FactoryParsesShellOption(t *testing.T) {
	opts := map[string]string{
		shellKey: "/bin/bash -c",
		"FOO":    "echo bar",
	}
	b, err := newBackend(opts)
	if err != nil {
		t.Fatalf("newBackend: %v", err)
	}
	sb := b.(*shellBackend)
	if len(sb.shellArgs) != 2 || sb.shellArgs[0] != "/bin/bash" || sb.shellArgs[1] != "-c" {
		t.Errorf("unexpected shellArgs: %v", sb.shellArgs)
	}
	if _, ok := sb.vars[shellKey]; ok {
		t.Error("shellKey should not appear in vars")
	}
	if sb.vars["FOO"] != "echo bar" {
		t.Errorf("expected vars[FOO]=%q, got %q", "echo bar", sb.vars["FOO"])
	}
}
