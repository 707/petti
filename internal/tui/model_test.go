package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nad/pkgview/internal/app"
	"github.com/nad/pkgview/internal/collectors"
	"github.com/nad/pkgview/internal/model"
)

func TestNewAppliesInitialFilter(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "ruff", Source: model.SourcePip},
		},
		Filter: "gh",
	})

	if got := m.filter.Value(); got != "gh" {
		t.Fatalf("filter.Value() = %q, want %q", got, "gh")
	}
	if len(m.state.VisiblePackages()) != 1 {
		t.Fatalf("len(VisiblePackages) = %d, want 1", len(m.state.VisiblePackages()))
	}
	if len(m.keys.FullHelp()) != 2 {
		t.Fatalf("len(FullHelp()) = %d, want 2", len(m.keys.FullHelp()))
	}
	if m.layout != LayoutFull {
		t.Fatalf("layout = %q, want %q", m.layout, LayoutFull)
	}
}

func TestUpdateTypingStartsFilterMode(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "ruff", Source: model.SourcePip},
		},
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = updated.(Model)

	if !m.filter.Focused() {
		t.Fatal("filter should be focused after typing")
	}
	if got := m.filter.Value(); got != "x" {
		t.Fatalf("filter.Value() = %q, want %q", got, "x")
	}
}

func TestUpdateDoesNotToggleLayoutAtRuntime(t *testing.T) {
	m := New(Config{Layout: LayoutCompact})
	if m.layout != LayoutCompact {
		t.Fatalf("layout = %q, want %q", m.layout, LayoutCompact)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.layout != LayoutCompact {
		t.Fatalf("layout = %q, want %q", m.layout, LayoutCompact)
	}
}

func TestViewFullLayoutShowsModeAndFilterBoxes(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Version: "2.0.0", Source: model.SourceHomebrew}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)
	view := m.View()
	if !strings.Contains(view, "MODE: NORMAL") {
		t.Fatalf("View() = %q, want mode indicator", view)
	}
	if !strings.Contains(view, "Search") || !strings.Contains(view, "Source") || !strings.Contains(view, "Sort") {
		t.Fatalf("View() = %q, want filter boxes", view)
	}
	if !strings.Contains(view, "Managers") {
		t.Fatalf("View() = %q, want top status bar", view)
	}
	if !strings.Contains(view, "Action (a)") || !strings.Contains(view, "Updated (u)") {
		t.Fatalf("View() = %q, want expanded filter boxes", view)
	}
	if strings.Contains(view, "Selected Package") {
		t.Fatalf("View() = %q, detail pane should be collapsed by default", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if lipgloss.Width(line) > 100 {
			t.Fatalf("line width %d exceeds window width 100: %q", lipgloss.Width(line), line)
		}
	}
}

func TestRenderFullChromeHelpers(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		Statuses: []model.CollectorStatus{
			{Source: model.SourceHomebrew, Label: "homebrew", State: model.CollectorStateReady},
			{Source: model.SourceNPM, Label: "npm", State: model.CollectorStateMissing},
		},
	})
	m.state.Width = 100
	top := m.renderTopBar()
	if !strings.Contains(top, "pkgview") || !strings.Contains(top, "Managers") {
		t.Fatalf("renderTopBar() = %q", top)
	}

	filters := m.renderFilterStrip()
	if !strings.Contains(filters, "Search") || !strings.Contains(filters, "All Sources") {
		t.Fatalf("renderFilterStrip() = %q", filters)
	}

	helpBar := m.renderModeBar()
	if !strings.Contains(helpBar, "V:select") || !strings.Contains(helpBar, "/:search") || !strings.Contains(helpBar, "f:source") {
		t.Fatalf("renderModeBar() = %q", helpBar)
	}
}

