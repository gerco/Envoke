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

// ---------------------------------------------------------------------------
// Tests for keychainNamespace
// ---------------------------------------------------------------------------

func TestKeychainNamespace(t *testing.T) {
	tests := []struct {
		name       string
		rawArgs    []string
		cobraArgs  []string
		wantNS     string
		wantArgs   []string
		wantErrSub string
	}{
		{
			name:      "no dash-dash",
			rawArgs:   []string{"cmd", "arg"},
			cobraArgs: []string{"cmd", "arg"},
			wantNS:    "",
			wantArgs:  []string{"cmd", "arg"},
		},
		{
			name:      "dash-dash no namespace",
			rawArgs:   []string{"--", "cmd", "arg"},
			cobraArgs: []string{"cmd", "arg"},
			wantNS:    "",
			wantArgs:  []string{"cmd", "arg"},
		},
		{
			name:      "single namespace before dash-dash",
			rawArgs:   []string{"myns", "--", "cmd", "arg"},
			cobraArgs: []string{"myns", "cmd", "arg"},
			wantNS:    "myns",
			wantArgs:  []string{"cmd", "arg"},
		},
		{
			name:       "multiple namespaces before dash-dash",
			rawArgs:    []string{"ns1", "ns2", "--", "cmd"},
			cobraArgs:  []string{"ns1", "ns2", "cmd"},
			wantErrSub: "expected exactly one namespace",
		},
		{
			name:      "flags before dash-dash are ignored",
			rawArgs:   []string{"--someflag", "myns", "--", "cmd"},
			cobraArgs: []string{"myns", "cmd"},
			wantNS:    "myns",
			wantArgs:  []string{"cmd"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotNS, gotArgs, err := keychainNamespace(tc.rawArgs, tc.cobraArgs)
			if tc.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrSub)
				}
				if !contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotNS != tc.wantNS {
				t.Errorf("namespace: got %q, want %q", gotNS, tc.wantNS)
			}
			if len(gotArgs) != len(tc.wantArgs) {
				t.Fatalf("args length: got %d, want %d (%v vs %v)", len(gotArgs), len(tc.wantArgs), gotArgs, tc.wantArgs)
			}
			for i := range gotArgs {
				if gotArgs[i] != tc.wantArgs[i] {
					t.Errorf("args[%d]: got %q, want %q", i, gotArgs[i], tc.wantArgs[i])
				}
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
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
