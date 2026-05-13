package config

import (
	"os"
	"regexp"
)

// envVarRegex matches ${VAR} patterns.
var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandEnv substitutes ${VAR} patterns in s with environment variable values.
func expandEnv(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the variable name from ${VAR}
		varName := match[2 : len(match)-1] // Remove ${ and }
		return os.Getenv(varName)
	})
}

// expandEnvInMap recursively expands environment variables in all string
// values within the map.
func expandEnvInMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = expandEnv(v)
	}
	return result
}

// expandEnvInBackendConfig expands environment variables in a BackendConfig.
func expandEnvInBackendConfig(bc *BackendConfig) {
	if bc == nil {
		return
	}
	bc.Type = expandEnv(bc.Type)
	bc.Config = expandEnvInMap(bc.Config)
}

// expandEnvInNamespace expands environment variables in a Namespace.
func expandEnvInNamespace(ns *Namespace) {
	if ns == nil {
		return
	}
	ns.Backend = expandEnv(ns.Backend)
	ns.Options = expandEnvInMap(ns.Options)
}

// expandEnvInGlobalConfig expands all environment variables in a GlobalConfig.
// It modifies the config in place.
func expandEnvInGlobalConfig(cfg *GlobalConfig) {
	for name, bc := range cfg.Backends {
		expandEnvInBackendConfig(&bc)
		cfg.Backends[name] = bc
	}
}

// expandEnvInDotFile expands all environment variables in a DotFile.
// It modifies the file in place.
func expandEnvInDotFile(df *DotFile) {
	for name, ns := range df.Namespaces {
		expandEnvInNamespace(&ns)
		df.Namespaces[name] = ns
	}
}