func TestFullChromeHelperValues(t *testing.T) {
	m := New(Config{})

	if got := m.activeSourceLabel(); got != "All Sources" {
		t.Fatalf("activeSourceLabel() = %q", got)
	}
	m.state.SourceFilter = model.SourceHomebrew
	if got := m.activeSourceLabel(); got != "Homebrew" {
		t.Fatalf("activeSourceLabel() = %q", got)
	}
	m.state.SourceFilter = model.SourceHomebrewCask
	if got := m.activeSourceLabel(); got != "Casks" {
		t.Fatalf("activeSourceLabel() = %q", got)
	}
	m.state.SourceFilter = model.SourceNPM
	if got := m.activeSourceLabel(); got != "npm" {
		t.Fatalf("activeSourceLabel() = %q", got)
	}
	m.state.SourceFilter = model.SourcePip
	if got := m.activeSourceLabel(); got != "pip" {
		t.Fatalf("activeSourceLabel() = %q", got)
	}

	if got := m.activeSortLabel(); got != "Default" {
		t.Fatalf("activeSortLabel() = %q", got)
	}
	m.state.SortMode = app.SortName
	if got := m.activeSortLabel(); got != "Name" {
		t.Fatalf("activeSortLabel() = %q", got)
	}
	m.state.SortMode = app.SortVersion
	if got := m.activeSortLabel(); got != "Version" {
		t.Fatalf("activeSortLabel() = %q", got)
	}
	m.state.SortMode = app.SortSource
	if got := m.activeSortLabel(); got != "Source" {
		t.Fatalf("activeSortLabel() = %q", got)
	}

	if got := m.managerSummary(); got != "not checked" {
		t.Fatalf("managerSummary() = %q", got)
	}
	m.state.Statuses = []model.CollectorStatus{
		{Source: model.SourceHomebrew, Label: "brew", State: model.CollectorStateReady},
		{Source: model.SourceNPM, State: model.CollectorStateMissing},
	}
	summary := m.managerSummary()
	if !strings.Contains(summary, "brew:ready") || !strings.Contains(summary, "npm:missing") {
		t.Fatalf("managerSummary() = %q", summary)
	}

	if got := valueOrPlaceholder("", "placeholder"); got != "placeholder" {
		t.Fatalf("valueOrPlaceholder() = %q", got)
	}
	if got := valueOrPlaceholder("value", "placeholder"); got != "value" {
		t.Fatalf("valueOrPlaceholder() = %q", got)
	}
	if got := filterBoxWidth("A", "BBBB"); got < 18 {
		t.Fatalf("filterBoxWidth() = %d, want >= 18", got)
	}
}

func TestViewCompactLayoutUsesCompactRenderer(t *testing.T) {
	m := New(Config{
		Layout:   LayoutCompact,
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	view := m.View()
	if strings.Contains(view, "Selected Package") {
		t.Fatalf("compact view should not show detail pane: %q", view)
	}
	if !strings.Contains(view, "Filter:") {
		t.Fatalf("View() = %q, want filter line", view)
	}

	m.exportMenu = true
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 20})
	m = updated.(Model)
	view = m.View()
	if !strings.Contains(view, "Terminal too small") || !strings.Contains(view, "Export") {
		t.Fatalf("compact view = %q, want resize warning and export menu", view)
	}
}

func TestUpdateEnterTogglesDetailPane(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	if m.detailOpen {
		t.Fatal("detailOpen should default to false in full layout")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.detailOpen {
		t.Fatal("detailOpen should toggle on")
	}
}

func TestUpdateColumnModeAppliesActions(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "ruff", Source: model.SourcePip},
		},
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m = updated.(Model)
	if m.mode != ModeColumn {
		t.Fatalf("mode = %q, want %q", m.mode, ModeColumn)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.popupKind != PopupSource {
		t.Fatalf("popupKind = %q, want %q", m.popupKind, PopupSource)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.state.SourceFilter != model.SourceHomebrew {
		t.Fatalf("SourceFilter = %q, want %q", m.state.SourceFilter, model.SourceHomebrew)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.mode != ModeFilter || !m.filter.Focused() {
		t.Fatalf("mode/focus = %q/%v, want FILTER/true", m.mode, m.filter.Focused())
	}
}

func TestUpdateColumnModeOtherBranches(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	m.mode = ModeColumn
	m.columnFocus = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(Model)
	if m.columnFocus != 4 {
		t.Fatalf("columnFocus = %d, want 4", m.columnFocus)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.mode != ModeNormal {
		t.Fatalf("mode = %q, want %q", m.mode, ModeNormal)
	}

	m.mode = ModeColumn
	m.columnFocus = 2
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(Model)
	if m.columnFocus != 1 {
		t.Fatalf("columnFocus = %d, want 1", m.columnFocus)
	}

	m.mode = ModeColumn
	m.columnFocus = 1
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.state.SortMode != app.SortVersion {
		t.Fatalf("SortMode = %d, want %d", m.state.SortMode, app.SortVersion)
	}

	m.mode = ModeColumn
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("unexpected cmd in default column-mode branch")
	}

	m.mode = ModeColumn
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("quit command should not be nil")
	}
}

func TestUpdateTogglesHelpAndCyclesSort(t *testing.T) {
	m := New(Config{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(Model)
	if !m.state.ShowHelp {
		t.Fatal("ShowHelp = false, want true")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if m.state.SortMode != app.SortName {
		t.Fatalf("SortMode = %d, want %d", m.state.SortMode, app.SortName)
	}
}

func TestUpdateRefreshesData(t *testing.T) {
	m := New(Config{
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{
				Packages: []model.Package{{Name: "new", Source: model.SourceNPM}},
				Statuses: []model.CollectorStatus{{Source: model.SourceNPM, State: model.CollectorStateReady}},
			}
		},
	})

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = updated.(Model)
	if !m.state.IsLoading {
		t.Fatal("IsLoading = false, want true")
	}

	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	if m.state.IsLoading {
		t.Fatal("IsLoading = true, want false")
	}
	if got := m.state.Packages[0].Name; got != "new" {
		t.Fatalf("Packages[0].Name = %q, want %q", got, "new")
	}
}

func TestUpdateExportsVisiblePackages(t *testing.T) {
	var (
		gotFormat ExportFormat
		gotNames  []string
	)
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		Export: func(format ExportFormat, packages []model.Package) (string, error) {
			gotFormat = format
			for _, pkg := range packages {
				gotNames = append(gotNames, pkg.Name)
			}
			return "pkgview-export.txt", nil
		},
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)
	if !m.exportMenu {
		t.Fatal("exportMenu = false, want true")
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)

	if gotFormat != ExportTXT {
		t.Fatalf("format = %q, want %q", gotFormat, ExportTXT)
	}
	if len(gotNames) != 1 || gotNames[0] != "gh" {
		t.Fatalf("exported names = %#v, want [gh]", gotNames)
	}
	if !strings.Contains(m.statusMessage, "pkgview-export.txt") {
		t.Fatalf("statusMessage = %q, want export path", m.statusMessage)
	}
}

func TestNewProvidesDefaultRefreshAndExport(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		Statuses: []model.CollectorStatus{{Source: model.SourceHomebrew, State: model.CollectorStateReady}},
	})
	t.Cleanup(func() {
		_ = os.Remove("pkgview-export.txt")
	})
	result := m.refresh(context.Background())
	if len(result.Packages) != 1 || result.Packages[0].Name != "gh" {
		t.Fatalf("refresh packages = %#v", result.Packages)
	}
	if _, err := m.export(ExportTXT, []model.Package{{Name: "gh", Source: model.SourceHomebrew}}); err != nil {
		t.Fatalf("default export error = %v", err)
	}
}

