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
	"github.com/charmbracelet/bubbles/viewport"
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

type LayoutMode string

const (
	LayoutFull    LayoutMode = "full"
	LayoutCompact LayoutMode = "compact"
)

type Mode string

const (
	ModeNormal Mode = "NORMAL"
	ModeFilter Mode = "FILTER"
	ModeColumn Mode = "SELECT"
	ModeExport Mode = "EXPORT"
)

type PopupKind string

const (
	PopupNone    PopupKind = ""
	PopupSource  PopupKind = "SOURCE"
	PopupAction  PopupKind = "ACTION"
	PopupUpdated PopupKind = "UPDATED"
)

type RefreshFunc func(context.Context) collectors.CollectResult
type ExportFunc func(ExportFormat, []model.Package) (string, error)

type Config struct {
	Packages []model.Package
	Statuses []model.CollectorStatus
	Filter   string
	Layout   LayoutMode
	Refresh  RefreshFunc
	Export   ExportFunc
}

type Model struct {
	state         app.State
	filter        textinput.Model
	table         table.Model
	grid          viewport.Model
	help          help.Model
	keys          KeyMap
	layout        LayoutMode
	mode          Mode
	refresh       RefreshFunc
	export        ExportFunc
	exportMenu    bool
	exportChoice  int
	columnFocus   int
	popupKind     PopupKind
	popupChoice   int
	detailOpen    bool
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
	grid := viewport.New(0, 0)

	m := Model{
		state: app.State{
			Packages: config.Packages,
			Statuses: config.Statuses,
			Filter:   config.Filter,
		},
		filter:  filter,
		table:   tbl,
		grid:    grid,
		help:    helpModel,
		keys:    keys,
		layout:  config.Layout,
		mode:    ModeNormal,
		refresh: config.Refresh,
		export:  config.Export,
	}
	if m.layout == "" {
		m.layout = LayoutFull
	}
	m.detailOpen = false
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
		m.resizeGrid()
		m.syncRows()
		return m, nil
	case refreshDoneMsg:
		m.state.Packages = msg.Packages
		m.state.Statuses = msg.Statuses
		m.state.IsLoading = false
		m.syncRows()
		return m, nil
	case exportDoneMsg:
		m.exportMenu = false
		m.mode = ModeNormal
		m.statusMessage = "exported to " + msg.path
		return m, nil
	case exportFailedMsg:
		m.exportMenu = false
		m.mode = ModeNormal
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

	if m.popupKind != PopupNone {
		return m.updatePopupMenu(keyMsg)
	}

	if m.mode == ModeColumn {
		return m.updateColumnMode(keyMsg)
	}

	if m.filter.Focused() {
		switch keyMsg.Type {
		case tea.KeyEsc:
			m.filter.SetValue("")
			m.filter.Blur()
			m.table.Focus()
			m.mode = ModeNormal
			m.state.Filter = ""
			m.syncRows()
			return m, nil
		case tea.KeyEnter:
			m.filter.Blur()
			m.table.Focus()
			m.mode = ModeNormal
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
	case "V":
		m.mode = ModeColumn
		m.columnFocus = 0
		return m, nil
	case "f":
		m.openPopup(PopupSource)
		return m, nil
	case "a":
		m.openPopup(PopupAction)
		return m, nil
	case "u":
		m.openPopup(PopupUpdated)
		return m, nil
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
		m.mode = ModeExport
		m.exportChoice = 0
		return m, nil
	case "/":
		m.focusFilter()
		m.mode = ModeFilter
		return m, nil
	case "enter":
		m.detailOpen = !m.detailOpen
		return m, nil
	case "esc":
		m.state.Filter = ""
		m.filter.SetValue("")
		m.syncRows()
		return m, nil
	}

	if shouldStartFilter(keyMsg) {
		m.focusFilter()
		m.mode = ModeFilter
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(keyMsg)
		m.state.Filter = m.filter.Value()
		m.syncRows()
		return m, cmd
	}

	if m.layout == LayoutFull {
		return m.updateFullGridNavigation(keyMsg)
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(keyMsg)
	m.state.Selected = m.table.Cursor()
	return m, cmd
}

func (m Model) View() string {
	if m.layout == LayoutCompact {
		return m.renderCompactView()
	}
	return m.renderFullView()
}

func (m Model) renderCompactView() string {
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

func (m Model) renderFullView() string {
	header := m.renderTopBar()
	filterLine := m.renderFilterStrip()
	body := m.renderFullBody()
	if m.state.TooSmall() {
		body = "Terminal too small - resize to at least 80x24"
	}
	if m.exportMenu {
		body += "\n\n" + m.renderExportMenu()
	}
	if m.popupKind != PopupNone {
		body = m.renderPopupMenu() + "\n\n" + body
	}
	footer := m.renderFooter()
	m.help.ShowAll = m.state.ShowHelp
	return strings.Join([]string{header, filterLine, body, footer, m.renderModeBar()}, "\n")
}

func (m Model) renderFullBody() string {
	totalWidth := m.state.Width
	if totalWidth <= 0 {
		totalWidth = 100
	}
	m.resizeGrid()
	if !m.detailOpen {
		gridWidth, _ := m.fullGridSize()
		return lipgloss.NewStyle().Width(gridWidth).MaxWidth(gridWidth).Render(m.renderGridPane())
	}

	gridWidth, _ := m.fullGridSize()
	gridPane := lipgloss.NewStyle().Width(gridWidth).MaxWidth(gridWidth).Render(m.renderGridPane())
	if !m.detailAsSide() {
		return lipgloss.JoinVertical(lipgloss.Left, gridPane, "", m.renderDetailPane())
	}

	detailWidth := fullDetailWidth(totalWidth)
	gapWidth := 2
	detailPane := lipgloss.NewStyle().Width(detailWidth).MaxWidth(detailWidth).Render(m.renderDetailPane())
	return lipgloss.JoinHorizontal(lipgloss.Top, gridPane, strings.Repeat(" ", gapWidth), detailPane)
}

func (m *Model) focusFilter() {
	m.table.Blur()
	m.filter.Focus()
}

func (m *Model) syncRows() {
	visible := m.state.VisiblePackages()
	rows := make([]table.Row, 0, len(visible))
	for _, pkg := range visible {
		rows = append(rows, table.Row{pkg.Name, pkg.Version, string(pkg.Source)})
	}
	m.state.ClampSelection()
	m.table.SetRows(rows)
	if m.layout == LayoutCompact && len(rows) > 0 {
		m.table.SetCursor(m.state.Selected)
	}
	m.syncGrid()
}

func (m Model) renderFooter() string {
	counts := m.state.SummaryCounts()
	parts := []string{
		"MODE: " + string(m.mode),
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

func (m Model) renderTopBar() string {
	title := lipgloss.NewStyle().Bold(true).Render("pkgview")
	counts := m.state.SummaryCounts()
	center := fmt.Sprintf(
		"Packages %d | brew %d | casks %d | npm %d | pip %d",
		len(m.state.VisiblePackages()),
		counts[model.SourceHomebrew],
		counts[model.SourceHomebrewCask],
		counts[model.SourceNPM],
		counts[model.SourcePip],
	)
	right := "Managers " + m.managerSummary()
	bar := lipgloss.JoinHorizontal(lipgloss.Top, title, "   ", center, "   ", right)
	contentWidth := max(20, m.viewWidth()-2)
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Width(contentWidth).
		MaxWidth(contentWidth).
		Render(bar)
}

func (m Model) renderFilterStrip() string {
	boxWidth := max(16, (m.viewWidth()-4)/5)
	parts := []string{
		m.renderFilterBox("Search (/)", valueOrPlaceholder(m.filter.Value(), "Press / to search"), boxWidth),
		m.renderFilterBox("Source (f)", m.activeSourceLabel(), boxWidth),
		m.renderFilterBox("Action (a)", m.activeActionLabel(), boxWidth),
		m.renderFilterBox("Updated (u)", m.activeUpdatedLabel(), boxWidth),
		m.renderFilterBox("Sort (s)", m.activeSortLabel(), boxWidth),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts[0], " ", parts[1], " ", parts[2], " ", parts[3], " ", parts[4])
}

func (m Model) renderFilterBox(label, value string, width int) string {
	content := label + "\n" + value
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Width(max(width-2, 16)).
		MaxWidth(max(width-2, 16)).
		Render(content)
}

func (m Model) renderModeBar() string {
	parts := m.modeHelpParts()
	contentWidth := max(20, m.viewWidth()-2)
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Width(contentWidth).
		MaxWidth(contentWidth).
		Render(strings.Join(parts, "   "))
}

func (m Model) renderPopupMenu() string {
	options := m.popupOptions()
	lines := []string{"Filter " + string(m.popupKind)}
	for index, option := range options {
		prefix := "  "
		if index == m.popupChoice {
			prefix = "> "
		}
		lines = append(lines, prefix+option)
	}
	width := max(24, min(40, m.viewWidth()/3))
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Width(width).
		MaxWidth(width).
		Render(strings.Join(lines, "\n"))
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

func (m Model) renderSourceTabs() string {
	tabs := []struct {
		label  string
		source model.Source
	}{
		{label: "All", source: ""},
		{label: "Homebrew", source: model.SourceHomebrew},
		{label: "Casks", source: model.SourceHomebrewCask},
		{label: "npm", source: model.SourceNPM},
		{label: "pip", source: model.SourcePip},
	}
	parts := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		label := tab.label
		if m.state.SourceFilter == tab.source {
			label = "[" + label + "]"
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, "  ")
}

func (m Model) renderDetailPane() string {
	pkg := m.selectedPackage()
	if pkg == nil {
		return "Selected Package\nNo package selected"
	}
	lines := []string{
		"Selected Package",
		"Name: " + pkg.Name,
		"Version: " + pkg.Version,
		"Source: " + string(pkg.Source),
		"Updated: " + displayGridValue(pkg.UpdatedAt),
		"Action: " + displayGridValue(pkg.ActionRequired),
		"Last Used: " + displayGridValue(pkg.LastUsed),
		"Description: " + displayGridValue(pkg.Description),
		"Uninstall: " + uninstallCommand(*pkg),
	}
	return strings.Join(lines, "\n")
}

func (m *Model) resizeGrid() {
	width, height := m.fullGridSize()
	m.grid.Width = width
	m.grid.Height = height
}

func (m Model) fullGridSize() (int, int) {
	totalWidth := m.viewWidth()
	gridWidth := max(20, totalWidth-4)
	if m.detailOpen && m.detailAsSide() {
		gridWidth = max(20, totalWidth-fullDetailWidth(totalWidth)-2)
	}
	totalHeight := m.state.Height
	if totalHeight <= 0 {
		totalHeight = 30
	}
	gridHeight := max(6, totalHeight-12)
	if m.detailOpen && !m.detailAsSide() {
		gridHeight = max(6, gridHeight-7)
	}
	if m.popupKind != PopupNone {
		gridHeight = max(6, gridHeight-6)
	}
	return gridWidth, gridHeight
}

func (m *Model) syncGrid() {
	m.resizeGrid()
	m.grid.SetContent(m.renderGridRows())
	m.syncGridOffset()
}

func (m *Model) syncGridOffset() {
	if m.grid.Height <= 0 {
		return
	}
	if m.state.Selected < m.grid.YOffset {
		m.grid.SetYOffset(m.state.Selected)
		return
	}
	if m.state.Selected >= m.grid.YOffset+m.grid.Height {
		m.grid.SetYOffset(m.state.Selected - m.grid.Height + 1)
	}
}

func (m Model) renderGridPane() string {
	return strings.Join([]string{m.renderGridHeader(), m.grid.View()}, "\n")
}

func (m Model) renderGridHeader() string {
	columns := m.fullGridColumns()
	titles := make([]string, 0, len(columns))
	for index, column := range columns {
		title := column.title
		if column.interactive && m.mode == ModeColumn && m.columnFocus == index {
			title = "[" + title + "]"
		}
		titles = append(titles, trimToWidth(title, column.width))
	}
	return strings.Join([]string{
		renderGridASCII(columns, titles),
		renderGridBorder(columns),
	}, "\n")
}

func (m Model) renderGridRows() string {
	visible := m.state.VisiblePackages()
	if len(visible) == 0 {
		return lipgloss.NewStyle().
			Width(m.grid.Width).
			MaxWidth(m.grid.Width).
			Render("No packages match the current view")
	}

	columns := m.fullGridColumns()
	lines := make([]string, 0, len(visible))
	for index, pkg := range visible {
		values := gridRowValues(pkg, columns)
		row := renderGridASCII(columns, values)
		rowStyle := lipgloss.NewStyle().Width(m.grid.Width).MaxWidth(m.grid.Width).Foreground(lipgloss.Color("248"))
		if index == m.state.Selected {
			rowStyle = rowStyle.Background(lipgloss.Color("63")).Foreground(lipgloss.Color("255")).Bold(true)
		}
		lines = append(lines, rowStyle.Render(row))
	}
	return strings.Join(lines, "\n")
}

func (m Model) fullGridColumns() []gridColumn {
	columns := []gridColumn{
		{key: "pkg", title: "PKG", width: 12, interactive: true},
		{key: "ver", title: "VER", width: 7, interactive: true},
		{key: "src", title: "SRC", width: 6, interactive: true},
		{key: "updated", title: "UPDATED", width: 8, interactive: true},
		{key: "action", title: "ACTION", width: 6, interactive: true},
	}
	if m.grid.Width >= 90 {
		columns = append(columns, gridColumn{key: "desc", title: "DESC", width: 12})
	}
	if m.grid.Width >= 108 {
		columns = append(columns, gridColumn{key: "used", title: "USED", width: 8})
	}
	usable := max(24, m.grid.Width-(len(columns)*3+1))
	fixed := 27
	if hasColumn(columns, "desc") {
		fixed += 12
	}
	if hasColumn(columns, "used") {
		fixed += 8
	}
	nameWidth := min(20, max(10, usable/3))
	descWidth := usable - fixed - nameWidth
	columns[0].width = nameWidth
	for index := range columns {
		if columns[index].key == "desc" {
			columns[index].width = max(8, descWidth)
		}
	}
	return columns
}

func (m Model) updateFullGridNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "down", "j":
		m.moveSelection(1)
	case "up", "k":
		m.moveSelection(-1)
	case "pgdown":
		m.moveSelection(max(1, m.grid.Height))
	case "pgup":
		m.moveSelection(-max(1, m.grid.Height))
	case "g":
		m.state.Selected = 0
		m.syncRows()
	case "G":
		m.state.Selected = max(0, len(m.state.VisiblePackages())-1)
		m.syncRows()
	default:
		return m, nil
	}
	return m, nil
}

func (m *Model) moveSelection(delta int) {
	if len(m.state.VisiblePackages()) == 0 {
		return
	}
	m.state.Selected += delta
	m.syncRows()
}

type gridColumn struct {
	key         string
	title       string
	width       int
	interactive bool
}

func (m Model) updateExportMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.exportMenu = false
		m.mode = ModeNormal
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

func (m Model) updateColumnMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	columns := m.interactiveColumns()
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = ModeNormal
		return m, nil
	case "right", "l":
		m.columnFocus = (m.columnFocus + 1) % len(columns)
		return m, nil
	case "left", "h":
		if m.columnFocus == 0 {
			m.columnFocus = len(columns) - 1
		} else {
			m.columnFocus--
		}
		return m, nil
	case "enter":
		switch columns[m.columnFocus].key {
		case "pkg":
			m.focusFilter()
			m.mode = ModeFilter
		case "ver":
			m.state.SortMode = app.SortVersion
			m.syncRows()
			m.mode = ModeNormal
		case "src":
			m.openPopup(PopupSource)
			m.mode = ModeNormal
		case "updated":
			m.state.SortMode = app.SortUpdated
			m.syncRows()
			m.mode = ModeNormal
		case "action":
			m.openPopup(PopupAction)
			m.mode = ModeNormal
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updatePopupMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	options := m.popupOptions()
	if len(options) == 0 {
		m.popupKind = PopupNone
		return m, nil
	}
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.popupKind = PopupNone
		return m, nil
	case "down", "j":
		m.popupChoice = (m.popupChoice + 1) % len(options)
		return m, nil
	case "up", "k":
		if m.popupChoice == 0 {
			m.popupChoice = len(options) - 1
		} else {
			m.popupChoice--
		}
		return m, nil
	case "enter":
		m.applyPopupChoice()
		m.popupKind = PopupNone
		m.syncRows()
		return m, nil
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
	case "q", "?", "s", "e", "r", "/", "f", "a", "u", "j", "k", "g", "G", "V":
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

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func fullDetailWidth(totalWidth int) int {
	return min(32, max(24, totalWidth/4))
}

func (m Model) activeSourceLabel() string {
	switch m.state.SourceFilter {
	case model.SourceHomebrew:
		return "Homebrew"
	case model.SourceHomebrewCask:
		return "Casks"
	case model.SourceNPM:
		return "npm"
	case model.SourcePip:
		return "pip"
	default:
		return "All Sources"
	}
}

func (m Model) activeSortLabel() string {
	switch m.state.SortMode {
	case app.SortName:
		return "Name"
	case app.SortVersion:
		return "Version"
	case app.SortSource:
		return "Source"
	case app.SortUpdated:
		return "Updated"
	default:
		return "Default"
	}
}

func (m Model) activeActionLabel() string {
	switch m.state.ActionFilter {
	case app.ActionFilterUpdate:
		return "Update"
	case app.ActionFilterAttention:
		return "Attention"
	case app.ActionFilterCurrent:
		return "Current"
	case app.ActionFilterUnknown:
		return "Unknown"
	default:
		return "All Actions"
	}
}

func (m Model) activeUpdatedLabel() string {
	switch m.state.UpdatedFilter {
	case app.UpdatedFilterKnown:
		return "Known"
	case app.UpdatedFilterUnknown:
		return "Unknown"
	default:
		return "All Dates"
	}
}

func (m Model) managerSummary() string {
	if len(m.state.Statuses) == 0 {
		return "not checked"
	}
	parts := make([]string, 0, len(m.state.Statuses))
	for _, status := range m.state.Statuses {
		label := status.Label
		if label == "" {
			label = string(status.Source)
		}
		parts = append(parts, label+":"+string(status.State))
	}
	return strings.Join(parts, " | ")
}

func valueOrPlaceholder(value, placeholder string) string {
	if strings.TrimSpace(value) == "" {
		return placeholder
	}
	return value
}

func filterBoxWidth(label, value string) int {
	return max(18, max(lipgloss.Width(label), lipgloss.Width(value))+2)
}

func (m Model) viewWidth() int {
	if m.state.Width > 0 {
		return m.state.Width
	}
	return 100
}

func (m Model) selectedPackage() *model.Package {
	visible := m.state.VisiblePackages()
	if len(visible) == 0 {
		return nil
	}
	index := m.state.Selected
	if index < 0 || index >= len(visible) {
		index = 0
	}
	return &visible[index]
}

func uninstallCommand(pkg model.Package) string {
	switch pkg.Source {
	case model.SourceHomebrew:
		return "brew uninstall " + pkg.Name
	case model.SourceHomebrewCask:
		return "brew uninstall --cask " + pkg.Name
	case model.SourceNPM:
		return "npm uninstall -g " + pkg.Name
	default:
		return "pip uninstall " + pkg.Name
	}
}

func (m Model) modeHelpParts() []string {
	if m.popupKind != PopupNone {
		return []string{"FILTER:" + string(m.popupKind), "j/k:move", "Enter:apply", "Esc:back", "q:quit"}
	}
	switch m.mode {
	case ModeFilter:
		return []string{"MODE:" + string(m.mode), "type:search", "Enter:apply", "Esc:back", "q:quit"}
	case ModeColumn:
		return []string{"MODE:" + string(m.mode), "h/l:move", "Enter:apply", "Esc:back", "q:quit"}
	case ModeExport:
		return []string{"MODE:" + string(m.mode), "j/k:pick", "Enter:write", "Esc:back", "q:quit"}
	default:
		return []string{
			"MODE:" + string(m.mode),
			"/:search",
			"f:source",
			"a:action",
			"u:updated",
			"V:select",
			"Enter:detail",
			"s:sort",
			"e:export",
			"r:refresh",
			"?:help",
			"q:quit",
		}
	}
}

func (m *Model) openPopup(kind PopupKind) {
	m.popupKind = kind
	m.popupChoice = m.currentPopupChoice(kind)
}

func (m Model) currentPopupChoice(kind PopupKind) int {
	switch kind {
	case PopupSource:
		switch m.state.SourceFilter {
		case model.SourceHomebrew:
			return 1
		case model.SourceHomebrewCask:
			return 2
		case model.SourceNPM:
			return 3
		case model.SourcePip:
			return 4
		default:
			return 0
		}
	case PopupAction:
		switch m.state.ActionFilter {
		case app.ActionFilterUpdate:
			return 1
		case app.ActionFilterAttention:
			return 2
		case app.ActionFilterCurrent:
			return 3
		case app.ActionFilterUnknown:
			return 4
		default:
			return 0
		}
	case PopupUpdated:
		switch m.state.UpdatedFilter {
		case app.UpdatedFilterKnown:
			return 1
		case app.UpdatedFilterUnknown:
			return 2
		default:
			return 0
		}
	default:
		return 0
	}
}

func (m Model) popupOptions() []string {
	switch m.popupKind {
	case PopupSource:
		return []string{"All Sources", "Homebrew", "Casks", "npm", "pip"}
	case PopupAction:
		return []string{"All Actions", "Update", "Attention", "Current", "Unknown"}
	case PopupUpdated:
		return []string{"All Dates", "Known", "Unknown"}
	default:
		return nil
	}
}

func (m *Model) applyPopupChoice() {
	switch m.popupKind {
	case PopupSource:
		switch m.popupChoice {
		case 1:
			m.state.SourceFilter = model.SourceHomebrew
		case 2:
			m.state.SourceFilter = model.SourceHomebrewCask
		case 3:
			m.state.SourceFilter = model.SourceNPM
		case 4:
			m.state.SourceFilter = model.SourcePip
		default:
			m.state.SourceFilter = ""
		}
	case PopupAction:
		switch m.popupChoice {
		case 1:
			m.state.ActionFilter = app.ActionFilterUpdate
		case 2:
			m.state.ActionFilter = app.ActionFilterAttention
		case 3:
			m.state.ActionFilter = app.ActionFilterCurrent
		case 4:
			m.state.ActionFilter = app.ActionFilterUnknown
		default:
			m.state.ActionFilter = app.ActionFilterAll
		}
	case PopupUpdated:
		switch m.popupChoice {
		case 1:
			m.state.UpdatedFilter = app.UpdatedFilterKnown
		case 2:
			m.state.UpdatedFilter = app.UpdatedFilterUnknown
		default:
			m.state.UpdatedFilter = app.UpdatedFilterAll
		}
	}
}

func (m Model) interactiveColumns() []gridColumn {
	columns := m.fullGridColumns()
	interactive := make([]gridColumn, 0, len(columns))
	for _, column := range columns {
		if column.interactive {
			interactive = append(interactive, column)
		}
	}
	return interactive
}

func (m Model) detailAsSide() bool {
	return m.viewWidth() >= 120
}

func hasColumn(columns []gridColumn, key string) bool {
	for _, column := range columns {
		if column.key == key {
			return true
		}
	}
	return false
}

func gridRowValues(pkg model.Package, columns []gridColumn) []string {
	values := make([]string, 0, len(columns))
	for _, column := range columns {
		switch column.key {
		case "pkg":
			values = append(values, pkg.Name)
		case "ver":
			values = append(values, displayGridValue(pkg.Version))
		case "src":
			values = append(values, displayGridValue(string(pkg.Source)))
		case "updated":
			values = append(values, displayGridValue(pkg.UpdatedAt))
		case "action":
			values = append(values, displayGridValue(pkg.ActionRequired))
		case "used":
			values = append(values, displayGridValue(pkg.LastUsed))
		case "desc":
			values = append(values, displayGridValue(pkg.Description))
		}
	}
	return values
}

func renderGridBorder(columns []gridColumn) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, strings.Repeat("-", column.width+2))
	}
	return "+" + strings.Join(parts, "+") + "+"
}

func renderGridASCII(columns []gridColumn, values []string) string {
	parts := make([]string, 0, len(columns))
	for index, column := range columns {
		value := ""
		if index < len(values) {
			value = values[index]
		}
		parts = append(parts, " "+padRight(trimToWidth(value, column.width), column.width)+" ")
	}
	return "|" + strings.Join(parts, "|") + "|"
}

func displayGridValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func trimToWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= width {
		return string(runes)
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func padRight(value string, width int) string {
	gap := width - lipgloss.Width(value)
	if gap <= 0 {
		return value
	}
	return value + strings.Repeat(" ", gap)
}
