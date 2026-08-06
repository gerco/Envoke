package main

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
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
	backendCmd.AddCommand(backendAddCmd)
	backendCmd.AddCommand(backendEditCmd)
	backendCmd.AddCommand(backendRemoveCmd)

	backendAddCmd.Flags().StringVarP(&addType, "type", "t", "", "Backend type (required)")
	backendAddCmd.Flags().StringArrayVarP(&addOptions, "set", "s", nil, "Set an option as key=value (repeatable)")
	_ = backendAddCmd.MarkFlagRequired("type")

	backendEditCmd.Flags().StringVarP(&editType, "type", "t", "", "Change the backend type")
	backendEditCmd.Flags().StringArrayVarP(&editSetOpts, "set", "s", nil, "Set or update an option as key=value (repeatable)")
	backendEditCmd.Flags().StringArrayVarP(&editUnsetOpts, "unset", "u", nil, "Remove an option by key (repeatable)")
}

var (
	addType    string
	addOptions []string

	editType      string
	editSetOpts   []string
	editUnsetOpts []string
)

var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "Manage backends",
}

var backendListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backends: implicit (compiled-in) and explicit (from config)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.LoadGlobal()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		backend.DefaultRegistry.SetDisabled(cfg.DisabledImplicitBackends)

		type row struct{ name, kind, detail string }
		var rows []row

		for _, name := range backend.Names() {
			detail := "enabled"
			if backend.DefaultRegistry.IsDisabled(name) {
				detail = "disabled"
			}
			rows = append(rows, row{name, "registered", detail})
		}
		for name, bc := range cfg.Backends {
			rows = append(rows, row{name, "explicit", bc.Type})
		}
		slices.SortFunc(rows, func(a, b row) int { return strings.Compare(a.name, b.name) })

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tKIND\tDETAIL")
		fmt.Fprintln(w, "────\t────\t──────")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.name, r.kind, r.detail)
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

		// Find and remove from disabled list
		newDisabled := cfg.DisabledImplicitBackends[:0]
		found := false
		for _, n := range cfg.DisabledImplicitBackends {
			if n == name {
				found = true
			} else {
				newDisabled = append(newDisabled, n)
			}
		}
		if !found {
			fmt.Printf("Backend %q is not disabled.\n", name)
			return nil
		}

		cfg.DisabledImplicitBackends = newDisabled
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

		// Check if already disabled
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

var backendAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add an explicit backend to the global config",
	Long: `Add a named, explicitly configured backend to ~/.config/envoke/config.yaml.

Examples:
  ee backend add work-vault --type keeper --set vault_id=abc123
  ee backend add prod-aws --type aws --set region=us-east-1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.LoadGlobal()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.BackendByName(name) != nil {
			return fmt.Errorf("backend %q already exists (use 'ee backend edit' to modify it)", name)
		}
		opts, err := parseOptions(addOptions)
		if err != nil {
			return err
		}

		if cfg.Backends == nil {
			cfg.Backends = make(map[string]config.BackendConfig)
		}
		cfg.Backends[name] = config.BackendConfig{
			Type:   addType,
			Config: opts,
		}
		if err := config.SaveGlobal(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Backend %q added.\n", name)
		return nil
	},
}

var backendEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an explicit backend in the global config",
	Long: `Modify an existing explicit backend in ~/.config/envoke/config.yaml.

Examples:
  ee backend edit prod-aws --set region=eu-west-1
  ee backend edit work-vault --type keeper --set vault_id=new123 --unset old_key`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.LoadGlobal()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		bc := cfg.BackendByName(name)
		if bc == nil {
			return fmt.Errorf("backend %q not found in config (use 'ee backend add' to create it)", name)
		}
		if editType != "" {
			bc.Type = editType
		}
		setOpts, err := parseOptions(editSetOpts)
		if err != nil {
			return err
		}
		if bc.Config == nil && len(setOpts) > 0 {
			bc.Config = make(map[string]string)
		}
		maps.Copy(bc.Config, setOpts)
		for _, k := range editUnsetOpts {
			delete(bc.Config, k)
		}

		// Update the backend in the map
		cfg.Backends[name] = *bc

		if err := config.SaveGlobal(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Backend %q updated.\n", name)
		return nil
	},
}

var backendRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove an explicit backend, or disable an implicit one",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.LoadGlobal()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Remove explicit backend if present.
		if _, exists := cfg.Backends[name]; exists {
			delete(cfg.Backends, name)
			if err := config.SaveGlobal(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Backend %q removed.\n", name)
			return nil
		}

		// For registered (compiled-in) backends, disable instead.
		if backend.DefaultRegistry.HasBackend(name) {
			// Check if already disabled
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
		}
		return fmt.Errorf("backend %q not found", name)
	},
}

// parseOptions parses a slice of "key=value" strings into a map.
func parseOptions(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	opts := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid option %q: expected key=value", p)
		}
		opts[k] = v
	}
	return opts, nil
}
