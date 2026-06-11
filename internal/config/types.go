// Package config handles loading and merging of the three config layers:
//
// Global config (OS-specific paths — describes backends):
//   - Linux: $XDG_CONFIG_HOME/envoke/config.yaml or ~/.config/envoke/config.yaml
//   - macOS: ~/Library/Application Support/envoke/config.yaml
//   - Windows: %APPDATA%\envoke\config.yaml
//
// Project configs:
//
//	<project>/.envoke             (dotfile — what the project needs)
//	<project>/.envoke.local       (local overrides — gitignored)
package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// DotFile is the parsed representation of .envoke / .envoke.local.
// Namespaces are a YAML sequence so that insertion order is preserved for
// backend chaining.
type DotFile struct {
	Namespaces []NamespaceEntry `yaml:"namespaces,omitempty"`
}

// BackendConfig holds the configuration for a single backend instance as
// declared in the global config.
// The YAML format is flat - all fields except "type" go into Config.
type BackendConfig struct {
	Type   string            `yaml:"type"`
	Config map[string]string `yaml:"-"` // Populated by UnmarshalYAML from all non-type fields
}

// UnmarshalYAML implements custom unmarshaling to extract "type" separately
// and put all other fields into the Config map.
func (bc *BackendConfig) UnmarshalYAML(node *yaml.Node) error {
	// Decode into a map to get all fields
	var raw map[string]interface{}
	if err := node.Decode(&raw); err != nil {
		return err
	}

	// Extract type
	if typeVal, ok := raw["type"]; ok {
		if typeStr, ok := typeVal.(string); ok {
			bc.Type = typeStr
		}
		delete(raw, "type")
	}

	// Convert remaining fields to map[string]string
	bc.Config = make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			bc.Config[k] = s
		} else {
			bc.Config[k] = fmt.Sprintf("%v", v)
		}
	}

	return nil
}

// MarshalYAML implements custom marshaling to write Config fields inline with Type.
// This produces the flat format: type: aws, region: us-east-1 (not nested under config:).
func (bc BackendConfig) MarshalYAML() (interface{}, error) {
	// Create a map with Type plus all Config entries
	result := make(map[string]string, len(bc.Config)+1)
	result["type"] = bc.Type
	for k, v := range bc.Config {
		result[k] = v
	}
	return result, nil
}

// Defaults holds default configuration values.
type Defaults struct {
	// NamespaceBackend is the fallback backend when .envoke doesn't specify one.
	NamespaceBackend string `yaml:"namespace_backend,omitempty"`
}

// GlobalConfig is the parsed representation of ~/.config/envoke/config.yaml.
type GlobalConfig struct {
	Backends                 map[string]BackendConfig `yaml:"backends,omitempty"`
	Defaults                 Defaults                 `yaml:"defaults,omitempty"`
	DisabledImplicitBackends []string                 `yaml:"disabled_implicit_backends,omitempty"`
}

// BackendByName returns the BackendConfig whose key matches name, or nil.
func (g *GlobalConfig) BackendByName(name string) *BackendConfig {
	if bc, ok := g.Backends[name]; ok {
		return &bc
	}
	return nil
}

// Loaded is the fully merged, ready-to-use configuration for a single
// invocation of ee.
type Loaded struct {
	Global     GlobalConfig
	Namespaces []NamespaceEntry // merged dotfile + local overrides
}

// NamespaceEntry represents one namespace entry, both in the on-disk sequence
// format and in the merged runtime config.
type NamespaceEntry struct {
	Name    string            `yaml:"name"`
	Backend string            `yaml:"backend,omitempty"`
	Options map[string]string `yaml:"options,omitempty"`
}
