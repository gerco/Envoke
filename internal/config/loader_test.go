package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setGlobalConfigDir redirects the global config lookup to dir for the duration
// of the test, in a platform-correct way.
//
// Platform-specific behavior:
//   - Linux: sets XDG_CONFIG_HOME
//   - macOS: sets HOME (since macOS uses ~/Library/Application Support)
//   - Windows: sets APPDATA
func setGlobalConfigDir(t *testing.T, dir string) {
	t.Helper()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("APPDATA", dir)
	case "darwin":
		// On macOS, the path is derived from $HOME/Library/Application Support
		// We can't easily change just the config dir, so we test path structure instead
		// Tests that need to override should use platform-specific integration tests
	default:
		// Linux and other Unix: use XDG_CONFIG_HOME
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
}

func writeYAML(t *testing.T, dir, name, content string) {
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
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: aws-dev
    backend: keeper
  - name: db-local
    backend: keychain
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(cfg.Namespaces))
	}

	if cfg.Namespaces[0].Name != "aws-dev" || cfg.Namespaces[0].Backend != "keeper" {
		t.Errorf("expected [0]=aws-dev/keeper, got %+v", cfg.Namespaces[0])
	}
	if cfg.Namespaces[1].Name != "db-local" || cfg.Namespaces[1].Backend != "keychain" {
		t.Errorf("expected [1]=db-local/keychain, got %+v", cfg.Namespaces[1])
	}
}

func TestLoad_LocalOverrideReplaces(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: aws-dev
    backend: keeper
`)
	// Local overrides the same namespace with a different backend.
	writeYAML(t, dir, ".envoke.local", `
namespaces:
  - name: aws-dev
    backend: keychain
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
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: aws-dev
    backend: keeper
`)
	writeYAML(t, dir, ".envoke.local", `
namespaces:
  - name: db-personal
    backend: keychain
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 2 {
		t.Fatalf("expected 2 namespaces after append, got %d", len(cfg.Namespaces))
	}

	// Base entry comes first; local-only entry is appended.
	if cfg.Namespaces[0].Name != "aws-dev" {
		t.Errorf("expected [0]=aws-dev, got %q", cfg.Namespaces[0].Name)
	}
	if cfg.Namespaces[1].Name != "db-personal" {
		t.Errorf("expected [1]=db-personal, got %q", cfg.Namespaces[1].Name)
	}
}

func TestMerge_EmptyLocal(t *testing.T) {
	base := DotFile{Namespaces: []NamespaceEntry{{Name: "a", Backend: "x"}}}
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
	writeYAML(t, globalDir+"/envoke", "config.yaml", `
backends:
  local:
    type: keeper
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

func TestLoad_OrderPreserved(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: first
    backend: keychain
  - name: second
    backend: keychain
  - name: third
    backend: keychain
`)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"first", "second", "third"}
	for i, w := range want {
		if cfg.Namespaces[i].Name != w {
			t.Errorf("position %d: got %q, want %q", i, cfg.Namespaces[i].Name, w)
		}
	}
}