func TestUpdateHandlesExportFailure(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		Export: func(ExportFormat, []model.Package) (string, error) {
			return "", errors.New("export failed")
		},
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	updated, _ = m.Update(cmd())
	m = updated.(Model)

	if m.exportMenu {
		t.Fatal("exportMenu = true, want false")
	}
	if m.statusMessage != "export failed" {
		t.Fatalf("statusMessage = %q, want %q", m.statusMessage, "export failed")
	}
}

func TestUpdateWindowSizeAndInit(t *testing.T) {
	m := New(Config{})
	if cmd := m.Init(); cmd != nil {
		t.Fatal("Init() should return nil")
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)
	if m.state.Width != 100 || m.state.Height != 30 {
		t.Fatalf("size = %dx%d, want 100x30", m.state.Width, m.state.Height)
	}
	if m.help.Width != 100 {
		t.Fatalf("help.Width = %d, want 100", m.help.Width)
	}
}

func TestUpdateFilterEscAndEnter(t *testing.T) {
	m := New(Config{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	if !m.filter.Focused() {
		t.Fatal("filter should be focused")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.filter.Focused() {
		t.Fatal("filter should blur on enter")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.filter.Value() != "" || m.state.Filter != "" {
		t.Fatalf("filter/state = %q/%q, want empty", m.filter.Value(), m.state.Filter)
	}
}

func TestUpdateEscClearsFilterOutsideFilterMode(t *testing.T) {
	m := New(Config{Filter: "gh"})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.state.Filter != "" || m.filter.Value() != "" {
		t.Fatalf("filter/state = %q/%q, want empty", m.filter.Value(), m.state.Filter)
	}
}

func TestUpdateQuitAndNonKeyMessage(t *testing.T) {
	m := New(Config{})
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(Model)
	if m.state.ShowHelp {
		t.Fatal("unexpected state change")
	}
	if cmd == nil {
		t.Fatal("quit command should not be nil")
	}

	updated, cmd = m.Update(struct{}{})
	_ = updated.(Model)
	if cmd != nil {
		t.Fatal("non-key message should return nil cmd")
	}
}

func TestUpdateHandlesEmptyExportAndTableMovement(t *testing.T) {
	m := New(Config{})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)
	if m.statusMessage != "nothing to export" {
		t.Fatalf("statusMessage = %q, want %q", m.statusMessage, "nothing to export")
	}

	m = New(Config{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "ruff", Source: model.SourcePip},
		},
	})
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.state.Selected != 1 {
		t.Fatalf("Selected = %d, want 1", m.state.Selected)
	}
}

func TestViewShowsExportMenuAndStatus(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	m.exportMenu = true
	m.state.IsLoading = true
	m.statusMessage = "done"

	view := m.View()
	if !strings.Contains(view, "Export") {
		t.Fatalf("View() = %q, want export menu", view)
	}
	if !strings.Contains(view, "refreshing...") {
		t.Fatalf("View() = %q, want loading footer", view)
	}
	if !strings.Contains(view, "done") {
		t.Fatalf("View() = %q, want status message", view)
	}
}

func TestViewShowsResizeWarning(t *testing.T) {
	m := New(Config{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 20})
	m = updated.(Model)

	view := m.View()
	if !strings.Contains(view, "Terminal too small") {
		t.Fatalf("View() = %q, want resize warning", view)
	}
}

func TestUpdateExportMenuNavigation(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	m.exportMenu = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.exportChoice != 1 {
		t.Fatalf("exportChoice = %d, want 1", m.exportChoice)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.exportChoice != 0 {
		t.Fatalf("exportChoice = %d, want 0", m.exportChoice)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.exportMenu {
		t.Fatal("exportMenu = true, want false")
	}
}

func TestUpdateExportMenuQuitAndDefault(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	m.exportMenu = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("quit command should not be nil")
	}

	m.exportMenu = true
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("unexpected command for default export-menu branch")
	}
	if !m.exportMenu {
		t.Fatal("exportMenu should remain open on unrelated key")
	}
}

func TestHelpersAndDefaultExport(t *testing.T) {
	if selectedFormat(0) != ExportTXT {
		t.Fatal("selectedFormat(0) should be txt")
	}
	if selectedFormat(1) != ExportJSON {
		t.Fatal("selectedFormat(1) should be json")
	}
	if !shouldStartFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}) {
		t.Fatal("x should start filter")
	}
	if shouldStartFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}) {
		t.Fatal("r should not start filter")
	}
	if shouldStartFilter(tea.KeyMsg{Type: tea.KeyUp}) {
		t.Fatal("non-rune key should not start filter")
	}
	if max(2, 1) != 2 || max(1, 2) != 2 {
		t.Fatal("max returned wrong value")
	}

	txtPath, err := defaultExportFunc(ExportTXT, []model.Package{{Name: "gh", Source: model.SourceHomebrew}})
	if err != nil {
		t.Fatalf("defaultExportFunc(txt) error = %v", err)
	}
	jsonPath, err := defaultExportFunc(ExportJSON, []model.Package{{Name: "gh", Source: model.SourceHomebrew}})
	if err != nil {
		t.Fatalf("defaultExportFunc(json) error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(txtPath)
		_ = os.Remove(jsonPath)
	})
	if !strings.HasSuffix(txtPath, ".txt") || !strings.HasSuffix(jsonPath, ".json") {
		t.Fatalf("paths = %q, %q", txtPath, jsonPath)
	}
}

