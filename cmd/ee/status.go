package main

import (
	"fmt"
	"os"
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
	Short: "Show backend health for the current project",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load(projectDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		globalPath, _ := config.GlobalConfigPath()
		if globalPath == "" {
			globalPath = "(unknown)"
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAMESPACE\tBACKEND\tSTATUS\tCONFIG")
		fmt.Fprintln(w, "─────────\t───────\t──────\t──────")

		for _, ns := range cfg.Namespaces {
			status, configSource := checkNamespace(cfg, ns, globalPath)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ns.Name, ns.Backend, status, configSource)
		}
		return w.Flush()
	},
}

func checkNamespace(cfg *config.Loaded, ns config.Namespace, globalPath string) (string, string) {
	bc := cfg.Global.BackendByName(ns.Backend)
	if bc == nil {
		return "✗ backend not configured", globalPath
	}

	b, err := backend.New(bc.Type, mergeOpts(bc.Options, ns.Options))
	if err != nil {
		return fmt.Sprintf("✗ %s", err), globalPath
	}

	_, err = b.List(ns.Name)
	if err != nil {
		return fmt.Sprintf("✗ %s", err), globalPath
	}
	return "✓ ok", globalPath
}
