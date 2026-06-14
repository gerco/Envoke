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
		return nil, fmt.Errorf("project config: %w", err)
	}

	local, err := loadDotfile(filepath.Join(projectDir, localOverrideName))
	if err != nil {
		return nil, fmt.Errorf("local override: %w", err)
	}

	return &Loaded{
		Global:     global,
		Namespaces: merge(base, local),
		Verbose:    global.Verbose || base.Verbose || local.Verbose,
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
		return GlobalConfig{}, fmt.Errorf("%s: %w", path, err)
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
		return DotFile{}, fmt.Errorf("%s: %w", path, err)
	}
	expandEnvInDotFile(&df)
	applyDotfileDefaults(&df)
	return df, nil
}

// applyDotfileDefaults fills in omitted fields with their default values.
// Namespaces without an explicit backend default to "keychain".
// For shell namespaces (inline "shell" or an explicit backend of type "shell")
// that carry a vars map, a synthetic per-namespace explicit config is registered
// so the runner can resolve it by name. The explicit backend's options (e.g. the
// "shell" invocation prefix) are inherited and the namespace vars are merged in.
func applyDotfileDefaults(df *DotFile) {
	for i := range df.Namespaces {
		ns := &df.Namespaces[i]
		if ns.Backend == "" {
			ns.Backend = "keychain"
		}
		if len(ns.Vars) > 0 {
			if baseOpts := shellBaseOpts(ns.Backend); baseOpts != nil {
				opts := make(map[string]string, len(baseOpts)+len(ns.Vars))
				for k, v := range baseOpts {
					opts[k] = v
				}
				for k, v := range ns.Vars {
					opts[k] = v
				}
				syntheticName := "shell:" + ns.Name
				backend.DefaultRegistry.RegisterExplicitConfig(syntheticName, "shell", opts)
				ns.Backend = syntheticName
			}
		}
	}
}

// shellBaseOpts returns the base options for a shell backend reference, or nil
// if the backend name does not refer to a shell-type backend.
// "shell" (the built-in name) returns an empty map so vars are used as-is.
// An explicit backend of type "shell" returns its configured options (e.g.
// a custom "shell" invocation prefix) so they are inherited by the namespace.
func shellBaseOpts(backendName string) map[string]string {
	if backendName == "shell" {
		return map[string]string{}
	}
	cfg, ok := backend.DefaultRegistry.GetExplicitConfig(backendName)
	if !ok || cfg.Type != "shell" {
		return nil
	}
	return cfg.Options
}

// decodeFile parses a YAML file into v. If the file does not exist the call
// is a no-op and v is left at its zero value. Errors do not include the file
// path — callers are expected to wrap with it for context.
func decodeFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	return yaml.Unmarshal(data, v)
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