func TestSelectedPackageAndUninstallCommand(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	m.state.Selected = 9
	selected := m.selectedPackage()
	if selected == nil || selected.Name != "gh" {
		t.Fatalf("selectedPackage() = %#v, want gh", selected)
	}

	empty := New(Config{})
	if empty.selectedPackage() != nil {
		t.Fatal("selectedPackage() should be nil when empty")
	}

	cases := []struct {
		source model.Source
		want   string
	}{
		{model.SourceHomebrew, "brew uninstall gh"},
		{model.SourceHomebrewCask, "brew uninstall --cask gh"},
		{model.SourceNPM, "npm uninstall -g gh"},
		{model.SourcePip, "pip uninstall gh"},
	}
	for _, tc := range cases {
		if got := uninstallCommand(model.Package{Name: "gh", Source: tc.source}); got != tc.want {
			t.Fatalf("uninstallCommand(%q) = %q, want %q", tc.source, got, tc.want)
		}
	}
}

func TestRenderFullBodyWithoutDetailPaneAndSizingHelpers(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	m.detailOpen = false
	m.state.Width = 90
	body := m.renderFullBody()
	if strings.Contains(body, "Selected Package") {
		t.Fatalf("renderFullBody() should omit detail pane when closed: %q", body)
	}

	if min(1, 2) != 1 {
		t.Fatalf("min(1, 2) = %d, want 1", min(1, 2))
	}
	if fullDetailWidth(200) != 32 {
		t.Fatalf("fullDetailWidth(200) = %d, want 32", fullDetailWidth(200))
	}
}

func TestFullLayoutUsesViewportBackedGrid(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Version: "2.0.0", Source: model.SourceHomebrew},
			{Name: "ruff", Version: "0.5.0", Source: model.SourcePip},
		},
	})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)

	if m.grid.Width == 0 || m.grid.Height == 0 {
		t.Fatalf("grid size = %dx%d, want non-zero", m.grid.Width, m.grid.Height)
	}
	if m.table.Cursor() != 0 {
		t.Fatalf("compact table cursor changed unexpectedly: %d", m.table.Cursor())
	}

	body := m.renderGridPane()
	if !strings.Contains(body, "PKG") || !strings.Contains(body, "VER") || !strings.Contains(body, "SRC") {
		t.Fatalf("renderGridPane() = %q, want custom header", body)
	}
	if !strings.Contains(body, "gh") || !strings.Contains(body, "ruff") {
		t.Fatalf("renderGridPane() = %q, want rendered package rows", body)
	}
}

