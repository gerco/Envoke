package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for parentDir
// ---------------------------------------------------------------------------

func TestParentDir(t *testing.T) {
	tests := []struct {
		name, path, want string
	}{
		{"unix nested", "/a/b/c", "/a/b"},
		{"unix two levels", "/a/b", "/a"},
		{"unix root child", "/a", "/"},
		{"unix root", "/", "/"},
		{"no separator", "abc", "abc"},
		{"windows nested", `C:\a\b`, `C:\a`},
		{"windows root child", `C:\a`, `C:`},
		{"trailing unix slash", "/a/b/", "/a/b"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parentDir(tc.path)
			if got != tc.want {
				t.Errorf("parentDir(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for findProjectRoot
// ---------------------------------------------------------------------------

// resolvedPath returns the real path of p, following symlinks.
// Used for cross-platform path comparisons in tests.
func resolvedPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return r
}

func TestFindProjectRoot_FindsEnvokeInCWD(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".envoke"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	got, err := findProjectRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolvedPath(t, got) != resolvedPath(t, dir) {
		t.Errorf("got %q, want %q", got, dir)
	}
}

func TestFindProjectRoot_FindsEnvokeInParent(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub", "nested")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".envoke"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(subdir)

	got, err := findProjectRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolvedPath(t, got) != resolvedPath(t, dir) {
		t.Errorf("got %q, want %q", got, dir)
	}
}

func TestFindProjectRoot_NoEnvoke_FallsBackToCWD(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	got, err := findProjectRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Without a .envoke file the function must return the CWD (no error).
	if got == "" {
		t.Error("expected a non-empty path, got empty string")
	}
}
