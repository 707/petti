package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nad/pkgview/internal/app"
	"github.com/nad/pkgview/internal/collectors"
	exports "github.com/nad/pkgview/internal/export"
	"github.com/nad/pkgview/internal/model"
)

type ExportFormat string

const (
	ExportTXT  ExportFormat = "txt"
	ExportJSON ExportFormat = "json"
)

type RefreshFunc func(context.Context) collectors.CollectResult
type ExportFunc func(ExportFormat, []model.Package) (string, error)

type Config struct {
	Packages []model.Package
	Statuses []model.CollectorStatus
	Filter   string
	Refresh  RefreshFunc
	Export   ExportFunc
}

type Model struct {
	state         app.State
	filter        textinput.Model
	table         table.Model
	help          help.Model
	keys          KeyMap
	refresh       RefreshFunc
	export        ExportFunc
	exportMenu    bool
	exportChoice  int
	statusMessage string
}

type refreshDoneMsg collectors.CollectResult

type exportDoneMsg struct {
	path string
}

type exportFailedMsg struct {
	err error
}

type KeyMap struct {
	Filter  key.Binding
	Sort    key.Binding
	Export  key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Filter, k.Sort, k.Export, k.Refresh, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Filter, k.Sort, k.Export}, {k.Refresh, k.Help, k.Quit}}
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Sort:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Export:  key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func New(config Config) Model {
	filter := textinput.New()
	filter.Prompt = "> "
	filter.SetValue(config.Filter)

	tbl := table.New(
		table.WithColumns([]table.Column{
			{Title: "Package", Width: 28},
			{Title: "Version", Width: 16},
			{Title: "Source", Width: 18},
		}),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	keys := DefaultKeyMap()
	helpModel := help.New()

	m := Model{
		state: app.State{
			Packages: config.Packages,
			Statuses: config.Statuses,
			Filter:   config.Filter,
		},
		filter:  filter,
		table:   tbl,
		help:    helpModel,
		keys:    keys,
		refresh: config.Refresh,
		export:  config.Export,
	}
	if m.refresh == nil {
		m.refresh = func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{
				Packages: m.state.Packages,
				Statuses: m.state.Statuses,
			}
		}
	}
	if m.export == nil {
		m.export = defaultExportFunc
	}
	m.syncRows()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.Width = msg.Width
		m.state.Height = msg.Height
		m.help.Width = msg.Width
		m.table.SetWidth(max(20, msg.Width-4))
		m.table.SetHeight(max(8, msg.Height-8))
		return m, nil
	case refreshDoneMsg:
		m.state.Packages = msg.Packages
		m.state.Statuses = msg.Statuses
		m.state.IsLoading = false
		m.syncRows()
		return m, nil
	case exportDoneMsg:
		m.exportMenu = false
		m.statusMessage = "exported to " + msg.path
		return m, nil
	case exportFailedMsg:
		m.exportMenu = false
		m.statusMessage = msg.err.Error()
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.exportMenu {
		return m.updateExportMenu(keyMsg)
	}

	if m.filter.Focused() {
		switch keyMsg.Type {
		case tea.KeyEsc:
			m.filter.SetValue("")
			m.filter.Blur()
			m.table.Focus()
			m.state.Filter = ""
			m.syncRows()
			return m, nil
		case tea.KeyEnter:
			m.filter.Blur()
			m.table.Focus()
			return m, nil
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(keyMsg)
		m.state.Filter = m.filter.Value()
		m.syncRows()
		return m, cmd
	}

	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.state.ShowHelp = !m.state.ShowHelp
		return m, nil
	case "s":
		m.state.CycleSort()
		m.syncRows()
		return m, nil
	case "r":
		m.state.IsLoading = true
		return m, refreshCmd(m.refresh)
	case "e":
		if len(m.state.VisiblePackages()) == 0 {
			m.statusMessage = "nothing to export"
			return m, nil
		}
		m.exportMenu = true
		m.exportChoice = 0
		return m, nil
	case "/":
		m.focusFilter()
		return m, nil
	case "esc":
		m.state.Filter = ""
		m.filter.SetValue("")
		m.syncRows()
		return m, nil
	}

	if shouldStartFilter(keyMsg) {
		m.focusFilter()
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(keyMsg)
		m.state.Filter = m.filter.Value()
		m.syncRows()
		return m, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(keyMsg)
	m.state.Selected = m.table.Cursor()
	return m, cmd
}

func (m Model) View() string {
	header := lipgloss.NewStyle().Bold(true).Render("pkgview")
	filterLine := "Filter: " + m.filter.View()

	body := m.table.View()
	if m.state.TooSmall() {
		body = "Terminal too small - resize to at least 80x24"
	}
	if m.exportMenu {
		body += "\n\n" + m.renderExportMenu()
	}

	footer := m.renderFooter()
	m.help.ShowAll = m.state.ShowHelp
	helpView := m.help.View(m.keys)

	return strings.Join([]string{header, filterLine, body, footer, helpView}, "\n")
}

func (m *Model) focusFilter() {
	m.table.Blur()
	m.filter.Focus()
}

func (m *Model) syncRows() {
	rows := make([]table.Row, 0, len(m.state.VisiblePackages()))
	for _, pkg := range m.state.VisiblePackages() {
		rows = append(rows, table.Row{pkg.Name, pkg.Version, string(pkg.Source)})
	}
	m.state.ClampSelection()
	m.table.SetRows(rows)
	if len(rows) > 0 {
		m.table.SetCursor(m.state.Selected)
	}
}

func (m Model) renderFooter() string {
	counts := m.state.SummaryCounts()
	parts := []string{
		fmt.Sprintf("%d packages", len(m.state.VisiblePackages())),
		fmt.Sprintf("homebrew: %d", counts[model.SourceHomebrew]),
		fmt.Sprintf("casks: %d", counts[model.SourceHomebrewCask]),
		fmt.Sprintf("npm: %d", counts[model.SourceNPM]),
		fmt.Sprintf("pip: %d", counts[model.SourcePip]),
	}
	if m.state.IsLoading {
		parts = append(parts, "refreshing...")
	}
	if m.statusMessage != "" {
		parts = append(parts, m.statusMessage)
	}
	return strings.Join(parts, "  |  ")
}

func (m Model) renderExportMenu() string {
	formats := []ExportFormat{ExportTXT, ExportJSON}
	lines := []string{"Export"}
	for i, format := range formats {
		prefix := "  "
		if i == m.exportChoice {
			prefix = "> "
		}
		lines = append(lines, prefix+string(format))
	}
	return strings.Join(lines, "\n")
}

func (m Model) updateExportMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.exportMenu = false
		return m, nil
	case "down", "j":
		m.exportChoice = (m.exportChoice + 1) % 2
		return m, nil
	case "up", "k":
		m.exportChoice = (m.exportChoice + 1) % 2
		return m, nil
	case "enter":
		packages := append([]model.Package(nil), m.state.VisiblePackages()...)
		return m, exportCmd(m.export, selectedFormat(m.exportChoice), packages)
	default:
		return m, nil
	}
}

func selectedFormat(choice int) ExportFormat {
	if choice == 1 {
		return ExportJSON
	}
	return ExportTXT
}

func refreshCmd(refresh RefreshFunc) tea.Cmd {
	return func() tea.Msg {
		return refreshDoneMsg(refresh(context.Background()))
	}
}

func exportCmd(export ExportFunc, format ExportFormat, packages []model.Package) tea.Cmd {
	return func() tea.Msg {
		path, err := export(format, packages)
		if err != nil {
			return exportFailedMsg{err: err}
		}
		return exportDoneMsg{path: path}
	}
}

func defaultExportFunc(format ExportFormat, packages []model.Package) (string, error) {
	switch format {
	case ExportJSON:
		path := filepath.Clean("pkgview-export.json")
		return path, exports.WriteJSON(path, packages)
	default:
		path := filepath.Clean("pkgview-export.txt")
		return path, exports.WriteTXT(path, packages)
	}
}

func shouldStartFilter(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes || len(msg.Runes) != 1 {
		return false
	}
	switch msg.String() {
	case "q", "?", "s", "e", "r", "/":
		return false
	default:
		return true
	}
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
