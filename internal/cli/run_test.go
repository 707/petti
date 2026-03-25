package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nad/pkgview/internal/collectors"
	"github.com/nad/pkgview/internal/model"
	"github.com/nad/pkgview/internal/tui"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	code := Run(context.Background(), []string{"--version"}, Deps{
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
		Version: "1.2.3",
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if strings.TrimSpace(stdout.String()) != "1.2.3" {
		t.Fatalf("stdout = %q, want version", stdout.String())
	}
}

func TestRunExportTXTUsesFilter(t *testing.T) {
	var exported []model.Package
	code := Run(context.Background(), []string{"--filter", "gh", "--export-txt", "out.txt"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{
				Packages: []model.Package{
					{Name: "gh", Source: model.SourceHomebrew},
					{Name: "ruff", Source: model.SourcePip},
				},
			}
		},
		WriteTXT: func(_ string, pkgs []model.Package) error {
			exported = pkgs
			return nil
		},
		WriteJSON: func(string, []model.Package) error { return nil },
		RunTUI:    func(tea.Model) error { return nil },
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if len(exported) != 1 || exported[0].Name != "gh" {
		t.Fatalf("exported = %#v, want only gh", exported)
	}
}

func TestRunExportJSON(t *testing.T) {
	var called bool
	code := Run(context.Background(), []string{"--export-json", "out.json"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{}
		},
		WriteTXT: func(string, []model.Package) error { return nil },
		WriteJSON: func(path string, pkgs []model.Package) error {
			called = path == "out.json"
			return nil
		},
		RunTUI: func(tea.Model) error { return nil },
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if !called {
		t.Fatal("WriteJSON was not called")
	}
}

func TestRunTUIError(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), nil, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{}
		},
		WriteTXT:  func(string, []model.Package) error { return nil },
		WriteJSON: func(string, []model.Package) error { return nil },
		RunTUI: func(tea.Model) error {
			return errors.New("boom")
		},
	})
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("stderr = %q, want error", stderr.String())
	}
}

func TestRunTUISuccess(t *testing.T) {
	var gotModel any
	code := Run(context.Background(), nil, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{}
		},
		WriteTXT:  func(string, []model.Package) error { return nil },
		WriteJSON: func(string, []model.Package) error { return nil },
		RunTUI: func(model tea.Model) error {
			gotModel = model
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if gotModel == nil {
		t.Fatal("expected TUI model")
	}
}

func TestRunRejectsBothExportFlags(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"--export-txt", "a", "--export-json", "b"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
	})
	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "choose only one") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFlagParseError(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"--unknown"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
	})
	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if stderr.Len() == 0 {
		t.Fatal("expected parse error output")
	}
}

