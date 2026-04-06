package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

const (
	dotfileName      = ".envoke"
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

// loadGlobal reads ~/.config/envoke/config.toml (XDG on Linux/macOS,
// %APPDATA%\envoke\config.toml on Windows).
func loadGlobal() (GlobalConfig, error) {
	path, err := globalConfigPath()
	if err != nil {
		return GlobalConfig{}, err
	}

	var cfg GlobalConfig
	if err := decodeFile(path, &cfg); err != nil {
		return GlobalConfig{}, err
	}
	return cfg, nil
}

func globalConfigPath() (string, error) {
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
