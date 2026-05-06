//go:build linux || darwin

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// defaultConfigDir returns the directory for global configuration on Unix systems.
// Uses os.UserConfigDir() which properly handles platform-specific paths:
//   - Linux: $XDG_CONFIG_HOME or ~/.config
//   - macOS: ~/Library/Application Support
func defaultConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine user config directory: %w", err)
	}
	return filepath.Join(configDir, appDirName), nil
}
