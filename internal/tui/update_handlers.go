package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) updateNonKey(msg tea.Msg) (bool, tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.Width = msg.Width
		m.state.Height = msg.Height
		m.help.Width = msg.Width
		m.table.SetWidth(max(20, msg.Width-4))
		m.table.SetHeight(max(8, msg.Height-8))
		m.resizeGrid()
		m.syncRows()
		return true, m, nil
	case refreshDoneMsg:
		m.state.Packages = msg.Packages
		m.state.Statuses = msg.Statuses
		m.state.IsLoading = false
		m.statusMessage = "refresh complete"
		m.syncRows()
		return true, m, nil
	case exportDoneMsg:
		m.exportMenu = false
		m.mode = ModeNormal
		m.statusMessage = "exported to " + msg.path
		return true, m, nil
	case exportFailedMsg:
		m.exportMenu = false
		m.mode = ModeNormal
		m.statusMessage = msg.err.Error()
		return true, m, nil
	case infoLoadedMsg:
		if msg.key != m.infoKey {
			return true, m, nil
		}
		m.infoLoading = false
		m.infoDetails = msg.details
		if msg.err != nil {
			m.infoError = msg.err.Error()
		} else {
			m.infoError = ""
		}
		return true, m, nil
	case tea.MouseMsg:
		updated, cmd := m.updateMouse(msg)
		return true, updated, cmd
	case loadingTickMsg:
		if !m.state.IsLoading {
			return true, m, nil
		}
		m.loadingFrame = (m.loadingFrame + 1) % len(loadingFrames)
		return true, m, loadingTickCmd()
	default:
		return false, m, nil
	}
}

func (m Model) updateKey(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.exportMenu {
		return m.updateExportMenu(keyMsg)
	}
	if m.infoOpen {
		return m.updateInfoModalKeys(keyMsg)
	}
	if m.popupKind != PopupNone {
		return m.updatePopupMenu(keyMsg)
	}
	if m.mode == ModeColumn {
		return m.updateColumnMode(keyMsg)
	}
	if m.filter.Focused() {
		return m.updateFilterKeys(keyMsg)
	}
	if handled, updated, cmd := m.updateNormalKeyBindings(keyMsg); handled {
		return updated, cmd
	}
	if shouldStartFilter(keyMsg) {
		return m.startFilterFromKey(keyMsg)
	}
	if m.layout == LayoutFull {
		return m.updateFullGridNavigation(keyMsg)
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(keyMsg)
	m.state.Selected = m.table.Cursor()
	return m, cmd
}

func (m Model) updateInfoModalKeys(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "i":
		m.closeInfoModal()
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updateFilterKeys(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m Model) updateNormalKeyBindings(keyMsg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return true, m, tea.Quit
	case "V":
		m.mode = ModeColumn
		m.columnFocus = 0
		return true, m, nil
	case "f":
		m.openPopup(PopupSource)
		return true, m, nil
	case "a":
		m.openPopup(PopupAction)
		return true, m, nil
	case "u":
		m.openPopup(PopupUpdated)
		return true, m, nil
	case "?":
		m.state.ShowHelp = !m.state.ShowHelp
		return true, m, nil
	case "s":
		m.state.CycleSort()
		m.syncRows()
		return true, m, nil
	case "t":
		m.theme = nextTheme(m.theme)
		m.statusMessage = "theme: " + string(m.theme)
		return true, m, nil
	case "r":
		m.state.IsLoading = true
		m.statusMessage = "refreshing " + m.loadingIndicator()
		return true, m, tea.Batch(refreshCmd(m.refresh), loadingTickCmd())
	case "e":
		if len(m.state.VisiblePackages()) == 0 {
			m.statusMessage = "nothing to export"
			return true, m, nil
		}
		m.exportMenu = true
		m.mode = ModeExport
		m.exportChoice = 0
		return true, m, nil
	case "d":
		updated, cmd := m.copyUninstallCommand()
		return true, updated, cmd
	case "i":
		updated, cmd := m.openInfoModal()
		return true, updated, cmd
	case "/":
		m.focusFilter()
		m.mode = ModeFilter
		return true, m, nil
	case "enter":
		m.detailOpen = !m.detailOpen
		return true, m, nil
	case "esc":
		m.state.Filter = ""
		m.filter.SetValue("")
		m.syncRows()
		return true, m, nil
	default:
		return false, m, nil
	}
}

func (m Model) startFilterFromKey(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.focusFilter()
	m.mode = ModeFilter
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(keyMsg)
	m.state.Filter = m.filter.Value()
	m.syncRows()
	return m, cmd
}

func (m Model) copyUninstallCommand() (tea.Model, tea.Cmd) {
	pkg := m.selectedPackage()
	if pkg == nil {
		m.statusMessage = "nothing selected"
		return m, nil
	}
	command := uninstallCommand(*pkg)
	if err := m.copyToClipboard(command); err != nil {
		m.statusMessage = err.Error()
		return m, nil
	}
	m.statusMessage = "copied uninstall command"
	return m, nil
}
