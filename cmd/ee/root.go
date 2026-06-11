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
		cfg, err := config.Load(projectDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
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
