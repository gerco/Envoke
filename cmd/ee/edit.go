package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	Short: "Interactive TUI for browsing and editing secrets",
	Long: `Browse, add, update, and delete secrets in a namespace.
Values are masked by default and can be toggled for reveal.
Changes are written back to the backend immediately.

In non-interactive environments falls back to plain text output.`,
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

		// If namespace provided, start directly in key list
		var startNamespace string
		if len(args) > 0 {
			startNamespace = args[0]
		}

		m := newEditModel(cfg, startNamespace)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("tui: %w", err)
		}
		return nil
	},
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

// --- TUI model ---

type screen int

const (
	screenNamespace screen = iota
	screenKeyList
	screenKeyDetail
	screenEdit
	screenConfirmDelete
)

type editModel struct {
	cfg         *config.Loaded
	backends    map[string]backend.Backend // namespace -> opened backend
	screen      screen
	width       int
	height      int
	namespace   string // currently selected namespace
	keys        []string
	selectedKey int
	selectedNS  int

	// key detail
	revealed bool
	keyValue string
	keyError error

	// edit form
	editKeyInput textinput.Model
	editValInput textinput.Model
	editIsNew    bool // true = add, false = update
	editFocus    int  // 0 = key, 1 = value
	editError    error

	// confirm delete
	confirmDeleteKey string

	// namespace list
	nsList list.Model
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
)

func newEditModel(cfg *config.Loaded, startNS string) *editModel {
	// Build namespace list items
	items := make([]list.Item, len(cfg.Namespaces))
	for i, ns := range cfg.Namespaces {
		items[i] = namespaceItem{name: ns.Name, backend: ns.Backend}
	}

	l := list.New(items, namespaceDelegate{}, 0, 0)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Title = "Select namespace"
	l.Styles.Title = titleStyle

	m := &editModel{
		cfg:      cfg,
		backends: make(map[string]backend.Backend),
		screen:   screenNamespace,
		nsList:   l,
	}

	// Pre-select namespace if provided
	if startNS != "" {
		m.namespace = startNS
		m.screen = screenKeyList
		m.loadKeys()
	}

	return m
}

type namespaceItem struct {
	name    string
	backend string
}

func (i namespaceItem) FilterValue() string { return i.name }

type namespaceDelegate struct{}

func (d namespaceDelegate) Height() int                             { return 1 }
func (d namespaceDelegate) Spacing() int                            { return 0 }
func (d namespaceDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d namespaceDelegate) Render(w io.Writer, m list.Model, idx int, item list.Item) {
	i := item.(namespaceItem)
	selected := idx == m.Index()
	var s string
	if selected {
		s = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("> " + i.name)
	} else {
		s = "  " + i.name
	}
	fmt.Fprint(w, s)
}

// --- model methods ---

func (m *editModel) Init() tea.Cmd {
	return nil
}

func (m *editModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.nsList.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.screen == screenNamespace {
				return m, tea.Quit
			}
		}
	}

	switch m.screen {
	case screenNamespace:
		return m.updateNamespace(msg)
	case screenKeyList:
		return m.updateKeyList(msg)
	case screenKeyDetail:
		return m.updateKeyDetail(msg)
	case screenEdit:
		return m.updateEdit(msg)
	case screenConfirmDelete:
		return m.updateConfirmDelete(msg)
	}

	return m, nil
}

func (m *editModel) updateNamespace(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if item, ok := m.nsList.SelectedItem().(namespaceItem); ok {
				m.namespace = item.name
				m.screen = screenKeyList
				m.loadKeys()
				return m, nil
			}
		}
		if msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.nsList, cmd = m.nsList.Update(msg)
	return m, cmd
}

func (m *editModel) loadKeys() {
	b, err := m.openBackend(m.namespace)
	if err != nil {
		m.keyError = err
		m.keys = nil
		return
	}
	keys, err := b.List(m.namespace)
	if err != nil {
		m.keyError = err
		m.keys = nil
		return
	}
	m.keys = keys
	m.selectedKey = 0
	m.keyError = nil
}

