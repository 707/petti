package tui

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nad/pkgview/internal/app"
	exports "github.com/nad/pkgview/internal/export"
	"github.com/nad/pkgview/internal/model"
)

func (m *Model) resizeGrid() {
	width, height := m.fullGridSize()
	m.grid.Width = width
	m.grid.Height = height
}

func (m Model) fullGridSize() (int, int) {
	totalWidth := m.viewWidth()
	gridWidth := max(20, totalWidth-4)
	totalHeight := m.state.Height
	if totalHeight <= 0 {
		totalHeight = 30
	}
	gridHeight := max(6, totalHeight-12)
	if m.detailOpen {
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
	lines := []string{renderGridBorderTop(columns)}
	for _, values := range renderGridHeaderLines(columns, m.mode, m.columnFocus) {
		lines = append(lines, renderGridASCII(columns, values))
	}
	lines = append(lines, renderGridBorderBottom(columns))
	return m.gridHeaderStyle().Render(strings.Join(lines, "\n"))
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
		lines = append(lines, m.renderStyledGridRow(pkg, columns, index == m.state.Selected))
	}
	return strings.Join(lines, "\n")
}

func (m Model) fullGridColumns() []gridColumn {
	columns := []gridColumn{
		{key: "pkg", title: "PKG", width: len("PKG"), interactive: true, minWidth: len("PKG") + 5},
		{key: "ver", title: "VER", width: len("VER"), interactive: true},
		{key: "src", title: "SRC", width: len("SRC"), interactive: true, noTruncate: true, minWidth: len("SRC") + 4},
	}
	if m.grid.Width >= 80 {
		columns = append(columns, gridColumn{key: "desc", title: "DESCRIPTION", width: len("DESCRIPTION"), minWidth: len("DESCRIPTION")})
	}
	columns = append(columns, gridColumn{key: "updated", title: "LAST UPDATED", width: len("LAST UPDATED"), interactive: true, minWidth: len("LAST UPDATED"), fixed: true})
	columns = append(columns, gridColumn{key: "status", title: "STATUS", width: len("STATUS"), interactive: true, minWidth: len("STATUS") + 4})
	columns = append(columns, gridColumn{key: "usedby", title: "DEPENDANCY", width: 6, minWidth: 6, fixed: true, headerLines: []string{"DEPEN", "DANCY"}})
	m.calculateDynamicColumnWidths(columns)
	return columns
}

func (m Model) calculateDynamicColumnWidths(columns []gridColumn) {
	visible := m.state.VisiblePackages()
	for i, col := range columns {
		maxWidth := lipgloss.Width(col.title)
		for _, pkg := range visible {
			var valLen int
			switch col.key {
			case "pkg":
				valLen = lipgloss.Width(pkg.Name)
			case "ver":
				valLen = lipgloss.Width(displayGridValue(pkg.Version))
			case "src":
				valLen = lipgloss.Width(displayGridValue(string(pkg.Source)))
			case "updated":
				valLen = lipgloss.Width(formatRelativeTime(pkg.UpdatedAt))
			case "status":
				valLen = lipgloss.Width(displayGridValue(pkg.ActionRequired))
			case "desc":
				valLen = lipgloss.Width(displayGridValue(pkg.Description))
			case "usedby":
				valLen = lipgloss.Width(displayGridValue(pkg.UsedBy))
			}
			if valLen > maxWidth {
				maxWidth = valLen
			}
		}
		if col.fixed {
			columns[i].width = col.width
			continue
		}
		columns[i].width = desiredColumnWidth(col.key, maxWidth)
	}

	maxContentWidth := max(20, m.grid.Width-(len(columns)*3+1))
	excess := gridInnerWidth(columns) - maxContentWidth
	if excess <= 0 {
		return
	}

	excess = shrinkColumns(columns, excess, false, []string{"desc"})
	excess = shrinkColumns(columns, excess, false, []string{"ver", "updated"})
	excess = shrinkColumns(columns, excess, false, []string{"pkg"})
	excess = shrinkColumns(columns, excess, false, []string{"status"})
	excess = shrinkColumns(columns, excess, false, []string{"usedby"})
	excess = shrinkColumns(columns, excess, false, []string{"src"})
	if excess > 0 {
		shrinkColumns(columns, excess, true, []string{"desc", "ver", "updated", "pkg", "status", "usedby", "src"})
	}
}

func desiredColumnWidth(key string, natural int) int {
	if key == "desc" {
		reduced := natural - natural/4
		return max(lipgloss.Width("DESCRIPTION"), reduced)
	}
	if key == "ver" {
		return max(lipgloss.Width("VER"), max(3, natural/2))
	}
	return natural
}

func (m Model) renderStyledGridRow(pkg model.Package, columns []gridColumn, selected bool) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		value := gridValueForColumn(pkg, column.key)
		displayValue := value
		if !column.noTruncate || lipgloss.Width(value) > column.width {
			displayValue = trimToWidth(value, column.width)
		}
		displayValue = padRight(displayValue, column.width)
		style := m.gridCellStyle(column.key, pkg, selected)
		parts = append(parts, " "+style.Render(displayValue)+" ")
	}
	return "│" + strings.Join(parts, "│") + "│"
}

func gridValueForColumn(pkg model.Package, key string) string {
	switch key {
	case "pkg":
		return pkg.Name
	case "ver":
		return displayGridValue(pkg.Version)
	case "src":
		return displayGridValue(string(pkg.Source))
	case "updated":
		return formatRelativeTime(pkg.UpdatedAt)
	case "status":
		return displayGridValue(pkg.ActionRequired)
	case "desc":
		return displayGridValue(pkg.Description)
	case "usedby":
		return displayGridValue(pkg.UsedBy)
	default:
		return ""
	}
}

func gridInnerWidth(columns []gridColumn) int {
	total := 0
	for _, col := range columns {
		total += col.width
	}
	return total
}

func shrinkColumns(columns []gridColumn, excess int, includeProtected bool, keys []string) int {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}
	for excess > 0 {
		shrunk := false
		for i := range columns {
			if len(keySet) > 0 {
				if _, ok := keySet[columns[i].key]; !ok {
					continue
				}
			}
			if columns[i].noTruncate && !includeProtected {
				continue
			}
			minWidth := columns[i].minWidth
			if minWidth == 0 {
				minWidth = lipgloss.Width(columns[i].title)
			}
			if columns[i].width <= minWidth {
				continue
			}
			columns[i].width--
			excess--
			shrunk = true
			if excess == 0 {
				break
			}
		}
		if !shrunk {
			return excess
		}
	}
	return excess
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
	noTruncate  bool
	minWidth    int
	fixed       bool
	headerLines []string
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
		case "status":
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

var loadingFrames = []string{"-", "\\", "|", "/"}

func loadingTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return loadingTickMsg{}
	})
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
	case "q", "?", "s", "t", "e", "d", "i", "r", "/", "f", "a", "u", "j", "k", "g", "G", "V":
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
