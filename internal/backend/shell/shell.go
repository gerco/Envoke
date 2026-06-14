// Package shell implements a backend that runs shell commands to obtain secret values.
// It is always compiled in — no build tag required.
package shell

import (
	"fmt"
	"os/exec"
	"strings"

	"git.dries.info/gerco/envoke/internal/backend"
)

// shellKey is the reserved option key for the shell invocation prefix.
// It is never treated as a variable name.
const shellKey = "shell"

func init() {
	backend.Register("shell", func(opts map[string]string, _ bool) (backend.Backend, error) {
		return newBackend(opts)
	})
}

func newBackend(opts map[string]string) (backend.Backend, error) {
	shellArgs := defaultShellArgs()
	if s, ok := opts[shellKey]; ok && s != "" {
		shellArgs = strings.Fields(s)
	}
	vars := make(map[string]string, len(opts))
	for k, v := range opts {
		if k != shellKey {
			vars[k] = v
		}
	}
	return &shellBackend{shellArgs: shellArgs, vars: vars}, nil
}

type shellBackend struct {
	shellArgs []string // e.g. ["sh", "-c"] or ["/bin/bash", "-c"]
	vars      map[string]string
}

// List returns all variable names declared for this namespace.
func (s *shellBackend) List(_ string) ([]string, error) {
	keys := make([]string, 0, len(s.vars))
	for k := range s.vars {
		keys = append(keys, k)
	}
	return keys, nil
}

// Get runs the command associated with key and returns its trimmed stdout.
func (s *shellBackend) Get(namespace, key string) (string, error) {
	cmd, ok := s.vars[key]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", backend.ErrNotFound, namespace, key)
	}
	args := append(s.shellArgs, cmd) //nolint:gocritic // intentional append to slice literal
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return "", fmt.Errorf("shell var %s/%s: command %q: %w", namespace, key, cmd, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Set is not supported by the shell backend.
func (s *shellBackend) Set(_, _, _ string) error {
	return fmt.Errorf("shell backend is read-only")
}