func (m *editModel) openBackend(ns string) (backend.Backend, error) {
	if b, ok := m.backends[ns]; ok {
		return b, nil
	}
	b, err := openBackend(m.cfg, ns)
	if err != nil {
		return nil, err
	}
	m.backends[ns] = b
	return b, nil
}

func (m *editModel) updateKeyList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "backspace":
			m.screen = screenNamespace
			return m, nil
		case "up", "k":
			if m.selectedKey > 0 {
				m.selectedKey--
			}
			return m, nil
		case "down", "j":
			if m.selectedKey < len(m.keys)-1 {
				m.selectedKey++
			}
			return m, nil
		case "enter":
			if len(m.keys) > 0 && m.selectedKey < len(m.keys) {
				m.revealed = false
				m.keyError = nil
				m.loadKeyValue(m.keys[m.selectedKey])
				m.screen = screenKeyDetail
			}
			return m, nil
		case "a":
			m.startEdit(true)
			return m, nil
		case "d":
			if len(m.keys) > 0 && m.selectedKey < len(m.keys) {
				m.confirmDeleteKey = m.keys[m.selectedKey]
				m.screen = screenConfirmDelete
			}
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *editModel) loadKeyValue(key string) {
	b, err := m.openBackend(m.namespace)
	if err != nil {
		m.keyError = err
		m.keyValue = ""
		return
	}
	val, err := b.Get(m.namespace, key)
	if err != nil {
		m.keyError = err
		m.keyValue = ""
		return
	}
	m.keyValue = val
	m.keyError = nil
}

func (m *editModel) updateKeyDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "backspace":
			m.screen = screenKeyList
			return m, nil
		case "v":
			m.revealed = !m.revealed
			return m, nil
		case "e":
			m.startEdit(false)
			return m, nil
		case "d":
			if len(m.keys) > 0 && m.selectedKey < len(m.keys) {
				m.confirmDeleteKey = m.keys[m.selectedKey]
				m.screen = screenConfirmDelete
			}
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *editModel) startEdit(isNew bool) {
	m.editIsNew = isNew
	m.editError = nil
	m.editFocus = 0

	keyInp := textinput.New()
	keyInp.Placeholder = "Key name"
	keyInp.Focus()
	keyInp.CharLimit = 256
	keyInp.Width = 40

	valInp := textinput.New()
	valInp.Placeholder = "Secret value"
	valInp.EchoMode = textinput.EchoPassword
	valInp.CharLimit = 4096
	valInp.Width = 40

	if !isNew && len(m.keys) > 0 && m.selectedKey < len(m.keys) {
		keyInp.SetValue(m.keys[m.selectedKey])
		keyInp.SetCursor(len(m.keys[m.selectedKey]))
		valInp.SetValue(m.keyValue)
		valInp.SetCursor(len(m.keyValue))
	}

	m.editKeyInput = keyInp
	m.editValInput = valInp
	m.screen = screenEdit
}

func (m *editModel) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.editIsNew {
				m.screen = screenKeyList
			} else {
				m.screen = screenKeyDetail
			}
			return m, nil
		case "tab":
			if m.editFocus == 0 {
				m.editFocus = 1
				m.editKeyInput.Blur()
				m.editValInput.Focus()
			} else {
				m.editFocus = 0
				m.editValInput.Blur()
				m.editKeyInput.Focus()
			}
			return m, nil
		case "enter":
			if m.editFocus == 0 && m.editIsNew {
				m.editFocus = 1
				m.editKeyInput.Blur()
				m.editValInput.Focus()
				return m, nil
			}
			return m.saveEdit()
		case "q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	if m.editFocus == 0 {
		m.editKeyInput, cmd = m.editKeyInput.Update(msg)
	} else {
		m.editValInput, cmd = m.editValInput.Update(msg)
	}
	return m, cmd
}

func (m *editModel) saveEdit() (tea.Model, tea.Cmd) {
	key := strings.TrimSpace(m.editKeyInput.Value())
	val := m.editValInput.Value()

	if key == "" {
		m.editError = fmt.Errorf("key name is required")
		return m, nil
	}

	b, err := m.openBackend(m.namespace)
	if err != nil {
		m.editError = err
		return m, nil
	}

	if err := b.Set(m.namespace, key, val); err != nil {
		m.editError = err
		return m, nil
	}

	// Refresh key list
	m.loadKeys()

	// Select the edited key
	for i, k := range m.keys {
		if k == key {
			m.selectedKey = i
			break
		}
	}

	if m.editIsNew {
		m.screen = screenKeyList
	} else {
		m.loadKeyValue(key)
		m.screen = screenKeyDetail
	}
	return m, nil
}

