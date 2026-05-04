package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"git.dries.info/gerco/envoke/internal/backend"
	"github.com/BurntSushi/toml"
)

const (
	dotfileName       = ".envoke"
	localOverrideName = ".envoke.local"
	globalConfigName  = "config.toml"
	appDirName        = "envoke"
)

// Load reads all three config layers starting from projectDir and returns a
// merged Loaded config. Missing files are silently ignored; parse errors are
// returned immediately.
func Load(projectDir string) (*Loaded, error) {
	global, err := loadGlobal()
	if err != nil {
		return nil, fmt.Errorf("global config: %w", err)
	}

	// Register explicit backends from global config with the registry
	// This is done lazily - backends are created on first Resolve() call
	registerExplicitBackendConfigs(global)

	base, err := loadDotfile(filepath.Join(projectDir, dotfileName))
	if err != nil {
		return nil, fmt.Errorf("dotfile: %w", err)
	}

	local, err := loadDotfile(filepath.Join(projectDir, localOverrideName))
	if err != nil {
		return nil, fmt.Errorf("local override: %w", err)
	}

	return &Loaded{
		Global:     global,
		Namespaces: merge(base, local),
	}, nil
}

// registerExplicitBackendConfigs stores backend configs in the registry for lazy creation.
// This allows config loading to succeed even if backend packages aren't imported yet.
func registerExplicitBackendConfigs(global GlobalConfig) {
	for _, bc := range global.Backends {
		backend.DefaultRegistry.RegisterExplicitConfig(bc.Name, bc.Type, bc.Options)
	}
}

// loadGlobal reads ~/.config/envoke/config.toml (XDG on Linux/macOS,
// %APPDATA%\envoke\config.toml on Windows).
func loadGlobal() (GlobalConfig, error) {
	path, err := GlobalConfigPath()
	if err != nil {
		return GlobalConfig{}, err
	}

	var cfg GlobalConfig
	if err := decodeFile(path, &cfg); err != nil {
		return GlobalConfig{}, err
	}
	applyDefaults(&cfg)
	return cfg, nil
}

// applyDefaults injects built-in backend configurations.
// This is now empty since keychain is always available as an implicit default backend.
func applyDefaults(cfg *GlobalConfig) {
	// No defaults needed - keychain backend is always available via the registry.
}

// GlobalConfigPath returns the path to the global configuration file.
// On Windows: %APPDATA%\envoke\config.toml
// On Unix: $XDG_CONFIG_HOME/envoke/config.toml or ~/.config/envoke/config.toml
func GlobalConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("%%APPDATA%% not set")
		}
		return filepath.Join(appData, appDirName, globalConfigName), nil
	}

	// XDG_CONFIG_HOME takes precedence, then ~/.config
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, appDirName, globalConfigName), nil
}

func loadDotfile(path string) (DotFile, error) {
	var df DotFile
	if err := decodeFile(path, &df); err != nil {
		return DotFile{}, err
	}
	return df, nil
}

// decodeFile parses a TOML file into v. If the file does not exist the call
// is a no-op and v is left at its zero value.
func decodeFile(path string, v any) error {
	_, err := toml.DecodeFile(path, v)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// merge overlays local on top of base. Namespaces with the same name in local
// replace those in base; namespaces only in local are appended.
func merge(base, local DotFile) []Namespace {
	out := make([]Namespace, len(base.Namespaces))
	copy(out, base.Namespaces)

	for _, ln := range local.Namespaces {
		found := false
		for i, bn := range out {
			if bn.Name == ln.Name {
				out[i] = ln
				found = true
				break
			}
		}
		if !found {
			out = append(out, ln)
		}
	}
	return out
}
