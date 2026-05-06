//go:build windows

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// defaultConfigDir returns the directory for global configuration on Windows.
// Uses the standard Windows APPDATA directory:
//
//	%APPDATA%\envoke
func defaultConfigDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("%%APPDATA%% not set")
	}
	return filepath.Join(appData, appDirName), nil
}
