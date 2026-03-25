package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nad/pkgview/internal/model"
)

func TestUpdateInfoModalOpensAndCloses(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Version: "2.0.0", Source: model.SourceHomebrew, Description: "GitHub CLI", UsedBy: "Y"},
		},
		Inspect: func(context.Context, model.Package) (model.PackageDetails, error) {
			return model.PackageDetails{
				Homepage:     "https://cli.github.com",
				Dependencies: []string{"git", "openssl"},
				Size:         "12 MB",
			}, nil
		},
	})
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = updated.(Model)
	if !m.infoOpen {
		t.Fatal("infoOpen = false, want true")
	}
	if !m.infoLoading {
		t.Fatal("infoLoading = false, want true before details load")
	}
	if cmd == nil {
		t.Fatal("expected inspect command")
	}

	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	if m.infoLoading {
		t.Fatal("infoLoading = true, want false after details load")
	}

	view := m.View()
	if !strings.Contains(view, "Package Info") || !strings.Contains(view, "https://cli.github.com") || !strings.Contains(view, "git, openssl") {
		t.Fatalf("View() = %q, want info modal details", view)
	}
	lines := strings.Split(view, "\n")
	found := -1
	for i, line := range lines {
		if strings.Contains(line, "Package Info") {
			found = i
			if idx := strings.Index(line, "Package Info"); idx < 20 {
				t.Fatalf("Package Info should be centered, line = %q", line)
			}
			break
		}
	}
	if found < 0 {
		t.Fatal("Package Info line not found")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.infoOpen {
		t.Fatal("infoOpen = true, want false after Esc")
	}
}

func TestEnterIsNoOpWhileInfoModalOpen(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		Inspect: func(context.Context, model.Package) (model.PackageDetails, error) {
			return model.PackageDetails{}, nil
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = updated.(Model)
	headerBefore := m.renderGridHeader()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = updated.(Model)
	updated, _ = m.Update(cmd())
	m = updated.(Model)
	if !m.infoOpen {
		t.Fatal("infoOpen = false, want true")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.infoOpen {
		t.Fatal("infoOpen = false after enter, want still open")
	}
	if m.detailOpen {
		t.Fatal("detailOpen = true, want false while modal open")
	}
	if got := m.renderGridHeader(); got != headerBefore {
		t.Fatalf("renderGridHeader() changed under modal enter\nbefore: %q\nafter: %q", headerBefore, got)
	}
}

func TestFullViewKeepsGridVisibleUnderInfoModal(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Version: "2.0.0", Source: model.SourceHomebrew, Description: "GitHub CLI", UsedBy: "Y"},
		},
		Inspect: func(context.Context, model.Package) (model.PackageDetails, error) {
			return model.PackageDetails{Homepage: "https://cli.github.com"}, nil
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = updated.(Model)
	updated, _ = m.Update(cmd())
	m = updated.(Model)

	view := m.View()
	if !strings.Contains(view, "Package Info") {
		t.Fatalf("View() = %q, want info modal", view)
	}
	if !strings.Contains(view, "PKG") || !strings.Contains(view, "DESCRIPTION") {
		t.Fatalf("View() = %q, want grid to remain visible beneath the modal", view)
	}
}

func TestInfoModalUsesFallbackValuesAndHelpEntry(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "ruff", Version: "0.4.0", Source: model.SourcePip, UsedBy: "-"},
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected inspect command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(Model)

	if got := strings.Join(m.modeHelpParts(), " "); !strings.Contains(got, "i:info") {
		t.Fatalf("modeHelpParts() = %q, want i:info", got)
	}
	view := m.View()
	if !strings.Contains(view, "Homepage: -") || !strings.Contains(view, "Dependencies: -") || !strings.Contains(view, "Dependancy: -") {
		t.Fatalf("View() = %q, want fallback placeholders in info modal", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if len([]rune(line)) > 100 {
			t.Fatalf("line width %d exceeds window width: %q", len([]rune(line)), line)
		}
	}
}

func TestMouseSelectsRowAndTogglesInfoModal(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "ruff", Source: model.SourcePip},
		},
		Inspect: func(context.Context, model.Package) (model.PackageDetails, error) {
			return model.PackageDetails{Homepage: "https://example.com"}, nil
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)

	x, y := m.gridRowPoint(1)
	updated, _ = m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if m.state.Selected != 1 {
		t.Fatalf("Selected = %d, want 1", m.state.Selected)
	}
	if m.infoOpen {
		t.Fatal("infoOpen = true, want false after first click")
	}

	updated, cmd := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if !m.infoOpen {
		t.Fatal("infoOpen = false, want true after second click")
	}
	if cmd == nil {
		t.Fatal("expected inspect command on second click")
	}
	updated, _ = m.Update(cmd())
	m = updated.(Model)

	updated, _ = m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if m.infoOpen {
		t.Fatal("infoOpen = true, want false after third click toggles modal off")
	}
}

