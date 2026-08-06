package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"git.dries.info/gerco/envoke/internal/backend"
	"git.dries.info/gerco/envoke/internal/config"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:   "edit [namespace]",
	Short: "Interactive editor for browsing and editing secrets",
	Long: `Browse, add, update, and delete secrets in a namespace.
Values are masked by default and can be toggled for reveal.
Changes are written back to the backend immediately.

Runs inline in the terminal (no full-screen mode).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(projectDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Non-interactive fallback
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return editFallback(cfg, args)
		}

		var startNamespace string
		if len(args) > 0 {
			startNamespace = args[0]
		}

		return runInlineEditor(cfg, startNamespace)
	},
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	valueStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
)

func runInlineEditor(cfg *config.Loaded, startNS string) error {
	editor := &inlineEditor{
		cfg:      cfg,
		backends: make(map[string]backend.Backend),
		reader:   bufio.NewReader(os.Stdin),
	}

	if startNS != "" {
		editor.namespace = startNS
		return editor.keyListLoop()
	}

	return editor.namespaceLoop()
}

type inlineEditor struct {
	cfg       *config.Loaded
	backends  map[string]backend.Backend
	reader    *bufio.Reader
	namespace string
}

func (e *inlineEditor) readLine() string {
	line, _ := e.reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func (e *inlineEditor) readPassword(prompt string) string {
	fmt.Print(prompt)
	pass, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	return string(pass)
}

func (e *inlineEditor) openBackend(ns string) (backend.Backend, error) {
	if b, ok := e.backends[ns]; ok {
		return b, nil
	}
	b, err := openBackend(e.cfg, ns)
	if err != nil {
		return nil, err
	}
	e.backends[ns] = b
	return b, nil
}

// --- namespace loop ---

func (e *inlineEditor) namespaceLoop() error {
	for {
		e.printNamespaceMenu()
		fmt.Print("> ")
		choice := e.readLine()

		switch choice {
		case "q", "quit":
			return nil
		case "a", "add":
			if err := e.addNamespace(); err != nil {
				fmt.Println(errStyle.Render(fmt.Sprintf("Error: %v", err)))
			}
		case "d", "del":
			if err := e.deleteNamespace(); err != nil {
				fmt.Println(errStyle.Render(fmt.Sprintf("Error: %v", err)))
			}
		default:
			idx, err := strconv.Atoi(choice)
			if err != nil || idx < 1 || idx > len(e.cfg.Namespaces) {
				fmt.Println(errStyle.Render("Invalid choice"))
				continue
			}
			e.namespace = e.cfg.Namespaces[idx-1].Name
			if err := e.keyListLoop(); err != nil {
				return err
			}
		}
	}
}

func (e *inlineEditor) printNamespaceMenu() {
	fmt.Println()
	fmt.Println(titleStyle.Render(" envoke edit "))
	fmt.Println()
	if len(e.cfg.Namespaces) == 0 {
		fmt.Println("(no namespaces)")
	} else {
		for i, ns := range e.cfg.Namespaces {
			fmt.Printf("%d. %s (%s)\n", i+1, ns.Name, ns.Backend)
		}
	}
	fmt.Println()
	fmt.Println(helpStyle.Render("1-N: select namespace  a: add  d: delete  q: quit"))
}

func (e *inlineEditor) addNamespace() error {
	fmt.Print("Namespace name: ")
	name := e.readLine()
	if name == "" {
		return fmt.Errorf("name is required")
	}

	for _, ns := range e.cfg.Namespaces {
		if ns.Name == name {
			return fmt.Errorf("namespace %q already exists", name)
		}
	}

	// Collect backend options
	opts := []string{"keychain"}
	for _, n := range backend.Names() {
		if n != "keychain" && !backend.DefaultRegistry.IsDisabled(n) {
			opts = append(opts, n)
		}
	}
	for _, n := range backend.DefaultRegistry.ExplicitNames() {
		if !slices.Contains(opts, n) {
			opts = append(opts, n)
		}
	}

	fmt.Printf("Backend (%s): ", strings.Join(opts, ", "))
	backendName := e.readLine()
	if backendName == "" {
		backendName = "keychain"
	}

	// Save
	newNS := config.NamespaceEntry{Name: name, Backend: backendName}
	e.cfg.Namespaces = append(e.cfg.Namespaces, newNS)
	df := config.DotFile{Namespaces: e.cfg.Namespaces}
	if err := config.SaveDotfile(projectDir, df); err != nil {
		return err
	}

	fmt.Println(valueStyle.Render("Namespace added."))
	return nil
}

func (e *inlineEditor) deleteNamespace() error {
	if len(e.cfg.Namespaces) == 0 {
		return fmt.Errorf("no namespaces to delete")
	}

	e.printNamespaceMenu()
	fmt.Print("Delete which namespace (1-N, or c to cancel): ")
	choice := e.readLine()
	if choice == "c" {
		return nil
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(e.cfg.Namespaces) {
		return fmt.Errorf("invalid choice")
	}

	ns := e.cfg.Namespaces[idx-1]
	fmt.Printf("Remove namespace %q from .envoke? [y/N]: ", ns.Name)
	confirm := e.readLine()
	if !strings.EqualFold(confirm, "y") {
		return nil
	}

	// Remove
	newNS := make([]config.NamespaceEntry, 0, len(e.cfg.Namespaces))
	for _, n := range e.cfg.Namespaces {
		if n.Name != ns.Name {
			newNS = append(newNS, n)
		}
	}
	e.cfg.Namespaces = newNS
	df := config.DotFile{Namespaces: e.cfg.Namespaces}
	if err := config.SaveDotfile(projectDir, df); err != nil {
		return err
	}

	fmt.Println(valueStyle.Render("Namespace removed."))
	return nil
}

// --- key list loop ---

func (e *inlineEditor) keyListLoop() error {
	b, err := e.openBackend(e.namespace)
	if err != nil {
		return fmt.Errorf("backend %q: %w", e.namespace, err)
	}

	for {
		keys, err := b.List(e.namespace)
		if err != nil {
			return fmt.Errorf("list keys: %w", err)
		}

		e.printKeyMenu(keys)
		fmt.Print("> ")
		choice := e.readLine()

		switch choice {
		case "b", "back":
			return nil
		case "q", "quit":
			return nil
		case "a", "add":
			if err := e.editKey(keys, true); err != nil {
				fmt.Println(errStyle.Render(fmt.Sprintf("Error: %v", err)))
			}
		case "d", "del":
			if err := e.deleteKey(keys); err != nil {
				fmt.Println(errStyle.Render(fmt.Sprintf("Error: %v", err)))
			}
		default:
			idx, err := strconv.Atoi(choice)
			if err != nil || idx < 1 || idx > len(keys) {
				fmt.Println(errStyle.Render("Invalid choice"))
				continue
			}
			if err := e.keyDetailLoop(keys[idx-1]); err != nil {
				return err
			}
		}
	}
}

func (e *inlineEditor) printKeyMenu(keys []string) {
	fmt.Println()
	fmt.Println(titleStyle.Render(fmt.Sprintf(" Namespace: %s ", e.namespace)))
	fmt.Println()
	if len(keys) == 0 {
		fmt.Println("(no keys)")
	} else {
		for i, k := range keys {
			fmt.Printf("%d. %s\n", i+1, k)
		}
	}
	fmt.Println()
	fmt.Println(helpStyle.Render("1-N: view key  a: add  d: delete  b: back  q: quit"))
}

func (e *inlineEditor) editKey(keys []string, isNew bool) error {
	var keyName string
	var currentValue string

	if isNew {
		fmt.Print("Key name: ")
		keyName = e.readLine()
		if keyName == "" {
			return fmt.Errorf("key name is required")
		}
		for _, k := range keys {
			if k == keyName {
				return fmt.Errorf("key %q already exists", keyName)
			}
		}
	} else {
		fmt.Print("Key name: ")
		keyName = e.readLine()
		b, err := e.openBackend(e.namespace)
		if err != nil {
			return err
		}
		val, err := b.Get(e.namespace, keyName)
		if err != nil {
			return err
		}
		currentValue = val
	}

	fmt.Printf("Current value: %s\n", maskValue(currentValue))
	val := e.readPassword("New value (empty to keep): ")
	if val == "" && !isNew {
		val = currentValue
	}

	b, err := e.openBackend(e.namespace)
	if err != nil {
		return err
	}
	if err := b.Set(e.namespace, keyName, val); err != nil {
		return err
	}

	fmt.Println(valueStyle.Render("Saved."))
	return nil
}

func (e *inlineEditor) deleteKey(keys []string) error {
	if len(keys) == 0 {
		return fmt.Errorf("no keys to delete")
	}
	fmt.Print("Delete which key (1-N, or c to cancel): ")
	choice := e.readLine()
	if choice == "c" {
		return nil
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(keys) {
		return fmt.Errorf("invalid choice")
	}

	key := keys[idx-1]
	fmt.Printf("Delete key %q? [y/N]: ", key)
	confirm := e.readLine()
	if !strings.EqualFold(confirm, "y") {
		return nil
	}

	b, err := e.openBackend(e.namespace)
	if err != nil {
		return err
	}
	if err := b.Set(e.namespace, key, ""); err != nil {
		return err
	}

	fmt.Println(valueStyle.Render("Deleted."))
	return nil
}

// --- key detail loop ---

func (e *inlineEditor) keyDetailLoop(key string) error {
	b, err := e.openBackend(e.namespace)
	if err != nil {
		return err
	}

	val, err := b.Get(e.namespace, key)
	if err != nil {
		return err
	}

	revealed := false
	for {
		fmt.Println()
		fmt.Println(titleStyle.Render(fmt.Sprintf(" Key: %s ", key)))
		fmt.Println()

		display := maskValue(val)
		if revealed {
			display = val
		}
		fmt.Println("Value: " + valueStyle.Render(display))
		fmt.Println()

		if revealed {
			fmt.Println(helpStyle.Render("h: hide  e: edit  d: delete  b: back  q: quit"))
		} else {
			fmt.Println(helpStyle.Render("v: reveal  e: edit  d: delete  b: back  q: quit"))
		}

		fmt.Print("> ")
		choice := e.readLine()

		switch choice {
		case "v", "reveal":
			revealed = true
		case "h", "hide":
			revealed = false
		case "e", "edit":
			newVal := e.readPassword("New value (empty to keep): ")
			if newVal != "" {
				val = newVal
				if err := b.Set(e.namespace, key, val); err != nil {
					fmt.Println(errStyle.Render(fmt.Sprintf("Error: %v", err)))
				} else {
					fmt.Println(valueStyle.Render("Saved."))
				}
			}
		case "d", "delete":
			fmt.Printf("Delete key %q? [y/N]: ", key)
			confirm := e.readLine()
			if strings.EqualFold(confirm, "y") {
				if err := b.Set(e.namespace, key, ""); err != nil {
					fmt.Println(errStyle.Render(fmt.Sprintf("Error: %v", err)))
				} else {
					fmt.Println(valueStyle.Render("Deleted."))
					return nil
				}
			}
		case "b", "back":
			return nil
		case "q", "quit":
			return nil
		}
	}
}

func maskValue(val string) string {
	if val == "" {
		return "(empty)"
	}
	return strings.Repeat("*", len(val))
}

// --- fallback for non-interactive use ---

func editFallback(cfg *config.Loaded, args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Namespaces:")
		for _, ns := range cfg.Namespaces {
			fmt.Println(ns.Name)
		}
		return nil
	}
	ns := args[0]
	b, err := openBackend(cfg, ns)
	if err != nil {
		return err
	}
	keys, err := b.List(ns)
	if err != nil {
		return fmt.Errorf("list %q: %w", ns, err)
	}
	for _, k := range keys {
		fmt.Println(k)
	}
	return nil
}