func TestRunLayoutFlag(t *testing.T) {
	var rendered tui.Model
	code := Run(context.Background(), []string{"--layout", "compact"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{}
		},
		WriteTXT:  func(string, []model.Package) error { return nil },
		WriteJSON: func(string, []model.Package) error { return nil },
		RunTUI: func(model tea.Model) error {
			rendered = model.(tui.Model)
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if strings.Contains(rendered.View(), "Selected Package") {
		t.Fatalf("compact view should not render full-layout detail pane: %q", rendered.View())
	}
}

func TestRunRejectsInvalidLayout(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"--layout", "wide"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
	})
	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "layout must be full or compact") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunThemeFlag(t *testing.T) {
	var rendered tui.Model
	code := Run(context.Background(), []string{"--theme", "ember"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{}
		},
		WriteTXT:  func(string, []model.Package) error { return nil },
		WriteJSON: func(string, []model.Package) error { return nil },
		RunTUI: func(model tea.Model) error {
			rendered = model.(tui.Model)
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if rendered.ThemeName() != tui.ThemeEmber {
		t.Fatalf("ThemeName() = %q, want %q", rendered.ThemeName(), tui.ThemeEmber)
	}
}

func TestRunThemeFlagAcceptsExplicitVariant(t *testing.T) {
	var rendered tui.Model
	code := Run(context.Background(), []string{"--theme", "default-light"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{}
		},
		WriteTXT:  func(string, []model.Package) error { return nil },
		WriteJSON: func(string, []model.Package) error { return nil },
		RunTUI: func(model tea.Model) error {
			rendered = model.(tui.Model)
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if rendered.ThemeName() != tui.ThemeDefaultLight {
		t.Fatalf("ThemeName() = %q, want %q", rendered.ThemeName(), tui.ThemeDefaultLight)
	}
}

func TestRunRejectsInvalidTheme(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"--theme", "nope"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
	})
	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "default-light") || !strings.Contains(stderr.String(), "frost-light") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunExportWriteError(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"--export-txt", "out.txt"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{}
		},
		WriteTXT:  func(string, []model.Package) error { return errors.New("write failed") },
		WriteJSON: func(string, []model.Package) error { return nil },
		RunTUI:    func(tea.Model) error { return nil },
	})
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "write failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunExportJSONWriteError(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"--export-json", "out.json"}, Deps{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		Refresh: func(context.Context) collectors.CollectResult {
			return collectors.CollectResult{}
		},
		WriteTXT:  func(string, []model.Package) error { return nil },
		WriteJSON: func(string, []model.Package) error { return errors.New("json failed") },
		RunTUI:    func(tea.Model) error { return nil },
	})
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "json failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestWithDefaultsAndMakeExportFunc(t *testing.T) {
	deps := withDefaults(Deps{})
	if deps.Version != "dev" {
		t.Fatalf("Version = %q, want dev", deps.Version)
	}
	if deps.Stdout == nil || deps.Stderr == nil || deps.Refresh == nil || deps.WriteTXT == nil || deps.WriteJSON == nil || deps.RunTUI == nil {
		t.Fatal("expected defaults to be populated")
	}
	originalProgram := newTeaProgram
	t.Cleanup(func() {
		newTeaProgram = originalProgram
	})
	newTeaProgram = func(tea.Model) teaProgram {
		return stubTeaProgram{}
	}
	if err := deps.RunTUI(nil); err != nil {
		t.Fatalf("default RunTUI error = %v", err)
	}

	exportFunc := makeExportFunc(Deps{
		WriteTXT: func(path string, pkgs []model.Package) error {
			if path != "pkgview-export.txt" || len(pkgs) != 1 {
				t.Fatalf("txt path/packages = %q/%d", path, len(pkgs))
			}
			return nil
		},
		WriteJSON: func(path string, pkgs []model.Package) error {
			if path != "pkgview-export.json" || len(pkgs) != 1 {
				t.Fatalf("json path/packages = %q/%d", path, len(pkgs))
			}
			return nil
		},
	})
	if _, err := exportFunc("txt", []model.Package{{Name: "gh"}}); err != nil {
		t.Fatalf("export txt error = %v", err)
	}
	if _, err := exportFunc("json", []model.Package{{Name: "gh"}}); err != nil {
		t.Fatalf("export json error = %v", err)
	}
}

func TestDefaultRefreshUsesCollectorFactoryAndAggregator(t *testing.T) {
	originalFactory := defaultCollectorFactory
	originalCollectAll := defaultCollectAll
	t.Cleanup(func() {
		defaultCollectorFactory = originalFactory
		defaultCollectAll = originalCollectAll
	})

	defaultCollectorFactory = func(collectors.Runner) []collectors.Collector {
		return []collectors.Collector{
			staticCollector{source: model.SourceHomebrew},
		}
	}
	defaultCollectAll = func(_ context.Context, list []collectors.Collector) collectors.CollectResult {
		if len(list) != 1 {
			t.Fatalf("len(list) = %d, want 1", len(list))
		}
		return collectors.CollectResult{
			Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
		}
	}

	result := defaultRefresh(context.Background())
	if len(result.Packages) != 1 || result.Packages[0].Name != "gh" {
		t.Fatalf("result = %#v", result)
	}

	defaultCollectorFactory = originalFactory
	list := defaultCollectorFactory(collectors.ExecRunner{})
	if len(list) != 3 {
		t.Fatalf("len(defaultCollectorFactory) = %d, want 3", len(list))
	}
}

func TestNewTeaProgram(t *testing.T) {
	program := newTeaProgram(nil)
	if program == nil {
		t.Fatal("newTeaProgram() returned nil")
	}
}

type staticCollector struct {
	source model.Source
}

func (s staticCollector) Name() model.Source             { return s.source }
func (s staticCollector) Available(context.Context) bool { return true }
func (s staticCollector) Collect(context.Context) ([]model.Package, model.CollectorStatus, error) {
	return nil, model.CollectorStatus{Source: s.source, State: model.CollectorStateReady}, nil
}

type stubTeaProgram struct{}

func (stubTeaProgram) Run() (tea.Model, error) { return nil, nil }
