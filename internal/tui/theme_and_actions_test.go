package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/707/petti/internal/model"
)

func TestGridColumnsFollowNewOrderAndSizing(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{
				Name:           "very-long-package-name",
				Version:        "123.456.789",
				Source:         model.SourceHomebrew,
				Description:    "Long package description",
				UpdatedAt:      "2026-03-01",
				ActionRequired: "update required soon",
				UsedBy:         "no",
			},
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = updated.(Model)

	columns := m.fullGridColumns()
	gotOrder := make([]string, 0, len(columns))
	widths := map[string]int{}
	for _, col := range columns {
		gotOrder = append(gotOrder, col.key)
		widths[col.key] = col.width
	}

	wantOrder := []string{"pkg", "ver", "src", "desc", "updated", "status", "usedby"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("column order = %v, want %v", gotOrder, wantOrder)
	}
	if widths["updated"] != len("LAST UPDATED") {
		t.Fatalf("updated width = %d, want %d", widths["updated"], len("LAST UPDATED"))
	}
	if widths["ver"] != 5 {
		t.Fatalf("ver width = %d, want 5 for half-natural version width", widths["ver"])
	}
	if widths["status"] < len("STATUS")+4 {
		t.Fatalf("status width = %d, want protected width", widths["status"])
	}
	if widths["usedby"] < 6 {
		t.Fatalf("usedby width = %d, want dependency column width", widths["usedby"])
	}
}

func TestRenderGridHeaderUsesTwoLineLastUpdatedHeader(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew, UsedBy: "no"}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = updated.(Model)

	header := m.renderGridHeader()
	if !strings.Contains(header, "DEPEN") || !strings.Contains(header, "DANCY") {
		t.Fatalf("renderGridHeader() = %q, want dependency header", header)
	}
	if !strings.Contains(header, "DESCRIPTION") || !strings.Contains(header, "LAST UPDATED") {
		t.Fatalf("renderGridHeader() = %q, want expanded header labels", header)
	}
}

func TestThemeSelectionChangesRenderedChrome(t *testing.T) {
	defaultModel := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		Theme:    ThemeDefault,
	})
	emberModel := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		Theme:    ThemeEmber,
	})

	defaultPalette := defaultModel.palette()
	emberPalette := emberModel.palette()
	if defaultPalette.topBackground == emberPalette.topBackground {
		t.Fatal("top background should differ across themes")
	}
	if defaultModel.ThemeName() != ThemeDefault {
		t.Fatalf("ThemeName() = %q, want %q", defaultModel.ThemeName(), ThemeDefault)
	}
	if emberModel.ThemeName() != ThemeEmber {
		t.Fatalf("ThemeName() = %q, want %q", emberModel.ThemeName(), ThemeEmber)
	}
	if defaultModel.palette().headerBackground == emberModel.palette().headerBackground {
		t.Fatal("grid header background should differ across themes")
	}
	if !strings.Contains(defaultModel.renderGridHeader(), "PKG") {
		t.Fatal("renderGridHeader() should still render header titles")
	}
	if defaultModel.palette().headerBackground != lipgloss.Color("57") {
		t.Fatalf("default header background = %q, want purple shade", defaultModel.palette().headerBackground)
	}
}

