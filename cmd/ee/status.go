package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show backend health and availability for the current project",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load(projectDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		globalPath, _ := config.GlobalConfigPath()
		if globalPath == "" {
			globalPath = "(unknown)"
		}

		// Show available (implicit/default) backends
		fmt.Fprintln(os.Stderr, "Available backends (zero-config, can reference directly in namespace config):")
		w := tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)

		defaultBackends := backend.DefaultRegistry.DefaultNames()
		sort.Strings(defaultBackends)

		for _, name := range defaultBackends {
			available, reason := checkBackendAvailability(name)
			if available {
				fmt.Fprintf(w, "  [implicit] %s\t✓ available\n", name)
			} else {
				fmt.Fprintf(w, "  [implicit] %s\t✗ not available (%s)\n", name, reason)
			}
		}
		w.Flush()

		// Show configured (explicit) backends
		if len(cfg.Global.Backends) > 0 {
			fmt.Fprintf(os.Stderr, "\nConfigured backends (from %s):\n", globalPath)
			w = tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)
			for _, bc := range cfg.Global.Backends {
				fmt.Fprintf(w, "  [explicit] %s\t(%s)\n", bc.Name, bc.Type)
			}
			w.Flush()
		} else {
			fmt.Fprintf(os.Stderr, "\nNo explicit backend configurations found in %s\n", globalPath)
		}

		// Show namespace status
		fmt.Fprintln(os.Stderr, "\nProject namespaces:")
		w = tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAMESPACE\tBACKEND\tSTATUS")
		fmt.Fprintln(w, "─────────\t───────\t──────")

		for _, ns := range cfg.Namespaces {
			status := checkNamespaceStatus(ns)
			fmt.Fprintf(w, "%s\t%s\t%s\n", ns.Name, ns.Backend, status)
		}

		if len(cfg.Namespaces) == 0 {
			fmt.Fprintln(w, "(none)\t\t")
		}

		return w.Flush()
	},
}

// checkBackendAvailability checks if a backend is available based on env vars.
// Returns (true, "") if available, (false, reason) if not.
func checkBackendAvailability(name string) (bool, string) {
	switch name {
	case "keychain":
		// Keychain is always available on macOS/Windows
		// On Linux it depends on Secret Service, but we can't quickly check that
		return true, ""
	case "aws":
		// Check for AWS credentials via env vars or config file indicator
		if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
			return true, ""
		}
		if os.Getenv("AWS_PROFILE") != "" {
			return true, ""
		}
		if os.Getenv("AWS_SDK_LOAD_CONFIG") != "" {
			return true, ""
		}
		// Check for config file existence would require file system access
		// For now, assume credentials might be in ~/.aws/credentials
		return false, "needs AWS_ACCESS_KEY_ID or AWS_PROFILE or ~/.aws/credentials"
	case "1password":
		if os.Getenv("OP_SERVICE_ACCOUNT_TOKEN") != "" {
			return true, ""
		}
		return false, "needs OP_SERVICE_ACCOUNT_TOKEN"
	case "keeper":
		if os.Getenv("KSM_CONFIG") != "" {
			return true, ""
		}
		return false, "needs KSM_CONFIG"
	case "jumpcloud":
		if os.Getenv("JUMPCLOUD_API_KEY") != "" {
			return true, ""
		}
		return false, "needs JUMPCLOUD_API_KEY"
	default:
		return false, "unknown backend"
	}
}

// checkNamespaceStatus checks if a namespace's backend can be resolved.
func checkNamespaceStatus(ns config.Namespace) string {
	_, err := backend.DefaultRegistry.Resolve(ns.Backend)
	if err != nil {
		return fmt.Sprintf("✗ %s", err)
	}
	return "✓ ok"
}
