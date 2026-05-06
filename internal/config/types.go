// Package config handles loading and merging of the three config layers:
//
// Global config (OS-specific paths — describes backends):
//   - Linux: $XDG_CONFIG_HOME/envoke/config.toml or ~/.config/envoke/config.toml
//   - macOS: ~/Library/Application Support/envoke/config.toml
//   - Windows: %APPDATA%\envoke\config.toml
//
// Project configs:
//	<project>/.envoke             (dotfile — what the project needs)
//	<project>/.envoke.local       (local overrides — gitignored)
package config

// Namespace is one entry in the project dotfile.
type Namespace struct {
	Name    string `toml:"name"`
	Backend string `toml:"backend"`
	// Options are backend-specific key/value pairs that can override the
	// global backend config for this namespace (e.g. a different vault path).
	Options map[string]string `toml:"options"`
}

// DotFile is the parsed representation of .envoke / .envoke.local.
type DotFile struct {
	Namespaces []Namespace `toml:"namespace"`
}

// BackendConfig holds the configuration for a single backend instance as
// declared in the global config.
type BackendConfig struct {
	Name    string            `toml:"name"`
	Type    string            `toml:"type"`
	Options map[string]string `toml:"options"`
}

// GlobalConfig is the parsed representation of ~/.config/envoke/config.toml.
type GlobalConfig struct {
	Backends                 []BackendConfig `toml:"backend"`
	DisabledImplicitBackends []string        `toml:"disabled_implicit_backends,omitempty"`
}

// BackendByName returns the BackendConfig whose Name matches name, or nil.
func (g *GlobalConfig) BackendByName(name string) *BackendConfig {
	for i := range g.Backends {
		if g.Backends[i].Name == name {
			return &g.Backends[i]
		}
	}
	return nil
}

// Loaded is the fully merged, ready-to-use configuration for a single
// invocation of ee.
type Loaded struct {
	Global     GlobalConfig
	Namespaces []Namespace // merged dotfile + local overrides
}