func TestThemeHelpersCoverValidationAndFallback(t *testing.T) {
	gotThemes := ValidThemes()
	if len(gotThemes) != len(validThemes()) {
		t.Fatalf("ValidThemes() len = %d, want %d", len(gotThemes), len(validThemes()))
	}
	if gotThemes[0] != ThemeDefault || gotThemes[len(gotThemes)-1] != ThemeFrostLight {
		t.Fatalf("ValidThemes() = %#v, want exported theme list in declaration order", gotThemes)
	}

	if !IsValidTheme(ThemeDefaultDark) || !IsValidTheme(ThemeFrostLight) {
		t.Fatal("expected dark and light theme variants to be valid themes")
	}
	if IsValidTheme(ThemeName("nope")) {
		t.Fatal("unexpected valid result for unknown theme")
	}

	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		Theme:    ThemeName("nope"),
	})
	if m.ThemeName() != ThemeDefault {
		t.Fatalf("ThemeName() = %q, want default fallback", m.ThemeName())
	}
	if paletteForTheme(ThemeFrostDark).topBackground == paletteForTheme(ThemeDefaultDark).topBackground {
		t.Fatal("frost palette should differ from default")
	}
	if nextTheme(ThemeDefaultDark) != ThemeDefaultLight {
		t.Fatalf("nextTheme(default dark) = %q, want %q", nextTheme(ThemeDefaultDark), ThemeDefaultLight)
	}
	if nextTheme(ThemeDefaultLight) != ThemeEmberDark {
		t.Fatalf("nextTheme(default light) = %q, want %q", nextTheme(ThemeDefaultLight), ThemeEmberDark)
	}
	if nextTheme(ThemeFrostLight) != ThemeDefaultDark {
		t.Fatalf("nextTheme(frost light) = %q, want %q", nextTheme(ThemeFrostLight), ThemeDefaultDark)
	}
	if nextTheme(ThemeName("nope")) != ThemeDefaultDark {
		t.Fatalf("nextTheme(unknown) = %q, want %q", nextTheme(ThemeName("nope")), ThemeDefaultDark)
	}
	if normalizeTheme(ThemeDefault) != ThemeDefaultDark {
		t.Fatalf("normalizeTheme(default) = %q, want %q", normalizeTheme(ThemeDefault), ThemeDefaultDark)
	}
	if normalizeTheme(ThemeEmber) != ThemeEmberDark {
		t.Fatalf("normalizeTheme(ember) = %q, want %q", normalizeTheme(ThemeEmber), ThemeEmberDark)
	}
	if normalizeTheme(ThemeFrost) != ThemeFrostDark {
		t.Fatalf("normalizeTheme(frost) = %q, want %q", normalizeTheme(ThemeFrost), ThemeFrostDark)
	}
	if normalizeTheme(ThemeDefaultLight) != ThemeDefaultLight {
		t.Fatalf("normalizeTheme(default-light) = %q, want unchanged", normalizeTheme(ThemeDefaultLight))
	}
	if paletteForTheme(ThemeDefaultLight).headerBackground == paletteForTheme(ThemeDefaultDark).headerBackground {
		t.Fatal("default light and dark header backgrounds should differ")
	}
	if paletteForTheme(ThemeEmberLight).headerBackground == paletteForTheme(ThemeEmberDark).headerBackground {
		t.Fatal("ember light and dark header backgrounds should differ")
	}
	if paletteForTheme(ThemeFrostLight).headerBackground == paletteForTheme(ThemeFrostDark).headerBackground {
		t.Fatal("frost light and dark header backgrounds should differ")
	}
}

func TestUpdateCyclesThemeAtRuntime(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "ruff", Source: model.SourcePip},
		},
		Theme: ThemeDefaultDark,
	})
	m.state.Selected = 1
	m.state.Filter = "ru"
	m.filter.SetValue("ru")
	m.grid.YOffset = 3
	m.popupKind = PopupNone

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = updated.(Model)

	if m.ThemeName() != ThemeDefaultLight {
		t.Fatalf("ThemeName() = %q, want %q", m.ThemeName(), ThemeDefaultLight)
	}
	if m.state.Selected != 1 {
		t.Fatalf("Selected = %d, want 1", m.state.Selected)
	}
	if m.filter.Value() != "ru" || m.state.Filter != "ru" {
		t.Fatalf("filter/state = %q/%q, want ru/ru", m.filter.Value(), m.state.Filter)
	}
	if m.grid.YOffset != 3 {
		t.Fatalf("grid.YOffset = %d, want 3", m.grid.YOffset)
	}
	if m.statusMessage != "theme: default-light" {
		t.Fatalf("statusMessage = %q, want %q", m.statusMessage, "theme: default-light")
	}
}

func TestUpdateCyclesThemeWrapsAndDoesNotStartFilter(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		Theme:    ThemeFrostLight,
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = updated.(Model)
	if m.ThemeName() != ThemeDefaultDark {
		t.Fatalf("ThemeName() = %q, want %q", m.ThemeName(), ThemeDefaultDark)
	}
	if m.filter.Focused() {
		t.Fatal("filter should not focus on theme toggle")
	}
	if m.statusMessage != "theme: default-dark" {
		t.Fatalf("statusMessage = %q, want %q", m.statusMessage, "theme: default-dark")
	}
}

func TestGridRowsUseSourceAndDependencyColors(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "brew-pkg", Version: "1.0.0", Source: model.SourceHomebrew, UsedBy: "Y"},
			{Name: "npm-pkg", Version: "2.0.0", Source: model.SourceNPM, UsedBy: "N"},
		},
		Theme: ThemeDefaultDark,
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = updated.(Model)

	rows := m.renderGridRows()
	if !strings.Contains(rows, "brew-pkg") || !strings.Contains(rows, "npm-pkg") {
		t.Fatalf("renderGridRows() = %q, want package rows", rows)
	}
	if m.sourceForeground(model.SourceHomebrew) == m.sourceForeground(model.SourceNPM) {
		t.Fatal("source colors should differ")
	}
	if m.sourceForeground(model.SourceHomebrewCask) == m.sourceForeground(model.SourcePip) {
		t.Fatal("cask and pip colors should differ")
	}
	if m.gridCellStyle("usedby", model.Package{UsedBy: "Y"}, false).GetForeground() != m.palette().dependencyYes {
		t.Fatal("Y dependency color should map to theme yellow")
	}
	if m.gridCellStyle("usedby", model.Package{UsedBy: "N"}, false).GetForeground() != m.palette().dependencyNo {
		t.Fatal("N dependency color should map to theme green")
	}
	if m.gridCellStyle("pkg", model.Package{Source: model.SourcePip}, false).GetForeground() != m.palette().pipForeground {
		t.Fatal("pkg color should map to source color")
	}
	if m.gridCellStyle("status", model.Package{Source: model.SourcePip}, false).GetForeground() != m.palette().mutedForeground {
		t.Fatal("non-colored columns should stay muted")
	}
	if m.sourceForeground(model.Source("other")) != m.palette().mutedForeground {
		t.Fatal("unknown sources should fall back to muted foreground")
	}
	if gridValueForColumn(model.Package{}, "other") != "" {
		t.Fatal("unknown grid column should return empty string")
	}
}

