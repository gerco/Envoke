package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureNamespace checks whether namespaceName is already declared in the
// project dotfile. If it is not, the namespace is appended to the file
// (creating it if necessary) using backendName as the backend.
//
// Returns true if the dotfile was modified, false if the namespace was already
// present.
func EnsureNamespace(projectDir, namespaceName, backendName string) (bool, error) {
	path := filepath.Join(projectDir, dotfileName)

	existing, err := loadDotfile(path)
	if err != nil {
		return false, fmt.Errorf("read dotfile: %w", err)
	}

	for _, ns := range existing.Namespaces {
		if ns.Name == namespaceName {
			return false, nil
		}
	}

	if err := appendNamespace(path, namespaceName, backendName); err != nil {
		return false, err
	}
	return true, nil
}

// appendNamespace writes a [[namespace]] block to path, creating the file if
// it does not exist.
func appendNamespace(path, namespaceName, backendName string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	// Add a blank line before the block when appending to a non-empty file.
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	prefix := ""
	if info.Size() > 0 {
		prefix = "\n"
	}

	_, err = fmt.Fprintf(f, "%s[[namespace]]\nname = %q\nbackend = %q\n", prefix, namespaceName, backendName)
	return err
}
