package tui

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
