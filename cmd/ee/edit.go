package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:   "edit [namespace]",
	Short: "Browse and edit secrets in a namespace (coming soon)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, _ []string) error {
		return fmt.Errorf("ee edit is not yet implemented (tracked in issue #13)")
	},
}
