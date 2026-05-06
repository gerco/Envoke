package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"git.dries.info/gerco/envoke/internal/backend"
	"github.com/BurntSushi/toml"
)

// LoadGlobal reads only the global config without loading project dotfiles.
func LoadGlobal() (GlobalConfig, error) {
	return loadGlobal()
}

// SaveGlobal writes cfg to the global config file, creating it if needed.
func SaveGlobal(cfg GlobalConfig) error {
	path, err := GlobalConfigPath()
	if err != nil {
		return fmt.Errorf("cannot determine config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

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
	backend.DefaultRegistry.SetDisabled(global.DisabledImplicitBackends)

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

// loadGlobal reads the global config file using OS-appropriate paths
// (XDG on Linux, Application Support on macOS, APPDATA on Windows).
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
// Platform-specific paths:
//   - Linux: $XDG_CONFIG_HOME/envoke/config.toml or ~/.config/envoke/config.toml
//   - macOS: ~/Library/Application Support/envoke/config.toml
//   - Windows: %APPDATA%\envoke\config.toml
func GlobalConfigPath() (string, error) {
	dir, err := defaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, globalConfigName), nil
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
