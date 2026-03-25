package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/nad/pkgview/internal/model"
)

func (m Model) renderCompactView() string {
	header := m.titleStyle().Render("pkgview")
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
	m.help.ShowAll = m.state.ShowHelp
	parts := []string{header, filterLine, body, m.renderModeBar()}
	if m.state.ShowHelp {
		parts = append(parts, m.help.View(m.keys))
	}
	return m.appStyle().Render(strings.Join(parts, "\n"))
}

func (m Model) renderFullBody() string {
	m.resizeGrid()
	gridWidth, _ := m.fullGridSize()
	gridPane := lipgloss.NewStyle().Width(gridWidth).MaxWidth(gridWidth).Render(m.renderGridPane())
	if !m.detailOpen {
		return gridPane
	}
	return lipgloss.JoinVertical(lipgloss.Left, gridPane, "", m.renderDetailPane())
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
		parts = append(parts, "refreshing "+m.loadingIndicator())
	}
	if m.statusMessage != "" {
		parts = append(parts, m.statusMessage)
	}
	return strings.Join(parts, "  |  ")
}

func (m Model) renderTopBar() string {
	title := m.titleStyle().Render("pkgview")
	counts := m.state.SummaryCounts()
	center := fmt.Sprintf(
		"%s | %d packages | brew %d | casks %d | npm %d | pip %d",
		m.modeLabel(),
		len(m.state.VisiblePackages()),
		counts[model.SourceHomebrew],
		counts[model.SourceHomebrewCask],
		counts[model.SourceNPM],
		counts[model.SourcePip],
	)
	right := "Managers " + m.managerSummary()
	bar := lipgloss.JoinHorizontal(lipgloss.Top, title, "   ", center, "   ", right)
	contentWidth := max(20, m.viewWidth()-2)
	return m.panelStyle().
		BorderBottom(false).
		Background(m.palette().topBackground).
		Foreground(m.palette().topForeground).
		Padding(0, 1).
		Width(contentWidth).
		MaxWidth(contentWidth).
		Render(bar)
}

func (m Model) renderFilterStrip() string {
	totalWidth := max(84, m.viewWidth())
	gapWidth := 1
	available := max(80, totalWidth-(gapWidth*4))
	boxWidth := max(16, available/5)
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
	return m.panelStyle().
		Padding(0, 1).
		Width(max(width-4, 12)).
		MaxWidth(max(width-4, 12)).
		Render(content)
}

func (m Model) renderModeBar() string {
	parts := m.modeHelpParts()
	if m.state.IsLoading {
		parts = append(parts, "refreshing "+m.loadingIndicator())
	}
	if m.statusMessage != "" {
		parts = append(parts, m.statusMessage)
	}
	contentWidth := max(20, m.viewWidth()-2)
	return m.panelStyle().
		Padding(0, 1).
		Width(contentWidth).
		MaxWidth(contentWidth).
		Render(m.statusStyle().Render(strings.Join(parts, "   ")))
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
	return m.panelStyle().
		Padding(0, 1).
		Width(width).
		MaxWidth(width).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderInfoModal() string {
	pkg := m.selectedPackage()
	if pkg == nil {
		return ""
	}
	lines := []string{
		"Package Info",
		"Name: " + displayGridValue(pkg.Name),
		"Source: " + displayGridValue(string(pkg.Source)),
		"Version: " + displayGridValue(pkg.Version),
		"Dependancy: " + displayGridValue(pkg.UsedBy),
		"Status: " + displayGridValue(pkg.ActionRequired),
		"Last Updated: " + displayGridValue(formatRelativeTime(pkg.UpdatedAt)),
		"Description: " + displayGridValue(pkg.Description),
		"Homepage: " + displayGridValue(m.infoDetails.Homepage),
		"Repository: " + displayGridValue(m.infoDetails.Repository),
		"Location: " + displayGridValue(m.infoDetails.Location),
		"Size: " + displayGridValue(m.infoDetails.Size),
		"Dependencies: " + displayGridValue(strings.Join(m.infoDetails.Dependencies, ", ")),
		"Dependents: " + displayGridValue(strings.Join(m.infoDetails.Dependents, ", ")),
		"Uninstall: " + uninstallCommand(*pkg),
	}
	if m.infoLoading {
		lines = append(lines, "Loading: live metadata")
	}
	if m.infoError != "" {
		lines = append(lines, "Error: "+m.infoError)
	}
	width := min(max(48, m.viewWidth()/2), max(48, m.viewWidth()-4))
	return m.panelStyle().
		Padding(0, 1).
		Width(width).
		MaxWidth(width).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderInfoOverlay(base string) string {
	modal := m.renderInfoModal()
	if modal == "" {
		return base
	}
	return strings.Join([]string{base, "", overlayCentered(m.viewWidth(), max(8, m.state.Height/3), modal)}, "\n")
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
		"Last Updated: " + displayGridValue(formatRelativeTime(pkg.UpdatedAt)),
		"Action: " + displayGridValue(pkg.ActionRequired),
		"Dependancy: " + displayGridValue(pkg.UsedBy),
		"Last Used: " + displayGridValue(pkg.LastUsed),
		"Description: " + displayGridValue(pkg.Description),
		"Uninstall: " + uninstallCommand(*pkg),
	}
	return strings.Join(lines, "\n")
}
