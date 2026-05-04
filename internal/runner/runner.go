// Package runner fetches secrets from configured backends and executes a
// subprocess with those secrets injected into its environment.
package runner

import (
	"fmt"
	"maps"
	"os"
	"os/exec"

	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

// Run executes args[0] with args[1:] as its arguments. Secrets from all
// namespaces in cfg are fetched and injected on top of the current
// environment. The subprocess inherits stdin, stdout, and stderr.
//
// Run returns the subprocess's exit code. It only returns a non-nil error for
// failures that prevent the subprocess from starting at all.
func Run(cfg *config.Loaded, args []string) (int, error) {
	if len(args) == 0 {
		return 1, fmt.Errorf("no command specified")
	}

	env, err := buildEnv(cfg)
	if err != nil {
		return 1, err
	}

	path, err := exec.LookPath(args[0])
	if err != nil {
		return 1, fmt.Errorf("command not found: %s", args[0])
	}

	cmd := exec.Command(path, args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the subprocess in a new process group (Unix) or create new console (Windows)
	// so signals go to the child, not envoke. This is standard behavior for wrapper tools.
	setSysProcAttr(cmd)

	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 1, fmt.Errorf("exec: %w", err)
	}
	return 0, nil
}

// buildEnv constructs the full environment slice for the subprocess:
// the current process's environment with secrets from all namespaces
// overlaid on top.
func buildEnv(cfg *config.Loaded) ([]string, error) {
	// Start with the current environment as a base.
	base := os.Environ()

	secrets, err := fetchSecrets(cfg)
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		return base, nil
	}

	// Build a map of existing env vars so injected secrets take precedence.
	envMap := make(map[string]string, len(base)+len(secrets))
	for _, e := range base {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				envMap[e[:i]] = e[i+1:]
				break
			}
		}
	}
	maps.Copy(envMap, secrets)

	out := make([]string, 0, len(envMap))
	for k, v := range envMap {
		out = append(out, k+"="+v)
	}
	return out, nil
}

// fetchSecrets opens each configured backend and lists + fetches all keys in
// every namespace, returning them as a flat KEY=VALUE map.
func fetchSecrets(cfg *config.Loaded) (map[string]string, error) {
	result := make(map[string]string)

	for _, ns := range cfg.Namespaces {
		// Use registry to resolve backend - supports both implicit (zero-config)
		// and explicit (configured) backends
		b, err := backend.DefaultRegistry.Resolve(ns.Backend)
		if err != nil {
			return nil, fmt.Errorf("namespace %q references backend %q: %w", ns.Name, ns.Backend, err)
		}

		keys, err := b.List(ns.Name)
		if err != nil {
			return nil, fmt.Errorf("list namespace %q (backend %q): %w", ns.Name, ns.Backend, err)
		}

		for _, key := range keys {
			value, err := b.Get(ns.Name, key)
			if err != nil {
				return nil, fmt.Errorf("get %s/%s (backend %q): %w", ns.Name, key, ns.Backend, err)
			}
			result[key] = value
		}
	}
	return result, nil
}
