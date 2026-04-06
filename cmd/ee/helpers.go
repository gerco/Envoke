package main

import (
	"maps"

	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

// resolveBackend returns the BackendConfig for namespaceName by consulting the
// dotfile mapping first, then falling back to the always-present "local" backend.
func resolveBackend(cfg *config.Loaded, namespaceName string) *config.BackendConfig {
	for _, ns := range cfg.Namespaces {
		if ns.Name == namespaceName {
			return cfg.Global.BackendByName(ns.Backend)
		}
	}
	return cfg.Global.BackendByName("local")
}

// openBackend resolves and opens the backend for namespaceName.
func openBackend(cfg *config.Loaded, namespaceName string) (backend.Backend, error) {
	bc := resolveBackend(cfg, namespaceName)
	if bc == nil {
		// resolveBackend only returns nil if "local" is somehow absent, which
		// should never happen after applyDefaults runs.
		return nil, backend.ErrNotFound
	}
	ns := namespaceOpts(cfg, namespaceName)
	return backend.New(bc.Type, mergeOpts(bc.Options, ns))
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
