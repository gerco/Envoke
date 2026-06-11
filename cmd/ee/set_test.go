package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for readSecret (non-interactive stdin path)
// ---------------------------------------------------------------------------

func TestReadSecret_ReadsFromStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })

	fmt.Fprintln(w, "mypassword")
	w.Close()

	got, err := readSecret("prompt: ")
	if err != nil {
		t.Fatalf("readSecret: %v", err)
	}
	if got != "mypassword" {
		t.Errorf("got %q, want %q", got, "mypassword")
	}
}

// ---------------------------------------------------------------------------
// Tests for setCmd RunE
// ---------------------------------------------------------------------------

// setProjectDir saves the current projectDir, sets it to dir for the test,
// and restores the original on cleanup.
func withProjectDir(t *testing.T, dir string) {
	t.Helper()
	orig := projectDir
	projectDir = dir
	t.Cleanup(func() { projectDir = orig })
}

func TestSetCmd_WritesToBackendWithInlineValue(t *testing.T) {
	dir := t.TempDir()
	withProjectDir(t, dir)

	fake := newMemBackend()
	const backendKey = "test-set-inline"
	registerTestBackend(backendKey, fake)

	// Write a .envoke that maps the namespace to our fake backend.
	envoke := fmt.Sprintf("namespaces:\n  myns:\n    backend: %s\n", backendKey)
	if err := os.WriteFile(filepath.Join(dir, ".envoke"), []byte(envoke), 0644); err != nil {
		t.Fatal(err)
	}

	if err := setCmd.RunE(setCmd, []string{"myns", "DB_PASS", "s3cr3t"}); err != nil {
		t.Fatalf("setCmd.RunE: %v", err)
	}

	got, err := fake.Get("myns", "DB_PASS")
	if err != nil {
		t.Fatalf("Get after set: %v", err)
	}
	if got != "s3cr3t" {
		t.Errorf("got %q, want %q", got, "s3cr3t")
	}
}

func TestSetCmd_WithBackendFlag(t *testing.T) {
	dir := t.TempDir()
	withProjectDir(t, dir)

	fake := newMemBackend()
	const backendKey = "test-set-flag"
	registerTestBackend(backendKey, fake)

	// Set the --backend flag, reset after test.
	if err := setCmd.Flags().Set("backend", backendKey); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = setCmd.Flags().Set("backend", "") })

	if err := setCmd.RunE(setCmd, []string{"anyns", "API_KEY", "tok123"}); err != nil {
		t.Fatalf("setCmd.RunE: %v", err)
	}

	got, err := fake.Get("anyns", "API_KEY")
	if err != nil {
		t.Fatalf("Get after set: %v", err)
	}
	if got != "tok123" {
		t.Errorf("got %q, want %q", got, "tok123")
	}
}

func TestSetCmd_CreatesEnvokeFile(t *testing.T) {
	dir := t.TempDir()
	withProjectDir(t, dir)

	fake := newMemBackend()
	const backendKey = "test-set-creates-file"
	registerTestBackend(backendKey, fake)

	if err := setCmd.Flags().Set("backend", backendKey); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = setCmd.Flags().Set("backend", "") })

	// No .envoke file exists yet.
	if err := setCmd.RunE(setCmd, []string{"newns", "TOKEN", "abc"}); err != nil {
		t.Fatalf("setCmd.RunE: %v", err)
	}

	// .envoke should have been created with the namespace entry.
	data, err := os.ReadFile(filepath.Join(dir, ".envoke"))
	if err != nil {
		t.Fatalf("read .envoke: %v", err)
	}
	if len(data) == 0 {
		t.Error(".envoke file is empty, expected namespace entry")
	}
}
