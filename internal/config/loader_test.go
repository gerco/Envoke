package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setGlobalConfigDir redirects the global config lookup to dir for the duration
// of the test, in a platform-correct way.
func setGlobalConfigDir(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", dir)
	} else {
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
}

func writeTOML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoad_NoDotfile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 0 {
		t.Errorf("expected 0 namespaces, got %d", len(cfg.Namespaces))
	}
}

func TestLoad_DotfileOnly(t *testing.T) {
	dir := t.TempDir()
	writeTOML(t, dir, ".envoke", `
[[namespace]]
name = "aws-dev"
backend = "keeper"

[[namespace]]
name = "db-local"
backend = "keychain"
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(cfg.Namespaces))
	}
	if cfg.Namespaces[0].Name != "aws-dev" || cfg.Namespaces[0].Backend != "keeper" {
		t.Errorf("unexpected namespace[0]: %+v", cfg.Namespaces[0])
	}
	if cfg.Namespaces[1].Name != "db-local" || cfg.Namespaces[1].Backend != "keychain" {
		t.Errorf("unexpected namespace[1]: %+v", cfg.Namespaces[1])
	}
}

func TestLoad_LocalOverrideReplaces(t *testing.T) {
	dir := t.TempDir()
	writeTOML(t, dir, ".envoke", `
[[namespace]]
name = "aws-dev"
backend = "keeper"
`)
	// Local overrides the same namespace with a different backend.
	writeTOML(t, dir, ".envoke.local", `
[[namespace]]
name = "aws-dev"
backend = "keychain"
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 1 {
		t.Fatalf("expected 1 namespace, got %d", len(cfg.Namespaces))
	}
	if cfg.Namespaces[0].Backend != "keychain" {
		t.Errorf("expected local override to win, got backend=%q", cfg.Namespaces[0].Backend)
	}
}

func TestLoad_LocalOverrideAppends(t *testing.T) {
	dir := t.TempDir()
	writeTOML(t, dir, ".envoke", `
[[namespace]]
name = "aws-dev"
backend = "keeper"
`)
	writeTOML(t, dir, ".envoke.local", `
[[namespace]]
name = "db-personal"
backend = "keychain"
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 2 {
		t.Fatalf("expected 2 namespaces after append, got %d", len(cfg.Namespaces))
	}
	if cfg.Namespaces[1].Name != "db-personal" {
		t.Errorf("expected appended namespace, got %q", cfg.Namespaces[1].Name)
	}
}

func TestMerge_EmptyLocal(t *testing.T) {
	base := DotFile{Namespaces: []Namespace{{Name: "a", Backend: "x"}}}
	result := merge(base, DotFile{})
	if len(result) != 1 || result[0].Name != "a" {
		t.Errorf("unexpected merge result: %+v", result)
	}
}

func TestLoad_NoDefaultLocalBackendInjected(t *testing.T) {
	dir := t.TempDir()
	// No global config, no dotfile — no default 'local' backend should be injected.
	// The keychain implicit backend is available via the registry instead.
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bc := cfg.Global.BackendByName("local")
	if bc != nil {
		t.Error("expected 'local' backend to NOT be injected by default (use implicit 'keychain' instead)")
	}
}

func TestLoad_LocalBackendNotOverriddenByDefault(t *testing.T) {
	dir := t.TempDir()
	// If the user declares their own "local" backend, it should not be replaced.
	globalDir := t.TempDir()
	setGlobalConfigDir(t, globalDir)
	if err := os.MkdirAll(globalDir+"/envoke", 0o700); err != nil {
		t.Fatal(err)
	}
	writeTOML(t, globalDir+"/envoke", "config.toml", `
[[backend]]
name = "local"
type = "keeper"
`)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bc := cfg.Global.BackendByName("local")
	if bc == nil || bc.Type != "keeper" {
		t.Errorf("expected user-defined local=keeper to win, got %v", bc)
	}
}

func TestBackendByName(t *testing.T) {
	g := &GlobalConfig{
		Backends: []BackendConfig{
			{Name: "my-keeper", Type: "keeper"},
			{Name: "my-chain", Type: "keychain"},
		},
	}
	if bc := g.BackendByName("my-keeper"); bc == nil || bc.Type != "keeper" {
		t.Errorf("expected to find my-keeper, got %v", bc)
	}
	if bc := g.BackendByName("missing"); bc != nil {
		t.Errorf("expected nil for missing backend, got %v", bc)
	}
}