func TestUpdateFullLayoutMovesSelectionWithCustomGrid(t *testing.T) {
	packages := make([]model.Package, 0, 20)
	for i := 0; i < 20; i++ {
		packages = append(packages, model.Package{
			Name:    fmt.Sprintf("pkg-%02d", i),
			Version: "1.0.0",
			Source:  model.SourceHomebrew,
		})
	}

	m := New(Config{Packages: packages})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 18})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.state.Selected != 1 {
		t.Fatalf("Selected = %d, want 1", m.state.Selected)
	}
	if m.table.Cursor() != 0 {
		t.Fatalf("table cursor = %d, want compact table untouched in full layout", m.table.Cursor())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = updated.(Model)
	if m.state.Selected != len(packages)-1 {
		t.Fatalf("Selected = %d, want %d", m.state.Selected, len(packages)-1)
	}
	if m.grid.YOffset == 0 {
		t.Fatal("grid should scroll when selecting the last row")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = updated.(Model)
	if m.state.Selected != 0 {
		t.Fatalf("Selected = %d, want 0", m.state.Selected)
	}
	if m.grid.YOffset != 0 {
		t.Fatalf("grid.YOffset = %d, want 0", m.grid.YOffset)
	}
}

func TestGridHelpersCoverScrollHeaderAndEmptyCases(t *testing.T) {
	m := New(Config{})
	m.grid.Height = 0
	m.state.Selected = 3
	m.grid.YOffset = 2
	m.syncGridOffset()
	if m.grid.YOffset != 2 {
		t.Fatalf("grid.YOffset = %d, want unchanged 2 when height is zero", m.grid.YOffset)
	}

	m = New(Config{
		Packages: []model.Package{
			{Name: "alpha", Version: "1.0.0", Source: model.SourceHomebrew},
			{Name: "beta", Version: "2.0.0", Source: model.SourcePip},
			{Name: "gamma", Version: "3.0.0", Source: model.SourceNPM},
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 18})
	m = updated.(Model)
	m.mode = ModeColumn
	m.columnFocus = 1

	header := m.renderGridHeader()
	if !strings.Contains(header, "VER") || !strings.Contains(header, "UPDATED") || !strings.Contains(header, "ACTION") || !strings.Contains(header, "DESC") {
		t.Fatalf("renderGridHeader() = %q, want expanded columns", header)
	}
	if !strings.Contains(header, "|") || !strings.Contains(header, "+") {
		t.Fatalf("renderGridHeader() = %q, want ascii header gridlines", header)
	}

	m.state.Selected = 0
	m.grid.YOffset = 2
	m.syncGridOffset()
	if m.grid.YOffset != 0 {
		t.Fatalf("grid.YOffset = %d, want 0 after scrolling selected row into view", m.grid.YOffset)
	}

	updated, _ = m.updateFullGridNavigation(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(Model)
	if m.state.Selected == 0 {
		t.Fatal("page down should move the selection")
	}

	updated, _ = m.updateFullGridNavigation(tea.KeyMsg{Type: tea.KeyPgUp})
	m = updated.(Model)
	if m.state.Selected != 0 {
		t.Fatalf("Selected = %d, want 0 after paging back up", m.state.Selected)
	}

	updated, _ = m.updateFullGridNavigation(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.state.Selected != 0 {
		t.Fatalf("Selected = %d, want clamped 0 after moving above start", m.state.Selected)
	}

	updated, cmd := m.updateFullGridNavigation(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("default full-grid navigation branch should not return a command")
	}

	empty := New(Config{})
	empty.grid.Width = 40
	if got := empty.renderGridRows(); !strings.Contains(got, "No packages match the current view") {
		t.Fatalf("renderGridRows() = %q, want empty-state message", got)
	}

	empty.moveSelection(1)
	if empty.state.Selected != 0 {
		t.Fatalf("Selected = %d, want 0 for empty grid", empty.state.Selected)
	}
}

func TestUpdateDirectSourceShortcutAndModeHelp(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "ruff", Source: model.SourcePip},
		},
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = updated.(Model)
	if m.popupKind != PopupSource {
		t.Fatalf("popupKind = %q, want %q", m.popupKind, PopupSource)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.state.SourceFilter != model.SourceHomebrew {
		t.Fatalf("SourceFilter = %q, want %q", m.state.SourceFilter, model.SourceHomebrew)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)
	if m.mode != ModeExport {
		t.Fatalf("mode = %q, want %q", m.mode, ModeExport)
	}
	helpBar := m.renderModeBar()
	if !strings.Contains(helpBar, "Esc:back") || !strings.Contains(helpBar, "Enter:write") {
		t.Fatalf("renderModeBar() = %q, want export mode hints", helpBar)
	}
}

func TestModeHelpPartsAndTrimHelpers(t *testing.T) {
	m := New(Config{})

	m.mode = ModeFilter
	if got := strings.Join(m.modeHelpParts(), " "); !strings.Contains(got, "Enter:apply") {
		t.Fatalf("modeHelpParts(filter) = %q", got)
	}

	m.mode = ModeColumn
	if got := strings.Join(m.modeHelpParts(), " "); !strings.Contains(got, "h/l:move") {
		t.Fatalf("modeHelpParts(column) = %q", got)
	}

	m.mode = ModeNormal
	if got := strings.Join(m.modeHelpParts(), " "); !strings.Contains(got, "f:source") {
		t.Fatalf("modeHelpParts(normal) = %q", got)
	}

	if got := trimToWidth("value", 0); got != "" {
		t.Fatalf("trimToWidth(width0) = %q, want empty", got)
	}
	if got := trimToWidth("ok", 4); got != "ok" {
		t.Fatalf("trimToWidth(short) = %q, want %q", got, "ok")
	}
	if got := trimToWidth("value", 3); got != "val" {
		t.Fatalf("trimToWidth(width3) = %q, want %q", got, "val")
	}
}

func TestPopupHelpersAndLabels(t *testing.T) {
	m := New(Config{})

	m.state.SortMode = app.SortUpdated
	if got := m.activeSortLabel(); got != "Updated" {
		t.Fatalf("activeSortLabel() = %q", got)
	}

	m.state.ActionFilter = app.ActionFilterUpdate
	if got := m.activeActionLabel(); got != "Update" {
		t.Fatalf("activeActionLabel() = %q", got)
	}
	m.state.ActionFilter = app.ActionFilterAttention
	if got := m.activeActionLabel(); got != "Attention" {
		t.Fatalf("activeActionLabel() = %q", got)
	}
	m.state.ActionFilter = app.ActionFilterCurrent
	if got := m.activeActionLabel(); got != "Current" {
		t.Fatalf("activeActionLabel() = %q", got)
	}
	m.state.ActionFilter = app.ActionFilterUnknown
	if got := m.activeActionLabel(); got != "Unknown" {
		t.Fatalf("activeActionLabel() = %q", got)
	}

	m.state.UpdatedFilter = app.UpdatedFilterKnown
	if got := m.activeUpdatedLabel(); got != "Known" {
		t.Fatalf("activeUpdatedLabel() = %q", got)
	}
	m.state.UpdatedFilter = app.UpdatedFilterUnknown
	if got := m.activeUpdatedLabel(); got != "Unknown" {
		t.Fatalf("activeUpdatedLabel() = %q", got)
	}

	m.state.SourceFilter = model.SourceNPM
	if got := m.currentPopupChoice(PopupSource); got != 3 {
		t.Fatalf("currentPopupChoice(source) = %d", got)
	}
	m.state.SourceFilter = model.SourceHomebrew
	if got := m.currentPopupChoice(PopupSource); got != 1 {
		t.Fatalf("currentPopupChoice(source) = %d", got)
	}
	m.state.SourceFilter = model.SourceHomebrewCask
	if got := m.currentPopupChoice(PopupSource); got != 2 {
		t.Fatalf("currentPopupChoice(source) = %d", got)
	}
	m.state.SourceFilter = model.SourcePip
	if got := m.currentPopupChoice(PopupSource); got != 4 {
		t.Fatalf("currentPopupChoice(source) = %d", got)
	}
	m.state.SourceFilter = ""
	if got := m.currentPopupChoice(PopupSource); got != 0 {
		t.Fatalf("currentPopupChoice(source all) = %d", got)
	}
	m.state.ActionFilter = app.ActionFilterCurrent
	if got := m.currentPopupChoice(PopupAction); got != 3 {
		t.Fatalf("currentPopupChoice(action) = %d", got)
	}
	m.state.ActionFilter = app.ActionFilterUpdate
	if got := m.currentPopupChoice(PopupAction); got != 1 {
		t.Fatalf("currentPopupChoice(action) = %d", got)
	}
	m.state.ActionFilter = app.ActionFilterAttention
	if got := m.currentPopupChoice(PopupAction); got != 2 {
		t.Fatalf("currentPopupChoice(action) = %d", got)
	}
	m.state.ActionFilter = app.ActionFilterUnknown
	if got := m.currentPopupChoice(PopupAction); got != 4 {
		t.Fatalf("currentPopupChoice(action unknown) = %d", got)
	}
	m.state.ActionFilter = app.ActionFilterAll
	if got := m.currentPopupChoice(PopupAction); got != 0 {
		t.Fatalf("currentPopupChoice(action all) = %d", got)
	}
	m.state.UpdatedFilter = app.UpdatedFilterUnknown
	if got := m.currentPopupChoice(PopupUpdated); got != 2 {
		t.Fatalf("currentPopupChoice(updated) = %d", got)
	}
	m.state.UpdatedFilter = app.UpdatedFilterKnown
	if got := m.currentPopupChoice(PopupUpdated); got != 1 {
		t.Fatalf("currentPopupChoice(updated) = %d", got)
	}
	m.state.UpdatedFilter = app.UpdatedFilterAll
	if got := m.currentPopupChoice(PopupUpdated); got != 0 {
		t.Fatalf("currentPopupChoice(updated all) = %d", got)
	}
	if got := m.currentPopupChoice(PopupKind("OTHER")); got != 0 {
		t.Fatalf("currentPopupChoice(other) = %d", got)
	}

	if len(m.popupOptions()) != 0 {
		t.Fatal("popupOptions() should be empty without popup kind")
	}
	m.popupKind = PopupSource
	if got := len(m.popupOptions()); got != 5 {
		t.Fatalf("popupOptions(source) = %d", got)
	}
	m.popupKind = PopupAction
	if got := len(m.popupOptions()); got != 5 {
		t.Fatalf("popupOptions(action) = %d", got)
	}
	m.popupKind = PopupUpdated
	if got := len(m.popupOptions()); got != 3 {
		t.Fatalf("popupOptions(updated) = %d", got)
	}

	m.openPopup(PopupSource)
	if m.popupKind != PopupSource {
		t.Fatalf("popupKind = %q, want %q", m.popupKind, PopupSource)
	}

	m.popupKind = PopupAction
	m.popupChoice = 2
	m.applyPopupChoice()
	if m.state.ActionFilter != app.ActionFilterAttention {
		t.Fatalf("ActionFilter = %q, want attention", m.state.ActionFilter)
	}
	m.popupChoice = 1
	m.applyPopupChoice()
	if m.state.ActionFilter != app.ActionFilterUpdate {
		t.Fatalf("ActionFilter = %q, want update", m.state.ActionFilter)
	}
	m.popupChoice = 3
	m.applyPopupChoice()
	if m.state.ActionFilter != app.ActionFilterCurrent {
		t.Fatalf("ActionFilter = %q, want current", m.state.ActionFilter)
	}
	m.popupChoice = 4
	m.applyPopupChoice()
	if m.state.ActionFilter != app.ActionFilterUnknown {
		t.Fatalf("ActionFilter = %q, want unknown", m.state.ActionFilter)
	}
	m.popupChoice = 0
	m.applyPopupChoice()
	if m.state.ActionFilter != app.ActionFilterAll {
		t.Fatalf("ActionFilter = %q, want all", m.state.ActionFilter)
	}

	m.popupKind = PopupSource
	m.popupChoice = 1
	m.applyPopupChoice()
	if m.state.SourceFilter != model.SourceHomebrew {
		t.Fatalf("SourceFilter = %q, want homebrew", m.state.SourceFilter)
	}
	m.popupChoice = 2
	m.applyPopupChoice()
	if m.state.SourceFilter != model.SourceHomebrewCask {
		t.Fatalf("SourceFilter = %q, want cask", m.state.SourceFilter)
	}
	m.popupChoice = 3
	m.applyPopupChoice()
	if m.state.SourceFilter != model.SourceNPM {
		t.Fatalf("SourceFilter = %q, want npm", m.state.SourceFilter)
	}
	m.popupChoice = 4
	m.applyPopupChoice()
	if m.state.SourceFilter != model.SourcePip {
		t.Fatalf("SourceFilter = %q, want pip", m.state.SourceFilter)
	}
	m.popupChoice = 0
	m.applyPopupChoice()
	if m.state.SourceFilter != "" {
		t.Fatalf("SourceFilter = %q, want all", m.state.SourceFilter)
	}

	m.popupKind = PopupUpdated
	m.popupChoice = 1
	m.applyPopupChoice()
	if m.state.UpdatedFilter != app.UpdatedFilterKnown {
		t.Fatalf("UpdatedFilter = %q, want known", m.state.UpdatedFilter)
	}
	m.popupChoice = 2
	m.applyPopupChoice()
	if m.state.UpdatedFilter != app.UpdatedFilterUnknown {
		t.Fatalf("UpdatedFilter = %q, want unknown", m.state.UpdatedFilter)
	}
	m.popupChoice = 0
	m.applyPopupChoice()
	if m.state.UpdatedFilter != app.UpdatedFilterAll {
		t.Fatalf("UpdatedFilter = %q, want all", m.state.UpdatedFilter)
	}

	m.popupKind = PopupKind("OTHER")
	m.state.SourceFilter = model.SourceHomebrew
	m.applyPopupChoice()
	if m.state.SourceFilter != model.SourceHomebrew {
		t.Fatalf("SourceFilter = %q, want unchanged homebrew", m.state.SourceFilter)
	}
}

func TestPopupMenuAndFullBodyBranches(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{
				Name:           "gh",
				Version:        "2.0.0",
				Source:         model.SourceHomebrew,
				Description:    "GitHub CLI",
				ActionRequired: "update",
				UpdatedAt:      "2024-01-01",
			},
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 32})
	m = updated.(Model)
	m.detailOpen = true
	m.popupKind = PopupAction
	m.popupChoice = 1

	view := m.View()
	if !strings.Contains(view, "Filter ACTION") || !strings.Contains(view, "Selected Package") {
		t.Fatalf("View() = %q, want popup and detail", view)
	}
	if !m.detailAsSide() {
		t.Fatal("detailAsSide() should be true for wide layouts")
	}
	if !strings.Contains(m.renderSourceTabs(), "All") {
		t.Fatalf("renderSourceTabs() = %q", m.renderSourceTabs())
	}

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	m = updated.(Model)
	m.detailOpen = true
	body := m.renderFullBody()
	if !strings.Contains(body, "Selected Package") {
		t.Fatalf("renderFullBody() = %q, want bottom detail pane", body)
	}
	if m.detailAsSide() {
		t.Fatal("detailAsSide() should be false for narrower layouts")
	}

	empty := New(Config{})
	if got := empty.renderDetailPane(); !strings.Contains(got, "No package selected") {
		t.Fatalf("renderDetailPane() = %q", got)
	}
}

func TestPopupMenuUpdatesAndGridHelpers(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	m = updated.(Model)

	m.popupKind = PopupSource
	updated, _ = m.updatePopupMenu(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.popupChoice != 1 {
		t.Fatalf("popupChoice = %d, want 1", m.popupChoice)
	}
	updated, _ = m.updatePopupMenu(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.popupChoice != 0 {
		t.Fatalf("popupChoice = %d, want 0", m.popupChoice)
	}
	updated, _ = m.updatePopupMenu(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.popupChoice != len(m.popupOptions())-1 {
		t.Fatalf("popupChoice = %d, want wrap to %d", m.popupChoice, len(m.popupOptions())-1)
	}
	updated, _ = m.updatePopupMenu(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.popupKind != PopupNone {
		t.Fatalf("popupKind = %q, want none", m.popupKind)
	}

	m.popupKind = PopupUpdated
	updated, cmd := m.updatePopupMenu(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("unexpected command for popup default branch")
	}
	updated, cmd = m.updatePopupMenu(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("quit command should not be nil")
	}

	m.popupKind = PopupUpdated
	m.popupChoice = 1
	updated, _ = m.updatePopupMenu(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.state.UpdatedFilter != app.UpdatedFilterKnown || m.popupKind != PopupNone {
		t.Fatalf("updated filter/popup = %q/%q", m.state.UpdatedFilter, m.popupKind)
	}

	columns := m.fullGridColumns()
	if !hasColumn(columns, "desc") || !hasColumn(columns, "used") {
		t.Fatalf("fullGridColumns() = %#v, want desc and used", columns)
	}
	if got := gridRowValues(model.Package{Name: "gh", Version: "1.0.0", Source: model.SourceHomebrew}, columns); len(got) != len(columns) {
		t.Fatalf("gridRowValues() len = %d, want %d", len(got), len(columns))
	}
	if got := renderGridBorder(columns); !strings.Contains(got, "+") {
		t.Fatalf("renderGridBorder() = %q", got)
	}
	if got := renderGridASCII(columns[:2], []string{"gh", "1.0.0"}); !strings.Contains(got, "|") {
		t.Fatalf("renderGridASCII() = %q", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(Model)
	if m.popupKind != PopupAction {
		t.Fatalf("popupKind = %q, want %q", m.popupKind, PopupAction)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = updated.(Model)
	if m.popupKind != PopupUpdated {
		t.Fatalf("popupKind = %q, want %q", m.popupKind, PopupUpdated)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	compact := New(Config{})
	compact.grid.Width = 70
	if hasColumn(compact.fullGridColumns(), "desc") || hasColumn(compact.fullGridColumns(), "used") {
		t.Fatalf("fullGridColumns() = %#v, want compact columns only", compact.fullGridColumns())
	}
	medium := New(Config{})
	medium.grid.Width = 95
	mediumCols := medium.fullGridColumns()
	if !hasColumn(mediumCols, "desc") || hasColumn(mediumCols, "used") {
		t.Fatalf("fullGridColumns() = %#v, want desc without used", mediumCols)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m = updated.(Model)
	m.columnFocus = 3
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.state.SortMode != app.SortUpdated {
		t.Fatalf("SortMode = %d, want %d", m.state.SortMode, app.SortUpdated)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m = updated.(Model)
	m.columnFocus = 4
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.popupKind != PopupAction {
		t.Fatalf("popupKind = %q, want %q", m.popupKind, PopupAction)
	}

	m.popupKind = PopupNone
	updated, _ = m.updatePopupMenu(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.popupKind != PopupNone {
		t.Fatalf("popupKind = %q, want none", m.popupKind)
	}
}
