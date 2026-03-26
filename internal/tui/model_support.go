package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/707/petti/internal/app"
	"github.com/707/petti/internal/model"
)

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
	if m.state.IsLoading {
		return "refreshing " + m.loadingIndicator()
	}
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

func (m Model) loadingIndicator() string {
	return loadingFrames[m.loadingFrame%len(loadingFrames)]
}

func (m Model) modeLabel() string {
	if !m.state.IsLoading {
		return string(m.mode)
	}
	return string(m.mode) + " " + m.loadingIndicator()
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
		return []string{"Filter " + string(m.popupKind), "j/k:move", "Enter:apply", "Esc:back", "q:quit"}
	}
	switch m.mode {
	case ModeFilter:
		return []string{"type:search", "Enter:apply", "Esc:back", "q:quit"}
	case ModeColumn:
		return []string{"h/l:move", "Enter:apply", "Esc:back", "q:quit"}
	case ModeExport:
		return []string{"j/k:pick", "Enter:write", "Esc:back", "q:quit"}
	default:
		return []string{
			"/:search",
			"f:source",
			"a:action",
			"u:updated",
			"V:select",
			"Enter:detail",
			"s:sort",
			"t:theme",
			"e:export",
			"d: copy uninstall cmd",
			"i:info",
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
	return false
}

func (m Model) openInfoModal() (Model, tea.Cmd) {
	pkg := m.selectedPackage()
	if pkg == nil {
		m.statusMessage = "nothing selected"
		return m, nil
	}
	m.infoOpen = true
	m.infoLoading = true
	m.infoKey = pkg.Key()
	m.infoDetails = model.PackageDetails{}
	m.infoError = ""
	return m, inspectCmd(m.inspect, *pkg)
}

func (m *Model) closeInfoModal() {
	m.infoOpen = false
	m.infoLoading = false
	m.infoError = ""
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
		values = append(values, gridValueForColumn(pkg, column.key))
	}
	return values
}

func renderGridHeaderLines(columns []gridColumn, mode Mode, focus int) [][]string {
	lines := [][]string{
		make([]string, 0, len(columns)),
		make([]string, 0, len(columns)),
	}
	interactiveIndex := 0
	for _, column := range columns {
		headerLines := column.headerLines
		if len(headerLines) == 0 {
			headerLines = []string{column.title, ""}
		}
		first := headerLines[0]
		second := ""
		if len(headerLines) > 1 {
			second = headerLines[1]
		}
		if column.interactive && mode == ModeColumn && focus == interactiveIndex {
			first = "[" + trimToWidth(first, max(1, column.width-2)) + "]"
		}
		lines[0] = append(lines[0], trimToWidth(first, column.width))
		lines[1] = append(lines[1], trimToWidth(second, column.width))
		if column.interactive {
			interactiveIndex++
		}
	}
	return lines
}

func renderGridBorderTop(columns []gridColumn) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, "─"+strings.Repeat("─", column.width)+"─")
	}
	if len(parts) == 0 {
		return ""
	}
	return "┌" + strings.Join(parts, "┬") + "┐"
}

func renderGridBorderBottom(columns []gridColumn) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, "─"+strings.Repeat("─", column.width)+"─")
	}
	if len(parts) == 0 {
		return ""
	}
	return "└" + strings.Join(parts, "┴") + "┘"
}

func renderGridBorder(columns []gridColumn) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, "─"+strings.Repeat("─", column.width)+"─")
	}
	if len(parts) == 0 {
		return ""
	}
	return "┌" + strings.Join(parts, "┬") + "┐"
}

func renderGridASCII(columns []gridColumn, values []string) string {
	parts := make([]string, 0, len(columns))
	for index, column := range columns {
		value := ""
		if index < len(values) {
			value = values[index]
		}
		displayValue := value
		if !column.noTruncate || lipgloss.Width(value) > column.width {
			displayValue = trimToWidth(value, column.width)
		}
		parts = append(parts, " "+padRight(displayValue, column.width)+" ")
	}
	return "│" + strings.Join(parts, "│") + "│"
}

func displayGridValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

var relativeNow = time.Now

