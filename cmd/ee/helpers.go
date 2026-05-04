package main

import (
	"maps"

	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

// getBackendName returns the backend name for namespaceName from the dotfile.
func getBackendName(cfg *config.Loaded, namespaceName string) string {
	for _, ns := range cfg.Namespaces {
		if ns.Name == namespaceName {
			return ns.Backend
		}
	}
	// Default to keychain if namespace not found
	return "keychain"
}

// openBackend resolves and opens the backend for namespaceName using the registry.
// The registry handles the priority: explicit configuration > default (zero-config) factory.
func openBackend(cfg *config.Loaded, namespaceName string) (backend.Backend, error) {
	backendName := getBackendName(cfg, namespaceName)

	// Registry.Resolve handles both explicit and default backends
	// Explicit backends are registered during config.Load()
	// Default backends are registered via init() in each backend package
	return backend.DefaultRegistry.Resolve(backendName)
}

// namespaceOpts returns the Options map for a namespace declared in the dotfile,
// or nil if the namespace is not declared.
func namespaceOpts(cfg *config.Loaded, namespaceName string) map[string]string {
	for _, ns := range cfg.Namespaces {
		if ns.Name == namespaceName {
			return ns.Options
		}
	}
	return nil
}

func mergeOpts(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(override))
	maps.Copy(out, base)
	maps.Copy(out, override)
	return out
}
