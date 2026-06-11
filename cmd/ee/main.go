package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	bin := filepath.Base(os.Args[0])
	// Strip platform extension (e.g. ".exe" on Windows).
	bin = strings.TrimSuffix(bin, filepath.Ext(bin))

	rootCmd.Use = bin + " [-- <command> [args...]]"
	rootCmd.Long = `Envoke (` + bin + `) reads a project .envoke file, fetches the required secrets
from one or more configured backends, and spawns your command as a subprocess
with those secrets in its environment. Nothing persists. Nothing leaks.

Examples:
  ` + bin + ` -- make dev
  ` + bin + ` -- psql -h $DB_HOST -U $DB_USER
  ` + bin + ` -- aider --model claude-sonnet-4-6`

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, bin+":", err)
		os.Exit(1)
	}
}