func formatRelativeTime(dateStr string) string {
	if strings.TrimSpace(dateStr) == "" {
		return "-"
	}
	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	duration := relativeNow().Sub(parsed)
	days := int(duration.Hours() / 24)
	if days < 0 {
		return dateStr
	}
	if days >= 365 {
		years := days / 365
		if years == 1 {
			return "1year"
		}
		return fmt.Sprintf("%dyears", years)
	}
	if days >= 30 {
		return fmt.Sprintf("%dm", days/30)
	}
	return fmt.Sprintf("%dd", days)
}

func (m Model) palette() themePalette {
	return paletteForTheme(m.theme)
}

func (m Model) appStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(m.palette().appForeground)
}

func (m Model) panelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.palette().border).
		Background(m.palette().panelBackground).
		Foreground(m.palette().appForeground)
}

func (m Model) titleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(m.palette().topForeground)
}

func (m Model) statusStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(m.palette().statusForeground)
}

func (m Model) gridHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.palette().headerForeground).
		Background(m.palette().headerBackground).
		Bold(true)
}

func (m Model) gridCellStyle(columnKey string, pkg model.Package, selected bool) lipgloss.Style {
	if selected {
		return lipgloss.NewStyle().
			Foreground(m.palette().selectedForeground).
			Background(m.palette().selectedBackground).
			Bold(true)
	}

	style := lipgloss.NewStyle().Foreground(m.palette().mutedForeground)
	switch columnKey {
	case "pkg", "ver", "src":
		style = style.Foreground(m.sourceForeground(pkg.Source))
	case "usedby":
		switch pkg.UsedBy {
		case "Y":
			style = style.Foreground(m.palette().dependencyYes)
		case "N":
			style = style.Foreground(m.palette().dependencyNo)
		}
	}
	return style
}

func (m Model) sourceForeground(source model.Source) lipgloss.Color {
	switch source {
	case model.SourceHomebrew:
		return m.palette().homebrewForeground
	case model.SourceHomebrewCask:
		return m.palette().caskForeground
	case model.SourceNPM:
		return m.palette().npmForeground
	case model.SourcePip:
		return m.palette().pipForeground
	default:
		return m.palette().mutedForeground
	}
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

func inspectCmd(inspect InspectFunc, pkg model.Package) tea.Cmd {
	return func() tea.Msg {
		if inspect == nil {
			return infoLoadedMsg{key: pkg.Key()}
		}
		details, err := inspect(context.Background(), pkg)
		return infoLoadedMsg{key: pkg.Key(), details: details, err: err}
	}
}

func maxRenderedLineWidth(value string) int {
	width := 0
	for _, line := range strings.Split(value, "\n") {
		width = max(width, lipgloss.Width(line))
	}
	return width
}

func overlayCentered(width, height int, overlay string) string {
	if strings.TrimSpace(overlay) == "" {
		return ""
	}
	return lipgloss.Place(
		max(1, width),
		max(1, height),
		lipgloss.Center,
		lipgloss.Center,
		overlay,
	)
}

func (m Model) gridRowPoint(index int) (int, int) {
	gridStartY := lipgloss.Height(m.renderTopBar()) + 1 + lipgloss.Height(m.renderFilterStrip()) + 1
	if m.popupKind != PopupNone {
		gridStartY += lipgloss.Height(m.renderPopupMenu()) + 2
	}
	gridStartY += lipgloss.Height(m.renderGridHeader())
	return 2, gridStartY + index
}

func (m Model) gridRowIndexAt(x, y int) int {
	if x < 0 || y < 0 {
		return -1
	}
	gridWidth, _ := m.fullGridSize()
	if x >= gridWidth {
		return -1
	}
	_, firstRowY := m.gridRowPoint(0)
	index := y - firstRowY + m.grid.YOffset
	if index < 0 || index >= len(m.state.VisiblePackages()) {
		return -1
	}
	return index
}

func (m Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.layout != LayoutFull || msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil
	}
	index := m.gridRowIndexAt(msg.X, msg.Y)
	if index < 0 {
		return m, nil
	}
	if m.state.Selected == index {
		if m.infoOpen {
			m.closeInfoModal()
			return m, nil
		}
		return m.openInfoModal()
	}
	m.state.Selected = index
	m.syncRows()
	m.closeInfoModal()
	return m, nil
}
