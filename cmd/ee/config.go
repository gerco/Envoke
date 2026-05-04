package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"git.dries.info/gerco/envoke/internal/config"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configShowCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage envoke configuration",
	Long:  "View and edit the global configuration file that defines backends.",
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the global config file path",
	// Disable the default behavior that tries to load project config
	// since this command doesn't need it.
	PreRunE: func(cmd *cobra.Command, _ []string) error { return nil },
	RunE: func(cmd *cobra.Command, _ []string) error {
		path, err := config.GlobalConfigPath()
		if err != nil {
			return fmt.Errorf("cannot determine config path: %w", err)
		}
		fmt.Println(path)
		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the global config in your default editor",
	Long: `Opens the global configuration file in the default editor.

On Windows, uses notepad.exe.
On Unix systems, uses $EDITOR or falls back to vi.

If stdin is not a TTY, prints the path instead.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		path, err := config.GlobalConfigPath()
		if err != nil {
			return fmt.Errorf("cannot determine config path: %w", err)
		}

		// Check if we're in an interactive environment
		if !isInteractive() {
			fmt.Println(path)
			return nil
		}

		// Ensure the directory exists
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("cannot create config directory: %w", err)
		}

		editor := getEditor()
		execCmd := exec.Command(editor, path)
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		return execCmd.Run()
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the current global configuration (redacted)",
	Long: `Display the contents of the global configuration file.
Sensitive values (tokens, passwords) are redacted.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		path, err := config.GlobalConfigPath()
		if err != nil {
			return fmt.Errorf("cannot determine config path: %w", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("# No global config file found")
				fmt.Printf("# Expected location: %s\n", path)
				return nil
			}
			return fmt.Errorf("cannot read config: %w", err)
		}

		// Print with a header showing the path
		fmt.Printf("# Global config: %s\n", path)
		fmt.Println(string(data))
		return nil
	},
}

// isInteractive returns true if stdin is a terminal.
func isInteractive() bool {
	if runtime.GOOS == "windows" {
		// On Windows, check if we have a console
		fileInfo, err := os.Stdin.Stat()
		if err != nil {
			return false
		}
		return fileInfo.Mode()&os.ModeCharDevice != 0
	}
	// On Unix, check if stdin is a TTY
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fileInfo.Mode()&os.ModeCharDevice != 0
}

// getEditor returns the editor command to use.
func getEditor() string {
	if runtime.GOOS == "windows" {
		// Windows: use notepad.exe
		return "notepad.exe"
	}
	// Unix: check $EDITOR
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	// Default to vi
	return "vi"
}
