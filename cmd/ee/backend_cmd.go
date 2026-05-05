package main

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

func init() {
	rootCmd.AddCommand(backendCmd)
	backendCmd.AddCommand(backendListCmd)
	backendCmd.AddCommand(backendEnableCmd)
	backendCmd.AddCommand(backendDisableCmd)
}

var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "Manage compiled-in implicit backends",
}

var backendListCmd = &cobra.Command{
	Use:   "list",
	Short: "List compiled-in implicit backends and their enabled/disabled state",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.LoadGlobal()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		backend.DefaultRegistry.SetDisabled(cfg.DisabledImplicitBackends)

		names := backend.DefaultRegistry.DefaultNames()
		sort.Strings(names)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATE")
		fmt.Fprintln(w, "────\t─────")
		for _, name := range names {
			state := "enabled"
			if backend.DefaultRegistry.IsDisabled(name) {
				state = "disabled"
			}
			fmt.Fprintf(w, "%s\t%s\n", name, state)
		}
		return w.Flush()
	},
}

var backendEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a previously disabled implicit backend",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.LoadGlobal()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		idx := slices.Index(cfg.DisabledImplicitBackends, name)
		if idx == -1 {
			fmt.Printf("Backend %q is not disabled.\n", name)
			return nil
		}
		cfg.DisabledImplicitBackends = slices.Delete(cfg.DisabledImplicitBackends, idx, idx+1)
		if err := config.SaveGlobal(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Backend %q enabled.\n", name)
		return nil
	},
}

var backendDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable an implicit backend so it is hidden and not auto-resolved",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.LoadGlobal()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if slices.Contains(cfg.DisabledImplicitBackends, name) {
			fmt.Printf("Backend %q is already disabled.\n", name)
			return nil
		}
		cfg.DisabledImplicitBackends = append(cfg.DisabledImplicitBackends, name)
		if err := config.SaveGlobal(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Backend %q disabled.\n", name)
		return nil
	},
}