func (m *editModel) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "n", "esc", "backspace":
			m.confirmDeleteKey = ""
			m.screen = screenKeyList
			return m, nil
		case "y":
			return m.doDelete()
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *editModel) doDelete() (tea.Model, tea.Cmd) {
	b, err := m.openBackend(m.namespace)
	if err != nil {
		// Show error but stay on list
		m.keyError = err
		m.screen = screenKeyList
		return m, nil
	}

	// Note: Backend interface doesn't have Delete, so we set empty value
	// or rely on backend-specific behavior. For now, set to empty string.
	if err := b.Set(m.namespace, m.confirmDeleteKey, ""); err != nil {
		m.keyError = err
		m.screen = screenKeyList
		return m, nil
	}

	m.confirmDeleteKey = ""
	m.loadKeys()
	if m.selectedKey >= len(m.keys) && m.selectedKey > 0 {
		m.selectedKey--
	}
	m.screen = screenKeyList
	return m, nil
}

// --- view ---

func (m *editModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.screen {
	case screenNamespace:
		return m.viewNamespace()
	case screenKeyList:
		return m.viewKeyList()
	case screenKeyDetail:
		return m.viewKeyDetail()
	case screenEdit:
		return m.viewEdit()
	case screenConfirmDelete:
		return m.viewConfirmDelete()
	}
	return ""
}

func (m *editModel) viewNamespace() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" envoke edit ") + "\n\n")
	b.WriteString(m.nsList.View())
	b.WriteString("\n" + helpStyle.Render("enter: select  q: quit"))
	return b.String()
}

func (m *editModel) viewKeyList() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf(" Namespace: %s ", m.namespace)) + "\n\n")

	if m.keyError != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.keyError)) + "\n\n")
	}

	if len(m.keys) == 0 {
		b.WriteString("(no keys)\n")
	} else {
		for i, k := range m.keys {
			prefix := "  "
			if i == m.selectedKey {
				prefix = "> "
			}
			b.WriteString(prefix + k + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("enter: view  a: add  d: delete  esc: back  q: quit"))
	return b.String()
}

func (m *editModel) viewKeyDetail() string {
	var b strings.Builder
	key := ""
	if m.selectedKey < len(m.keys) {
		key = m.keys[m.selectedKey]
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf(" Key: %s ", key)) + "\n\n")

	if m.keyError != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.keyError)) + "\n")
	} else {
		val := m.keyValue
		if !m.revealed {
			if val == "" {
				val = "(empty)"
			} else {
				val = strings.Repeat("*", len(val))
			}
		}
		b.WriteString("Value: " + valueStyle.Render(val) + "\n")
	}

	revealedStatus := "hidden"
	if m.revealed {
		revealedStatus = "revealed"
	}
	b.WriteString("\n" + helpStyle.Render(fmt.Sprintf("v: toggle (%s)  e: edit  d: delete  esc: back  q: quit", revealedStatus)))
	return b.String()
}

func (m *editModel) viewEdit() string {
	var b strings.Builder
	action := "Update"
	if m.editIsNew {
		action = "Add"
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf(" %s secret ", action)) + "\n\n")

	if m.editError != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.editError)) + "\n\n")
	}

	b.WriteString("Key:\n" + m.editKeyInput.View() + "\n\n")
	b.WriteString("Value:\n" + m.editValInput.View() + "\n\n")
	b.WriteString(helpStyle.Render("tab: switch field  enter: save  esc: cancel  q: quit"))
	return b.String()
}

func (m *editModel) viewConfirmDelete() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Confirm delete ") + "\n\n")
	b.WriteString(fmt.Sprintf("Delete key %q from namespace %q?\n\n", m.confirmDeleteKey, m.namespace))
	b.WriteString(helpStyle.Render("y: confirm  n/esc: cancel  q: quit"))
	return b.String()
}
