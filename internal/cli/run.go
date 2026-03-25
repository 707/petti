package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/707/petti/internal/app"
	"github.com/707/petti/internal/collectors"
	exports "github.com/707/petti/internal/export"
	"github.com/707/petti/internal/model"
	"github.com/707/petti/internal/tui"
)

type Deps struct {
	Stdout    io.Writer
	Stderr    io.Writer
	Version   string
	Refresh   func(context.Context) collectors.CollectResult
	WriteTXT  func(string, []model.Package) error
	WriteJSON func(string, []model.Package) error
	RunTUI    func(tea.Model) error
}

type teaProgram interface {
	Run() (tea.Model, error)
}

var (
	defaultCollectorFactory = func(runner collectors.Runner) []collectors.Collector {
		return []collectors.Collector{
			collectors.NewBrewCollector(runner),
			collectors.NewNPMCollector(runner),
			collectors.NewPipCollector(runner),
		}
	}
	defaultCollectAll = collectors.CollectAll
	newTeaProgram     = func(model tea.Model) teaProgram {
		return tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	}
)

func Run(ctx context.Context, args []string, deps Deps) int {
	deps = withDefaults(deps)
	inspector := collectors.NewPackageInspector(collectors.ExecRunner{})

	fs := flag.NewFlagSet("petti", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	exportTXTPath := fs.String("export-txt", "", "export list to .txt and exit")
	exportJSONPath := fs.String("export-json", "", "export list to .json and exit")
	filterValue := fs.String("filter", "", "pre-populate the filter")
	layoutValue := fs.String("layout", string(tui.LayoutFull), "startup layout: full or compact")
	themeValue := fs.String("theme", string(tui.ThemeDefault), "startup theme")
	_ = fs.Bool("no-color", false, "disable colour output")
	showVersion := fs.Bool("version", false, "show version and exit")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		_, _ = fmt.Fprintln(deps.Stdout, deps.Version)
		return 0
	}
	if *exportTXTPath != "" && *exportJSONPath != "" {
		_, _ = fmt.Fprintln(deps.Stderr, "choose only one export flag")
		return 2
	}
	if *layoutValue != string(tui.LayoutFull) && *layoutValue != string(tui.LayoutCompact) {
		_, _ = fmt.Fprintln(deps.Stderr, "layout must be full or compact")
		return 2
	}
	if !tui.IsValidTheme(tui.ThemeName(*themeValue)) {
		names := make([]string, 0, len(tui.ValidThemes()))
		for _, name := range tui.ValidThemes() {
			names = append(names, string(name))
		}
		_, _ = fmt.Fprintf(deps.Stderr, "theme must be one of: %s\n", strings.Join(names, ", "))
		return 2
	}

	result := deps.Refresh(ctx)
	visible := visiblePackages(result.Packages, result.Statuses, *filterValue)

	if *exportTXTPath != "" {
		if err := deps.WriteTXT(*exportTXTPath, visible); err != nil {
			_, _ = fmt.Fprintln(deps.Stderr, err)
			return 1
		}
		return 0
	}
	if *exportJSONPath != "" {
		if err := deps.WriteJSON(*exportJSONPath, visible); err != nil {
			_, _ = fmt.Fprintln(deps.Stderr, err)
			return 1
		}
		return 0
	}

	model := tui.New(tui.Config{
		Packages: result.Packages,
		Statuses: result.Statuses,
		Filter:   *filterValue,
		Layout:   tui.LayoutMode(*layoutValue),
		Theme:    tui.ThemeName(*themeValue),
		Refresh:  deps.Refresh,
		Export:   makeExportFunc(deps),
		Inspect:  inspector.Inspect,
	})
	if err := deps.RunTUI(model); err != nil {
		_, _ = fmt.Fprintln(deps.Stderr, err)
		return 1
	}
	return 0
}

func withDefaults(deps Deps) Deps {
	if deps.Stdout == nil {
		deps.Stdout = io.Discard
	}
	if deps.Stderr == nil {
		deps.Stderr = io.Discard
	}
	if deps.Version == "" {
		deps.Version = "dev"
	}
	if deps.Refresh == nil {
		deps.Refresh = defaultRefresh
	}
	if deps.WriteTXT == nil {
		deps.WriteTXT = exports.WriteTXT
	}
	if deps.WriteJSON == nil {
		deps.WriteJSON = exports.WriteJSON
	}
	if deps.RunTUI == nil {
		deps.RunTUI = func(model tea.Model) error {
			_, err := newTeaProgram(model).Run()
			return err
		}
	}
	return deps
}

func visiblePackages(packages []model.Package, statuses []model.CollectorStatus, filter string) []model.Package {
	state := app.State{
		Packages: packages,
		Statuses: statuses,
		Filter:   filter,
	}
	return state.VisiblePackages()
}

func makeExportFunc(deps Deps) tui.ExportFunc {
	return func(format tui.ExportFormat, packages []model.Package) (string, error) {
		switch format {
		case tui.ExportJSON:
			path := "petti-export.json"
			return path, deps.WriteJSON(path, packages)
		default:
			path := "petti-export.txt"
			return path, deps.WriteTXT(path, packages)
		}
	}
}

func defaultRefresh(ctx context.Context) collectors.CollectResult {
	runner := collectors.ExecRunner{}
	return defaultCollectAll(ctx, defaultCollectorFactory(runner))
}
