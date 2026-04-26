package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

func init() {
	rootCmd.AddCommand(setCmd)
	setCmd.Flags().String("backend", "", "Backend to use (overrides namespace lookup)")
}

var setCmd = &cobra.Command{
	Use:   "set <namespace> <key> [value]",
	Short: "Store a secret in a backend namespace",
	Long: `Store a secret in the backend configured for namespace.
If value is omitted, it is read from stdin (hidden if the terminal supports it).
This avoids the value appearing in shell history.

Examples:
  ee set db-local DB_PASSWORD
  echo "s3cr3t" | ee set db-local DB_PASSWORD
  ee set --backend aws production-db DB_PASSWORD`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, key := args[0], args[1]
		var value string
		if len(args) == 3 {
			value = args[2]
		} else {
			v, err := readSecret(fmt.Sprintf("Value for %s/%s: ", namespace, key))
			if err != nil {
				return fmt.Errorf("read value: %w", err)
			}
			value = v
		}

		cfg, err := config.Load(projectDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Check for --backend flag
		backendName, _ := cmd.Flags().GetString("backend")

		var bc *config.BackendConfig
		if backendName != "" {
			// Use specified backend
			bc = cfg.Global.BackendByName(backendName)
			if bc == nil {
				return fmt.Errorf("backend %q not configured", backendName)
			}
		} else {
			// Use namespace lookup
			bc = resolveBackend(cfg, namespace)
			if bc == nil {
				return fmt.Errorf("no backend available for namespace %q, use --backend to specify", namespace)
			}
			backendName = bc.Name
		}

		// Open the backend with options
		ns := namespaceOpts(cfg, namespace)
	opts := mergeOpts(bc.Options, ns)
		b, err := backend.New(bc.Type, opts)
		if err != nil {
			return fmt.Errorf("open backend %q: %w", bc.Name, err)
		}

		if err := b.Set(namespace, key, value); err != nil {
			return fmt.Errorf("set %s/%s: %w", namespace, key, err)
		}

		fmt.Fprintf(os.Stderr, "Stored %s/%s\n", namespace, key)

		// Only update .envoke if --backend was used (creating new namespace)
		// or if namespace not already declared
		added, err := config.EnsureNamespace(projectDir, namespace, backendName)
		if err != nil {
			return fmt.Errorf("update .envoke: %w", err)
		}
		if added {
			fmt.Fprintf(os.Stderr, "Added namespace %q (backend: %s) to .envoke\n", namespace, backendName)
		}

		return nil
	},
}

// readSecret reads a value from stdin, suppressing echo when on a real terminal.
func readSecret(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprint(os.Stderr, prompt)
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	// Non-interactive: read a line from stdin.
	var line string
	_, err := fmt.Scanln(&line)
	if err != nil {
		return "", err
	}
	return line, nil
}
