package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"git.dries.info/gerco/envoke/internal/config"
	"git.dries.info/gerco/envoke/internal/runner"
)

var rootCmd = &cobra.Command{
	// Use and Long are set in main() using the actual binary name from os.Args[0].
	Short: "Inject secrets per-command from pluggable backends",
	// Allow arbitrary args so that "ee -- <command>" reaches RunE.
	Args:          cobra.ArbitraryArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		ns, cmdArgs, err := keychainNamespace(os.Args[1:], args)
		if err != nil {
			return err
		}

		var cfg *config.Loaded
		if ns != "" {
			cfg = &config.Loaded{
				Namespaces: []config.NamespaceEntry{
					{Name: ns, Backend: "keychain"},
				},
			}
			args = cmdArgs
		} else {
			cfg, err = config.Load(projectDir)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
		}

		code, err := runner.Run(cfg, args)
		if err != nil {
			return err
		}
		if code != 0 {
			os.Exit(code)
		}
		return nil
	},
}

// projectDir is resolved at command startup and used by all subcommands.
var projectDir string

func init() {
	cobra.OnInitialize(resolveProjectDir)
}

// resolveProjectDir walks up from the current working directory looking for a
// .envoke file. Falls back to the current directory if none is found.
func resolveProjectDir() {
	dir, err := findProjectRoot()
	if err != nil {
		// Non-fatal: commands that need a project root will fail later.
		dir, _ = os.Getwd()
	}
	projectDir = dir
}

// findProjectRoot walks up the directory tree looking for a .envoke file.
func findProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	dir := cwd
	for {
		if _, err := os.Stat(dir + "/.envoke"); err == nil {
			return dir, nil
		}
		parent := parentDir(dir)
		if parent == dir {
			// Reached the filesystem root.
			return cwd, nil
		}
		dir = parent
	}
}

// keychainNamespace inspects the raw os args (everything after the binary name)
// to detect the pattern "ee <namespace> -- <command>". It returns the namespace
// and the cobra args shifted past it. If no -- separator is present, or nothing
// appears before it, it returns ("", cobraArgs, nil) so the caller falls through
// to dotfile-based config. Multiple positional args before -- are an error.
func keychainNamespace(rawArgs []string, cobraArgs []string) (string, []string, error) {
	dashIdx := -1
	for i, a := range rawArgs {
		if a == "--" {
			dashIdx = i
			break
		}
	}
	if dashIdx < 0 {
		return "", cobraArgs, nil
	}

	var positional []string
	for _, a := range rawArgs[:dashIdx] {
		if a != "" && a[0] != '-' {
			positional = append(positional, a)
		}
	}

	switch len(positional) {
	case 0:
		return "", cobraArgs, nil
	case 1:
		// cobra strips --, so cobraArgs starts with the namespace followed by the command.
		return positional[0], cobraArgs[1:], nil
	default:
		return "", nil, fmt.Errorf("expected exactly one namespace before --, got: %v", positional)
	}
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			if i == 0 {
				return string(path[0])
			}
			return path[:i]
		}
	}
	return path
}
