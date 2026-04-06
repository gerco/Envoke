package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard (coming soon)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return fmt.Errorf("ee init is not yet implemented (tracked in issue #10)")
	},
}
