package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for parseOptions
// ---------------------------------------------------------------------------

func TestParseOptions_Nil(t *testing.T) {
	got, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestParseOptions_ValidPairs(t *testing.T) {
	got, err := parseOptions([]string{"region=us-east-1", "prefix=prod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["region"] != "us-east-1" {
		t.Errorf("region = %q, want %q", got["region"], "us-east-1")
	}
	if got["prefix"] != "prod" {
		t.Errorf("prefix = %q, want %q", got["prefix"], "prod")
	}
}

func TestParseOptions_ValueContainsEquals(t *testing.T) {
	// Ensure only the first '=' is used as separator.
	got, err := parseOptions([]string{"token=abc=def"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["token"] != "abc=def" {
		t.Errorf("token = %q, want %q", got["token"], "abc=def")
	}
}

func TestParseOptions_MissingEquals_ReturnsError(t *testing.T) {
	_, err := parseOptions([]string{"noequals"})
	if err == nil {
		t.Error("expected error for missing '=', got nil")
	}
}

func TestParseOptions_EmptyKey_ReturnsError(t *testing.T) {
	_, err := parseOptions([]string{"=value"})
	if err == nil {
		t.Error("expected error for empty key, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests for backend subcommands (backendAddCmd, backendListCmd, etc.)
// These redirect the global config to a temp dir via APPDATA / XDG_CONFIG_HOME.
// ---------------------------------------------------------------------------

// globalConfigDir creates a temp dir, redirects the global config lookup to
// it, and returns the envoke subdirectory path (pre-created).
func globalConfigDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	setGlobalConfigDir(t, tmp)
	envokedir := filepath.Join(tmp, "envoke")
	if err := os.MkdirAll(envokedir, 0755); err != nil {
		t.Fatal(err)
	}
	return envokedir
}

func TestBackendAddCmd_AddsBackendToConfig(t *testing.T) {
	globalConfigDir(t)

	// Set the required flags directly (bypasses cobra's required-flag check).
	origType := addType
	addType = "aws"
	t.Cleanup(func() { addType = origType })

	origOpts := addOptions
	addOptions = []string{"region=eu-west-1"}
	t.Cleanup(func() { addOptions = origOpts })

	if err := backendAddCmd.RunE(backendAddCmd, []string{"prod-aws"}); err != nil {
		t.Fatalf("backendAddCmd.RunE: %v", err)
	}

	// Running again with the same name must return an error.
	err := backendAddCmd.RunE(backendAddCmd, []string{"prod-aws"})
	if err == nil {
		t.Error("expected error when adding duplicate backend, got nil")
	}
}

func TestBackendListCmd_IncludesImplicitBackends(t *testing.T) {
	globalConfigDir(t)

	out := captureStdout(t, func() {
		if err := backendListCmd.RunE(backendListCmd, nil); err != nil {
			t.Errorf("backendListCmd.RunE: %v", err)
		}
	})

	// The keychain backend is always compiled in as an implicit backend.
	if !strings.Contains(out, "keychain") {
		t.Errorf("expected 'keychain' in backend list output; got:\n%s", out)
	}
}

func TestBackendDisableCmd_DisablesImplicitBackend(t *testing.T) {
	globalConfigDir(t)

	if err := backendDisableCmd.RunE(backendDisableCmd, []string{"keychain"}); err != nil {
		t.Fatalf("backendDisableCmd.RunE: %v", err)
	}

	// Disabling again should report already-disabled, not error.
	if err := backendDisableCmd.RunE(backendDisableCmd, []string{"keychain"}); err != nil {
		t.Fatalf("second disable: %v", err)
	}
}

func TestBackendEnableCmd_EnablesDisabledBackend(t *testing.T) {
	globalConfigDir(t)

	// Disable first, then re-enable.
	if err := backendDisableCmd.RunE(backendDisableCmd, []string{"keychain"}); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if err := backendEnableCmd.RunE(backendEnableCmd, []string{"keychain"}); err != nil {
		t.Fatalf("enable: %v", err)
	}
}