func TestLoad_LocalOverrideReplacesInPlace(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: first
    backend: keychain
  - name: second
    backend: keychain
`)
	// Local overrides "first" — it should stay at position 0, not move to the end.
	writeYAML(t, dir, ".envoke.local", `
namespaces:
  - name: first
    backend: aws
`)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(cfg.Namespaces))
	}
	if cfg.Namespaces[0].Name != "first" || cfg.Namespaces[0].Backend != "aws" {
		t.Errorf("expected [0]=first/aws (in-place override), got %+v", cfg.Namespaces[0])
	}
	if cfg.Namespaces[1].Name != "second" {
		t.Errorf("expected [1]=second, got %+v", cfg.Namespaces[1])
	}
}

func TestBackendByName(t *testing.T) {
	g := &GlobalConfig{
		Backends: map[string]BackendConfig{
			"my-keeper": {Type: "keeper"},
			"my-chain":  {Type: "keychain"},
		},
	}
	if bc := g.BackendByName("my-keeper"); bc == nil || bc.Type != "keeper" {
		t.Errorf("expected to find my-keeper, got %v", bc)
	}
	if bc := g.BackendByName("missing"); bc != nil {
		t.Errorf("expected nil for missing backend, got %v", bc)
	}
}

func TestGlobalConfigPath_PlatformSpecific(t *testing.T) {
	path, err := GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath() error = %v", err)
	}

	// Verify path ends with config.yaml
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("expected path to end with config.yaml, got %s", filepath.Base(path))
	}

	// Verify platform-specific path structure
	switch runtime.GOOS {
	case "windows":
		// Windows: should contain AppData\Roaming\envoke
		if !contains(path, "AppData") && !contains(path, "envoke") {
			t.Errorf("Windows path should contain AppData and envoke, got: %s", path)
		}
	case "darwin":
		// macOS: should contain Library/Application Support/envoke
		if !contains(path, "Library") || !contains(path, "Application Support") {
			t.Errorf("macOS path should contain Library/Application Support, got: %s", path)
		}
	default:
		// Linux: should contain .config/envoke or respect XDG_CONFIG_HOME
		if !contains(path, "envoke") {
			t.Errorf("Linux path should contain envoke directory, got: %s", path)
		}
	}
}

func TestGlobalConfigPath_Linux_XDGConfigHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping: Linux-only test")
	}

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath() error = %v", err)
	}

	expected := filepath.Join(dir, "envoke", "config.yaml")
	if path != expected {
		t.Errorf("XDG_CONFIG_HOME not respected: got %s, want %s", path, expected)
	}
}

func TestGlobalConfigPath_Windows_AppData(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping: Windows-only test")
	}

	dir := t.TempDir()
	t.Setenv("APPDATA", dir)

	path, err := GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath() error = %v", err)
	}

	expected := filepath.Join(dir, "envoke", "config.yaml")
	if path != expected {
		t.Errorf("APPDATA not respected: got %s, want %s", path, expected)
	}
}

func TestGlobalConfigPath_Windows_NoAppData(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping: Windows-only test")
	}

	// Clear APPDATA
	t.Setenv("APPDATA", "")

	_, err := GlobalConfigPath()
	if err == nil {
		t.Error("expected error when APPDATA is not set, got nil")
	}
}

// contains reports whether substr is within s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestLoad_NoBackendDefaultsToKeychain(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: ns1
  - name: ns2
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(cfg.Namespaces))
	}
	for _, ns := range cfg.Namespaces {
		if ns.Backend != "keychain" {
			t.Errorf("namespace %q: expected backend=keychain, got %q", ns.Name, ns.Backend)
		}
	}
}

func TestLoad_MixedBackends(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: local-ns
  - name: aws-ns
    backend: aws
  - name: another-local
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 3 {
		t.Fatalf("expected 3 namespaces, got %d", len(cfg.Namespaces))
	}
	if cfg.Namespaces[0].Backend != "keychain" {
		t.Errorf("[0] expected keychain, got %q", cfg.Namespaces[0].Backend)
	}
	if cfg.Namespaces[1].Backend != "aws" {
		t.Errorf("[1] expected aws, got %q", cfg.Namespaces[1].Backend)
	}
	if cfg.Namespaces[2].Backend != "keychain" {
		t.Errorf("[2] expected keychain, got %q", cfg.Namespaces[2].Backend)
	}
}

func TestEnvExpansion_InDotfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TEST_BACKEND", "keychain")
	writeYAML(t, dir, ".envoke", `
namespaces:
  - name: test-ns
    backend: "${TEST_BACKEND}"
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Namespaces) != 1 {
		t.Fatalf("expected 1 namespace, got %d", len(cfg.Namespaces))
	}
	if cfg.Namespaces[0].Backend != "keychain" {
		t.Errorf("expected backend to be expanded to 'keychain', got %q", cfg.Namespaces[0].Backend)
	}
}

func TestEnvExpansion_InGlobalConfig(t *testing.T) {
	globalDir := t.TempDir()
	setGlobalConfigDir(t, globalDir)
	t.Setenv("TEST_REGION", "us-west-2")

	if err := os.MkdirAll(globalDir+"/envoke", 0o700); err != nil {
		t.Fatal(err)
	}
	writeYAML(t, globalDir+"/envoke", "config.yaml", `
backends:
  aws-test:
    type: aws
    region: "${TEST_REGION}"
`)

	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bc := cfg.Global.BackendByName("aws-test")
	if bc == nil {
		t.Fatal("expected aws-test backend")
	}
	if bc.Config["region"] != "us-west-2" {
		t.Errorf("expected region to be expanded to 'us-west-2', got %q", bc.Config["region"])
	}
}