func TestMouseClickOutsideRowsDoesNotChangeSelection(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "ruff", Source: model.SourcePip},
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)
	m.state.Selected = 1

	updated, _ = m.Update(tea.MouseMsg{X: 90, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(Model)
	if m.state.Selected != 1 {
		t.Fatalf("Selected = %d, want unchanged", m.state.Selected)
	}
}

func TestInfoHelpersCoverFallbackBranches(t *testing.T) {
	m := New(Config{})
	if got := m.View(); !strings.Contains(got, "No packages match") {
		t.Fatalf("View() = %q, want empty-state view", got)
	}

	m.infoOpen = true
	if got := m.renderInfoModal(); got != "" {
		t.Fatalf("renderInfoModal() = %q, want empty without selection", got)
	}
	base := "base"
	if got := m.renderInfoOverlay(base); got != base {
		t.Fatalf("renderInfoOverlay() = %q, want base when modal empty", got)
	}

	opened, cmd := m.openInfoModal()
	m = opened
	if cmd != nil || m.statusMessage != "nothing selected" {
		t.Fatalf("openInfoModal() = %q, want nothing selected", m.statusMessage)
	}

	overlay := overlayCentered(20, 5, "XYZ")
	if !strings.Contains(overlay, "XYZ") {
		t.Fatalf("overlayCentered() = %q, want overlay content", overlay)
	}
	if got := overlayCentered(20, 5, ""); got != "" {
		t.Fatalf("overlayCentered(empty) = %q, want empty", got)
	}
	if got := maxRenderedLineWidth("aa\nbbbb"); got != 4 {
		t.Fatalf("maxRenderedLineWidth() = %d, want 4", got)
	}
}

func TestGridRowHelpersAndMouseFallbacks(t *testing.T) {
	m := New(Config{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)
	m.popupKind = PopupSource

	x, y := m.gridRowPoint(0)
	if x <= 0 || y <= 0 {
		t.Fatalf("gridRowPoint() = %d,%d", x, y)
	}
	if got := m.gridRowIndexAt(-1, y); got != -1 {
		t.Fatalf("gridRowIndexAt(negative) = %d, want -1", got)
	}
	gridWidth, _ := m.fullGridSize()
	if got := m.gridRowIndexAt(gridWidth, y); got != -1 {
		t.Fatalf("gridRowIndexAt(outside x) = %d, want -1", got)
	}
	if got := m.gridRowIndexAt(x, y+10); got != -1 {
		t.Fatalf("gridRowIndexAt(outside y) = %d, want -1", got)
	}

	updatedModel, _ := m.updateMouse(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})
	m = updatedModel.(Model)
	if m.state.Selected != 0 {
		t.Fatalf("Selected = %d, want unchanged", m.state.Selected)
	}

	m.layout = LayoutCompact
	updatedModel, _ = m.updateMouse(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updatedModel.(Model)
	if m.layout != LayoutCompact {
		t.Fatalf("layout = %q, want compact", m.layout)
	}
}

func TestInfoModalAdditionalBranches(t *testing.T) {
	m := New(Config{
		Layout: LayoutCompact,
		Packages: []model.Package{
			{Name: "gh", Version: "2.0.0", Source: model.SourceHomebrew, UsedBy: "Y"},
		},
	})
	m.infoOpen = true
	m.infoLoading = true
	m.infoError = "inspect failed"
	m.state.Width = 100
	m.state.Height = 30
	view := m.View()
	if !strings.Contains(view, "Package Info") || !strings.Contains(view, "Loading: live metadata") || !strings.Contains(view, "Error: inspect failed") {
		t.Fatalf("View() = %q, want compact overlay with loading and error", view)
	}

	updated, _ := m.Update(infoLoadedMsg{key: "other", details: model.PackageDetails{Homepage: "skip"}})
	m = updated.(Model)
	if m.infoDetails.Homepage != "" {
		t.Fatalf("Homepage = %q, want ignored mismatched payload", m.infoDetails.Homepage)
	}

	m.infoKey = model.Package{Name: "gh", Source: model.SourceHomebrew}.Key()
	updated, _ = m.Update(infoLoadedMsg{key: m.infoKey, err: context.DeadlineExceeded})
	m = updated.(Model)
	if !strings.Contains(m.infoError, "deadline") {
		t.Fatalf("infoError = %q, want deadline error", m.infoError)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = updated.(Model)
	if m.infoOpen {
		t.Fatal("infoOpen = true, want false after i closes modal")
	}
	if cmd != nil {
		t.Fatal("unexpected command when closing modal with i")
	}

	m.infoOpen = true
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = updated.(Model)
	if !m.infoOpen || cmd != nil {
		t.Fatal("unexpected change while modal swallows unrelated keys")
	}

	updatedModel, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if quitCmd == nil {
		t.Fatal("expected quit command from modal q")
	}
	if _, ok := updatedModel.(Model); !ok {
		t.Fatal("expected model return when quitting from modal")
	}

	updatedModel, cmd2 := m.Update(struct{}{})
	if cmd2 != nil {
		t.Fatal("unexpected command for non-key non-mouse message")
	}
	if _, ok := updatedModel.(Model); !ok {
		t.Fatal("expected model return for unknown message")
	}
}
