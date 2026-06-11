package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureNamespace_NoFileNoCreate(t *testing.T) {
	dir := t.TempDir()
	added, err := EnsureNamespace(dir, "test-ns", "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added {
		t.Fatal("expected added=false when .envoke does not exist")
	}
	if _, err := os.Stat(filepath.Join(dir, ".envoke")); !os.IsNotExist(err) {
		t.Fatal("expected .envoke to not be created")
	}
}

func TestEnsureNamespace_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	// Create an empty .envoke to simulate an initialised-but-blank project file.
	f, err := os.Create(filepath.Join(dir, ".envoke"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	added, err := EnsureNamespace(dir, "test-ns", "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected added=true for new namespace in empty file")
	}

	data, err := os.ReadFile(filepath.Join(dir, ".envoke"))
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "name: test-ns") {
		t.Errorf("expected 'name: test-ns' in file, got:\n%s", content)
	}
	if !strings.Contains(content, "backend: local") {
		t.Errorf("expected backend in file, got:\n%s", content)
	}
}

func TestEnsureNamespace_Idempotent(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: test-ns
    backend: local
`)

	added, err := EnsureNamespace(dir, "test-ns", "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added {
		t.Fatal("expected added=false when namespace already present")
	}

	df, err := loadDotfile(filepath.Join(dir, ".envoke"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(df.Namespaces) != 1 {
		t.Errorf("expected 1 namespace, got %d", len(df.Namespaces))
	}
	if df.Namespaces[0].Name != "test-ns" {
		t.Errorf("expected test-ns namespace, got %q", df.Namespaces[0].Name)
	}
}

func TestEnsureNamespace_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: existing
    backend: keeper
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
	nsNames := map[string]bool{}
	for _, ns := range df.Namespaces {
		nsNames[ns.Name] = true
	}
	if !nsNames["existing"] {
		t.Errorf("expected existing namespace to still be present")
	}
	if !nsNames["new-ns"] {
		t.Errorf("expected new-ns namespace to be present")
	}
}
