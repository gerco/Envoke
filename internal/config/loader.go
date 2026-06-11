package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"git.dries.info/gerco/envoke/internal/backend"
	"gopkg.in/yaml.v3"
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

	encoder := yaml.NewEncoder(f)
	defer encoder.Close()
	return encoder.Encode(cfg)
}

const (
	dotfileName       = ".envoke"
	localOverrideName = ".envoke.local"
	globalConfigName  = "config.yaml"
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
	for name, bc := range global.Backends {
		backend.DefaultRegistry.RegisterExplicitConfig(name, bc.Type, bc.Config)
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
	expandEnvInGlobalConfig(&cfg)
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
//   - Linux: $XDG_CONFIG_HOME/envoke/config.yaml or ~/.config/envoke/config.yaml
//   - macOS: ~/Library/Application Support/envoke/config.yaml
//   - Windows: %APPDATA%\envoke\config.yaml
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
	expandEnvInDotFile(&df)
	return df, nil
}

// decodeFile parses a YAML file into v. If the file does not exist the call
// is a no-op and v is left at its zero value.
func decodeFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// merge overlays local on top of base, preserving insertion order.
// Local entries with the same name replace the base entry in-place;
// local-only entries are appended after all base entries.
func merge(base, local DotFile) []NamespaceEntry {
	result := make([]NamespaceEntry, len(base.Namespaces))
	copy(result, base.Namespaces)

	for _, localNS := range local.Namespaces {
		replaced := false
		for i, entry := range result {
			if entry.Name == localNS.Name {
				result[i] = localNS
				replaced = true
				break
			}
		}
		if !replaced {
			result = append(result, localNS)
		}
	}
	return result
}
