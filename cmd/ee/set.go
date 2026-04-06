package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"git.dries.info/gerco/envoke/internal/config"

	_ "git.dries.info/gerco/envoke/internal/backend/jumpcloud"
	_ "git.dries.info/gerco/envoke/internal/backend/keeper"
	_ "git.dries.info/gerco/envoke/internal/backend/keychain"
)

func init() {
	rootCmd.AddCommand(setCmd)
}

var setCmd = &cobra.Command{
	Use:   "set <namespace> <key> [value]",
	Short: "Store a secret in a backend namespace",
	Long: `Store a secret in the backend configured for namespace.

If value is omitted, it is read from stdin (hidden if the terminal supports it).
This avoids the value appearing in shell history.

Examples:
  ee set db-local DB_PASSWORD
  echo "s3cr3t" | ee set db-local DB_PASSWORD`,
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

		b, err := openBackend(cfg, namespace)
		if err != nil {
			return fmt.Errorf("open backend for %q: %w", namespace, err)
		}

		if err := b.Set(namespace, key, value); err != nil {
			return fmt.Errorf("set %s/%s: %w", namespace, key, err)
		}

		fmt.Fprintf(os.Stderr, "Stored %s/%s\n", namespace, key)
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
