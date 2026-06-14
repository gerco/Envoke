// Package runner fetches secrets from configured backends and executes a
// subprocess with those secrets injected into its environment.
package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

// Verbose enables verbose diagnostic output to stderr when set to true.
// Set by the caller (e.g. from a --verbose flag or config file) before calling Run.
var Verbose bool

// VeryVerbose enables extra diagnostic output (individual key fetches) when set to true.
// Implies Verbose.
var VeryVerbose bool

// Run executes args[0] with args[1:] as its arguments. Secrets from all
// namespaces in cfg are fetched and injected on top of the current
// environment. The subprocess inherits stdin, stdout, and stderr.
//
// Run returns the subprocess's exit code. It only returns a non-nil error for
// failures that prevent the subprocess from starting all.
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

	if Verbose {
		fmt.Fprintf(os.Stderr, "running: %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command(path, args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 1, fmt.Errorf("exec: %w", err)
	}
	return 0, nil
}

// buildEnv constructs the full environment slice for the subprocess.
// Namespaces are processed sequentially - each namespace's secrets are added
// to the process environment before the next backend is resolved, enabling
// credential chaining (e.g., fetch AWS creds from keychain, then use them
// to access AWS Secrets Manager).
func buildEnv(cfg *config.Loaded) ([]string, error) {
	// Start with the current environment as a base.
	envMap := environToMap(os.Environ())

	// Process namespaces sequentially, updating environment before each backend resolution
	for _, ns := range cfg.Namespaces {
		// Update process environment so backends can see accumulated secrets
		// This enables credential chaining between namespaces
		for k, v := range envMap {
			os.Setenv(k, v)
		}

		// Resolve backend with current env state (always fresh instance)
		b, err := backend.DefaultRegistry.Resolve(ns.Backend)
		if err != nil {
			return nil, fmt.Errorf("namespace %q references backend %q: %w", ns.Name, ns.Backend, err)
		}

		keys, err := b.List(ns.Name)
		if err != nil {
			return nil, fmt.Errorf("list namespace %q (backend %q): %w", ns.Name, ns.Backend, err)
		}

		if Verbose {
			fmt.Fprintf(os.Stderr, "namespace %q: backend %q, fetched %d key(s)\n", ns.Name, ns.Backend, len(keys))
		}

		for _, key := range keys {
			if VeryVerbose {
				fmt.Fprintf(os.Stderr, "  fetching %s/%s\n", ns.Name, key)
			}
			value, err := b.Get(ns.Name, key)
			if err != nil {
				if errors.Is(err, backend.ErrNotFound) {
					fmt.Fprintf(os.Stderr, "warning: key %s/%s not found in backend %q, skipping\n", ns.Name, key, ns.Backend)
					continue
				}
				return nil, fmt.Errorf("get %s/%s (backend %q): %w", ns.Name, key, ns.Backend, err)
			}
			envMap[key] = value
		}
	}

	return envMapToSlice(envMap), nil
}

// environToMap converts an environment slice (KEY=VALUE) to a map.
func environToMap(environ []string) map[string]string {
	result := make(map[string]string, len(environ))
	for _, e := range environ {
		if i := strings.Index(e, "="); i > 0 {
			result[e[:i]] = e[i+1:]
		}
	}
	return result
}

// envMapToSlice converts an environment map to a slice (KEY=VALUE).
func envMapToSlice(envMap map[string]string) []string {
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, k+"="+v)
	}
	return result
}