func TestThemeToggleAppearsInHelp(t *testing.T) {
	m := New(Config{})
	if got := strings.Join(m.modeHelpParts(), " "); !strings.Contains(got, "t:theme") {
		t.Fatalf("modeHelpParts() = %q, want t:theme", got)
	}
	if got := m.keys.ShortHelp(); len(got) != 9 {
		t.Fatalf("len(ShortHelp()) = %d, want 9", len(got))
	}
}

func TestUpdateCopyUninstallCommand(t *testing.T) {
	var copied string
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		CopyToClipboard: func(value string) error {
			copied = value
			return nil
		},
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(Model)

	if copied != "brew uninstall gh" {
		t.Fatalf("copied = %q, want brew uninstall command", copied)
	}
	if !strings.Contains(m.statusMessage, "copied") {
		t.Fatalf("statusMessage = %q, want copy confirmation", m.statusMessage)
	}
}

func TestUpdateCopyUninstallCommandFailure(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "ruff", Source: model.SourcePip}},
		CopyToClipboard: func(string) error {
			return errors.New("clipboard unavailable")
		},
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(Model)

	if !strings.Contains(m.statusMessage, "clipboard unavailable") {
		t.Fatalf("statusMessage = %q, want clipboard error", m.statusMessage)
	}
}

func TestUpdateCopyUninstallCommandWithoutSelection(t *testing.T) {
	m := New(Config{})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(Model)
	if m.statusMessage != "nothing selected" {
		t.Fatalf("statusMessage = %q, want nothing selected", m.statusMessage)
	}
}

func TestGridRowValuesIncludeUsedBy(t *testing.T) {
	columns := []gridColumn{
		{key: "pkg", title: "PKG", width: 3},
		{key: "usedby", title: "DEPENDANCY", width: 6},
	}
	values := gridRowValues(model.Package{Name: "gh", UsedBy: "Y"}, columns)
	if strings.Join(values, ",") != "gh,Y" {
		t.Fatalf("gridRowValues() = %v, want pkg and dependency-safety values", values)
	}
}

func TestRenderFooterAndShrinkColumnsCoverage(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
	})
	m.state.IsLoading = true
	m.statusMessage = "done"
	footer := m.renderFooter()
	if !strings.Contains(footer, "refreshing ") || !strings.Contains(footer, "done") {
		t.Fatalf("renderFooter() = %q, want loading and status", footer)
	}

	columns := []gridColumn{
		{key: "src", title: "SRC", width: 3, noTruncate: true, minWidth: 3},
	}
	if got := shrinkColumns(columns, 2, false, []string{"src"}); got != 2 {
		t.Fatalf("shrinkColumns() = %d, want unconsumed excess when protected", got)
	}

	columns = []gridColumn{
		{key: "pkg", title: "PKG", width: 6},
	}
	if got := shrinkColumns(columns, 1, true, []string{"missing"}); got != 1 {
		t.Fatalf("shrinkColumns() = %d, want unconsumed excess when keys do not match", got)
	}
}

func TestCalculateDynamicColumnWidthsFallsBackToProtectedColumns(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{
				Name:           "package-with-a-long-name",
				Version:        "123.456.789",
				Source:         model.SourceHomebrewCask,
				Description:    "Extremely long description that forces aggressive shrinking",
				UpdatedAt:      "2026-03-01",
				ActionRequired: "update required soon",
				UsedBy:         "no",
			},
		},
	})
	m.grid.Width = 29

	columns := []gridColumn{
		{key: "pkg", title: "PKG", width: 3, minWidth: 8},
		{key: "src", title: "SRC", width: 3, noTruncate: true, minWidth: 3},
		{key: "updated", title: "LAST UPDATED", width: 4, minWidth: 4, fixed: true},
	}
	m.calculateDynamicColumnWidths(columns)

	if columns[1].width >= len("homebrew-cask") {
		t.Fatalf("src width = %d, want protected column to shrink in final pass", columns[1].width)
	}
}
