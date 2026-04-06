package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureNamespace_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	added, err := EnsureNamespace(dir, "test-ns", "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected added=true for new namespace")
	}

	data, err := os.ReadFile(filepath.Join(dir, ".envoke"))
	if err != nil {
		t.Fatalf("expected .envoke to be created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `name = "test-ns"`) {
		t.Errorf("expected name in file, got:\n%s", content)
	}
	if !strings.Contains(content, `backend = "local"`) {
		t.Errorf("expected backend in file, got:\n%s", content)
	}
}

func TestEnsureNamespace_Idempotent(t *testing.T) {
	dir := t.TempDir()

	if _, err := EnsureNamespace(dir, "test-ns", "local"); err != nil {
		t.Fatal(err)
	}
	added, err := EnsureNamespace(dir, "test-ns", "local")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if added {
		t.Fatal("expected added=false when namespace already present")
	}

	// File should still be valid TOML with exactly one namespace.
	df, err := loadDotfile(filepath.Join(dir, ".envoke"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(df.Namespaces) != 1 {
		t.Errorf("expected 1 namespace, got %d", len(df.Namespaces))
	}
}

func TestEnsureNamespace_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	writeTOML(t, dir, ".envoke", `[[namespace]]
name = "existing"
backend = "keeper"
`)

	added, err := EnsureNamespace(dir, "new-ns", "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected added=true")
	}

	df, err := loadDotfile(filepath.Join(dir, ".envoke"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(df.Namespaces) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(df.Namespaces))
	}
	if df.Namespaces[0].Name != "existing" {
		t.Errorf("expected existing namespace first, got %q", df.Namespaces[0].Name)
	}
	if df.Namespaces[1].Name != "new-ns" {
		t.Errorf("expected new-ns second, got %q", df.Namespaces[1].Name)
	}
}
