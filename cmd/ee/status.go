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

		// Show all backends table
		fmt.Fprintln(os.Stderr, "Backends:")
		w := tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tSTATUS")
		fmt.Fprintln(w, "────\t────\t──────")

		// Collect and sort all backend names
		allBackends := make(map[string]struct {
			typ    string
			status string
		})

		// Add registered backends (from compiled-in backends)
		for _, name := range backend.Names() {
			if backend.DefaultRegistry.IsDisabled(name) {
				allBackends[name] = struct {
					typ    string
					status string
				}{typ: "registered", status: "✓ compiled in"}
				continue
			}
			// Try to resolve to check if it's available
			_, err := backend.DefaultRegistry.Resolve(name)
			status := "✓ available"
			if err != nil {
				status = fmt.Sprintf("✗ %s", err)
			}
			allBackends[name] = struct {
				typ    string
				status string
			}{typ: "registered", status: status}
		}

		// Add explicit backends from config
		for name, bc := range cfg.Global.Backends {
			status := fmt.Sprintf("(%s)", bc.Type)
			// Check if the explicit backend can be resolved
			if _, err := backend.DefaultRegistry.Resolve(name); err != nil {
				status = fmt.Sprintf("✗ %s", err)
			}
			allBackends[name] = struct {
				typ    string
				status string
			}{typ: "explicit", status: status}
		}

		// Sort and print
		names := make([]string, 0, len(allBackends))
		for name := range allBackends {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			b := allBackends[name]
			fmt.Fprintf(w, "%s\t%s\t%s\n", name, b.typ, b.status)
		}
		w.Flush()

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

// checkNamespaceStatus checks if a namespace's backend can be resolved.
func checkNamespaceStatus(ns config.NamespaceEntry) string {
	_, err := backend.DefaultRegistry.Resolve(ns.Backend)
	if err != nil {
		return fmt.Sprintf("✗ %s", err)
	}
	return "✓ ok"
}
