package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"git.dries.info/gerco/envoke/internal/config"

	_ "git.dries.info/gerco/envoke/internal/backend/jumpcloud"
	_ "git.dries.info/gerco/envoke/internal/backend/keeper"
	_ "git.dries.info/gerco/envoke/internal/backend/keychain"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list [namespace]",
	Short: "List secret keys in a namespace or all project namespaces",
	Long: `List the names of secrets stored in a namespace.

With no argument, lists keys across all namespaces declared in .envoke.
With a namespace argument, lists keys in that namespace (defaulting to the
local backend if the namespace is not declared in .envoke).

Only key names are printed — never values.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(projectDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if len(args) == 1 {
			return listOne(cfg, args[0])
		}
		return listAll(cfg)
	},
}

// listOne prints all keys in a single namespace, one per line.
func listOne(cfg *config.Loaded, namespaceName string) error {
	b, err := openBackend(cfg, namespaceName)
	if err != nil {
		return fmt.Errorf("open backend for %q: %w", namespaceName, err)
	}

	keys, err := b.List(namespaceName)
	if err != nil {
		return fmt.Errorf("list %q: %w", namespaceName, err)
	}

	sort.Strings(keys)
	for _, k := range keys {
		fmt.Println(k)
	}
	return nil
}

// listAll prints keys across every namespace declared in the project dotfile,
// grouped by namespace in a two-column table.
func listAll(cfg *config.Loaded) error {
	if len(cfg.Namespaces) == 0 {
		fmt.Fprintln(os.Stderr, "no namespaces declared in .envoke")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, ns := range cfg.Namespaces {
		b, err := openBackend(cfg, ns.Name)
		if err != nil {
			fmt.Fprintf(w, "%s\t(error: %s)\n", ns.Name, err)
			continue
		}

		keys, err := b.List(ns.Name)
		if err != nil {
			fmt.Fprintf(w, "%s\t(error: %s)\n", ns.Name, err)
			continue
		}

		sort.Strings(keys)
		if len(keys) == 0 {
			fmt.Fprintf(w, "%s\t(empty)\n", ns.Name)
			continue
		}
		for _, k := range keys {
			fmt.Fprintf(w, "%s\t%s\n", ns.Name, k)
		}
	}
	return w.Flush()
}
