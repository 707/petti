package collectors

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/707/petti/internal/model"
)

type stubRunner struct {
	paths   map[string]error
	results map[string]Result
	errs    map[string]error
}

func (s stubRunner) LookPath(file string) (string, error) {
	if err, ok := s.paths[file]; ok {
		return "", err
	}
	return "/usr/bin/" + file, nil
}

func (s stubRunner) Run(_ context.Context, command string, args ...string) (Result, error) {
	key := command
	for _, arg := range args {
		key += " " + arg
	}
	if err, ok := s.errs[key]; ok {
		return Result{}, err
	}
	if result, ok := s.results[key]; ok {
		return result, nil
	}
	return Result{}, nil
}

func TestBrewCollectorCollectsFormulaeAndCasks(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"brew leaves --installed-on-request": {Stdout: "gh\nfzf"},
			"brew list --versions":               {Stdout: "gh 2.47.0\nfzf 0.54.0"},
			"brew list --cask --versions":        {Stdout: "font-hack-nerd-font 3.4.0"},
			"brew info --json=v2 gh fzf": {Stdout: `{"formulae":[
				{"name":"gh","desc":"GitHub CLI","outdated":true,"installed":[{"time":1704067200}]},
				{"name":"fzf","desc":"Fuzzy finder","outdated":false,"installed":[{"time":1706745600}]}
			],"casks":[]}`},
			"brew info --json=v2 --cask font-hack-nerd-font": {Stdout: `{"formulae":[],"casks":[
				{"token":"font-hack-nerd-font","desc":"Fonts","outdated":false,"installed_time":1709251200}
			]}`},
		},
	}

	pkgs, status, err := NewBrewCollector(runner).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if status.State != model.CollectorStateReady {
		t.Fatalf("status.State = %q", status.State)
	}

	want := []model.Package{
		{Name: "gh", Version: "2.47.0", Source: model.SourceHomebrew, Description: "GitHub CLI", ActionRequired: "update", UpdatedAt: "2024-01-01", UsedBy: "N"},
		{Name: "fzf", Version: "0.54.0", Source: model.SourceHomebrew, Description: "Fuzzy finder", ActionRequired: "current", UpdatedAt: "2024-02-01", UsedBy: "N"},
		{Name: "font-hack-nerd-font", Version: "3.4.0", Source: model.SourceHomebrewCask, Description: "Fonts", ActionRequired: "current", UpdatedAt: "2024-03-01", UsedBy: "-"},
	}
	if !reflect.DeepEqual(pkgs, want) {
		t.Fatalf("Collect() packages = %#v, want %#v", pkgs, want)
	}
}

func TestCollectorNames(t *testing.T) {
	if got := NewBrewCollector(stubRunner{}).Name(); got != model.SourceHomebrew {
		t.Fatalf("BrewCollector.Name() = %q", got)
	}
	if got := NewNPMCollector(stubRunner{}).Name(); got != model.SourceNPM {
		t.Fatalf("NPMCollector.Name() = %q", got)
	}
	if got := NewPipCollector(stubRunner{}).Name(); got != model.SourcePip {
		t.Fatalf("PipCollector.Name() = %q", got)
	}
}

func TestBrewCollectorMissingBinary(t *testing.T) {
	runner := stubRunner{paths: map[string]error{"brew": exec.ErrNotFound}}
	_, status, err := NewBrewCollector(runner).Collect(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Collect() error = %v, want ErrUnavailable", err)
	}
	if status.State != model.CollectorStateMissing {
		t.Fatalf("status.State = %q, want %q", status.State, model.CollectorStateMissing)
	}
}

func TestBrewCollectorTimeout(t *testing.T) {
	runner := stubRunner{
		errs: map[string]error{
			"brew leaves --installed-on-request": context.DeadlineExceeded,
		},
	}
	_, status, err := NewBrewCollector(runner).Collect(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Collect() error = %v", err)
	}
	if status.State != model.CollectorStateTimeout {
		t.Fatalf("status.State = %q", status.State)
	}
}

func TestBrewCollectFormulaeExitErrorUsesExitStatus(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"brew leaves --installed-on-request": {ExitCode: 2},
		},
	}
	_, status, err := NewBrewCollector(runner).collectFormulae(context.Background())
	if err == nil {
		t.Fatal("collectFormulae() error = nil, want error")
	}
	if status.Details != "exit code 2" {
		t.Fatalf("status.Details = %q, want %q", status.Details, "exit code 2")
	}
}

func TestBrewCollectFormulaeSkipsBlankLines(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"brew leaves --installed-on-request": {Stdout: "gh\n\nfzf\n"},
			"brew list --versions":               {Stdout: "gh 2.47.0\nfzf 0.54.0"},
		},
	}
	pkgs, _, err := NewBrewCollector(runner).collectFormulae(context.Background())
	if err != nil {
		t.Fatalf("collectFormulae() error = %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("len(packages) = %d, want 2", len(pkgs))
	}
}

func TestBrewCollectFormulaeVersionTimeout(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"brew leaves --installed-on-request": {Stdout: "gh"},
		},
		errs: map[string]error{
			"brew list --versions": context.DeadlineExceeded,
		},
	}
	_, status, err := NewBrewCollector(runner).collectFormulae(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("collectFormulae() error = %v", err)
	}
	if status.State != model.CollectorStateTimeout {
		t.Fatalf("status.State = %q", status.State)
	}
}

func TestBrewCollectFormulaeVersionExitError(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"brew leaves --installed-on-request": {Stdout: "gh"},
			"brew list --versions":               {ExitCode: 1, Stderr: "bad versions"},
		},
	}
	_, status, err := NewBrewCollector(runner).collectFormulae(context.Background())
	if err == nil {
		t.Fatal("collectFormulae() error = nil, want error")
	}
	if status.Details != "bad versions" {
		t.Fatalf("status.Details = %q, want %q", status.Details, "bad versions")
	}
}

func TestBrewCollectCasksTimeout(t *testing.T) {
	runner := stubRunner{
		errs: map[string]error{
			"brew list --cask --versions": context.DeadlineExceeded,
		},
	}
	_, status, err := NewBrewCollector(runner).collectCasks(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("collectCasks() error = %v", err)
	}
	if status.State != model.CollectorStateTimeout {
		t.Fatalf("status.State = %q", status.State)
	}
}

func TestParseVersionLinesSkipsMalformedLines(t *testing.T) {
	got := parseVersionLines("gh 2.47.0\ninvalid\n\nfzf 0.54.0")
	want := map[string]string{"gh": "2.47.0", "fzf": "0.54.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseVersionLines() = %#v, want %#v", got, want)
	}
}

func TestExitStatusUsesStderrWhenPresent(t *testing.T) {
	status := exitStatus(model.SourceNPM, "npm", Result{ExitCode: 1, Stderr: "boom"})
	if status.Details != "boom" {
		t.Fatalf("status.Details = %q, want %q", status.Details, "boom")
	}
}

func TestBrewCollectorKeepsFormulaeWhenCasksFail(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"brew leaves --installed-on-request": {Stdout: "gh"},
			"brew list --versions":               {Stdout: "gh 2.47.0"},
			"brew list --cask --versions":        {ExitCode: 1, Stderr: "cask failure"},
		},
	}

	pkgs, status, err := NewBrewCollector(runner).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	want := []model.Package{{Name: "gh", Version: "2.47.0", Source: model.SourceHomebrew, UsedBy: "N"}}
	if !reflect.DeepEqual(pkgs, want) {
		t.Fatalf("Collect() packages = %#v, want %#v", pkgs, want)
	}
	if status.State != model.CollectorStateError {
		t.Fatalf("status.State = %q, want %q", status.State, model.CollectorStateError)
	}
	if status.Details != "cask failure" {
		t.Fatalf("status.Details = %q, want %q", status.Details, "cask failure")
	}
}

func TestBrewDependencySafetyIsBulkDerived(t *testing.T) {
	packages := []model.Package{
		{Name: "gh", Source: model.SourceHomebrew, UsedBy: "Y"},
		{Name: "fzf", Source: model.SourceHomebrew, UsedBy: "Y"},
		{Name: "dep", Source: model.SourceHomebrew, UsedBy: "Y"},
		{Name: "font-hack-nerd-font", Source: model.SourceHomebrewCask, UsedBy: "Y"},
	}
	leaves := map[string]struct{}{
		"gh":  {},
		"fzf": {},
	}

	got := applyBrewDependencySafety(append([]model.Package(nil), packages...), leaves)
	if got[0].UsedBy != "N" {
		t.Fatalf("gh UsedBy = %q, want N", got[0].UsedBy)
	}
	if got[1].UsedBy != "N" {
		t.Fatalf("fzf UsedBy = %q, want N", got[1].UsedBy)
	}
	if got[2].UsedBy != "Y" {
		t.Fatalf("dep UsedBy = %q, want Y", got[2].UsedBy)
	}
	if got[3].UsedBy != "-" {
		t.Fatalf("cask UsedBy = %q, want -", got[3].UsedBy)
	}
}

func TestBrewCollectorRetainsDependencyInstalledFormulae(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"brew leaves --installed-on-request": {Stdout: "gh"},
			"brew list --versions":               {Stdout: "gh 2.47.0\ndep 1.0.0"},
			"brew info --json=v2 gh dep": {Stdout: `{"formulae":[
				{"name":"gh","desc":"GitHub CLI","outdated":false,"installed":[{"time":1704067200}]},
				{"name":"dep","desc":"Dependency package","outdated":false,"installed":[{"time":1704067200}]}
			],"casks":[]}`},
			"brew list --cask --versions": {Stdout: ""},
		},
	}

	pkgs, _, err := NewBrewCollector(runner).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	want := []model.Package{
		{Name: "gh", Version: "2.47.0", Source: model.SourceHomebrew, Description: "GitHub CLI", ActionRequired: "current", UpdatedAt: "2024-01-01", UsedBy: "N"},
		{Name: "dep", Version: "1.0.0", Source: model.SourceHomebrew, Description: "Dependency package", ActionRequired: "current", UpdatedAt: "2024-01-01", UsedBy: "Y"},
	}
	if !reflect.DeepEqual(pkgs, want) {
		t.Fatalf("Collect() packages = %#v, want %#v", pkgs, want)
	}
}

func TestNPMCollectorStripsNPM(t *testing.T) {
	dir := t.TempDir()
	packageDir := filepath.Join(dir, "typescript")
	if err := os.Mkdir(packageDir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	mtime := time.Date(2024, time.July, 4, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(packageDir, mtime, mtime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	runner := stubRunner{
		results: map[string]Result{
			"npm list -g --depth=0 --json -l": {Stdout: `{"dependencies":{"npm":{"version":"10.0.0"},"typescript":{"version":"5.4.3","description":"TypeScript language","path":"` + packageDir + `"}}}`},
			"npm ls -g --all --json":          {Stdout: `{"dependencies":{"typescript":{"version":"5.4.3"}}}`},
		},
	}
	pkgs, _, err := NewNPMCollector(runner).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	want := []model.Package{{Name: "typescript", Version: "5.4.3", Source: model.SourceNPM, Description: "TypeScript language", UpdatedAt: "2024-07-04", UsedBy: "N"}}
	if !reflect.DeepEqual(pkgs, want) {
		t.Fatalf("Collect() = %#v, want %#v", pkgs, want)
	}
}

func TestNPMCollectorMarksDependencyUnsafeWhenNestedUnderAnotherPackage(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"npm list -g --depth=0 --json -l": {Stdout: `{"dependencies":{"npm":{"version":"10.0.0"},"typescript":{"version":"5.4.3","description":"TypeScript language","path":"/tmp/typescript"},"eslint":{"version":"9.0.0","description":"Linter","path":"/tmp/eslint"}}}`},
			"npm ls -g --all --json":          {Stdout: `{"dependencies":{"typescript":{"version":"5.4.3"},"eslint":{"version":"9.0.0","dependencies":{"typescript":{"version":"5.4.3"}}}}}`},
		},
	}
	pkgs, _, err := NewNPMCollector(runner).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	for _, pkg := range pkgs {
		if pkg.Name == "typescript" && pkg.UsedBy != "Y" {
			t.Fatalf("typescript UsedBy = %q, want Y", pkg.UsedBy)
		}
		if pkg.Name == "eslint" && pkg.UsedBy != "N" {
			t.Fatalf("eslint UsedBy = %q, want N", pkg.UsedBy)
		}
	}
}

func TestNPMDependencySafetyHelpers(t *testing.T) {
	base := []model.Package{
		{Name: "typescript", Source: model.SourceNPM, UsedBy: "-"},
		{Name: "eslint", Source: model.SourceNPM, UsedBy: "-"},
	}
	got := markNPMDependencySafety(append([]model.Package(nil), base...), `{"dependencies":{"typescript":{"version":"5.4.3"},"eslint":{"version":"9.0.0","dependencies":{"typescript":{"version":"5.4.3"}}}}}`)
	if got[0].UsedBy != "Y" {
		t.Fatalf("typescript UsedBy = %q, want Y", got[0].UsedBy)
	}
	if got[1].UsedBy != "N" {
		t.Fatalf("eslint UsedBy = %q, want N", got[1].UsedBy)
	}

	unchanged := markNPMDependencySafety(append([]model.Package(nil), base...), "{bad json")
	if !reflect.DeepEqual(unchanged, base) {
		t.Fatalf("markNPMDependencySafety(invalid) = %#v, want unchanged", unchanged)
	}

	runner := stubRunner{
		errs: map[string]error{
			"npm ls -g --all --json": context.DeadlineExceeded,
		},
	}
	got = NewNPMCollector(runner).applyDependencySafety(context.Background(), append([]model.Package(nil), base...))
	if !reflect.DeepEqual(got, base) {
		t.Fatalf("applyDependencySafety(err) = %#v, want unchanged", got)
	}

	exitRunner := stubRunner{
		results: map[string]Result{
			"npm ls -g --all --json": {ExitCode: 1},
		},
	}
	got = NewNPMCollector(exitRunner).applyDependencySafety(context.Background(), append([]model.Package(nil), base...))
	if !reflect.DeepEqual(got, base) {
		t.Fatalf("applyDependencySafety(exit) = %#v, want unchanged", got)
	}

	got = NewNPMCollector(stubRunner{}).applyDependencySafety(context.Background(), nil)
	if got != nil {
		t.Fatalf("applyDependencySafety(nil) = %#v, want nil", got)
	}

	unchanged = markNPMDependencySafety(append([]model.Package(nil), base...), `{"name":"root"}`)
	for _, pkg := range unchanged {
		if pkg.UsedBy != "N" {
			t.Fatalf("markNPMDependencySafety(no deps) UsedBy = %q, want N", pkg.UsedBy)
		}
	}
}

func TestNPMCollectorMissingBinary(t *testing.T) {
	runner := stubRunner{paths: map[string]error{"npm": exec.ErrNotFound}}
	_, status, err := NewNPMCollector(runner).Collect(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Collect() error = %v, want ErrUnavailable", err)
	}
	if status.State != model.CollectorStateMissing {
		t.Fatalf("status.State = %q", status.State)
	}
}

func TestNPMCollectorInvalidJSON(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"npm list -g --depth=0 --json -l": {Stdout: "{not-json"},
		},
	}
	_, status, err := NewNPMCollector(runner).Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() error = nil, want error")
	}
	if status.State != model.CollectorStateError {
		t.Fatalf("status.State = %q", status.State)
	}
}

func TestNPMCollectorTimeout(t *testing.T) {
	runner := stubRunner{
		errs: map[string]error{
			"npm list -g --depth=0 --json -l": context.DeadlineExceeded,
		},
	}
	_, status, err := NewNPMCollector(runner).Collect(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Collect() error = %v", err)
	}
	if status.State != model.CollectorStateTimeout {
		t.Fatalf("status.State = %q", status.State)
	}
}

func TestNPMCollectorExitError(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"npm list -g --depth=0 --json -l": {ExitCode: 1, Stderr: "npm failed"},
		},
	}
	_, status, err := NewNPMCollector(runner).Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() error = nil, want error")
	}
	if status.Details != "npm failed" {
		t.Fatalf("status.Details = %q, want %q", status.Details, "npm failed")
	}
}

func TestPipCollectorFallsBackToPip3(t *testing.T) {
	dir := t.TempDir()
	distInfo := filepath.Join(dir, "ruff-0.3.4.dist-info")
	if err := os.Mkdir(distInfo, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	mtime := time.Date(2024, time.August, 5, 8, 0, 0, 0, time.UTC)
	if err := os.Chtimes(distInfo, mtime, mtime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	runner := stubRunner{
		paths: map[string]error{"pip": exec.ErrNotFound},
		results: map[string]Result{
			"pip3 list --not-required --format=json": {Stdout: `[{"name":"ruff","version":"0.3.4"}]`},
			"pip3 show ruff":                         {Stdout: "Name: ruff\nSummary: Fast Python linter\nLocation: " + dir},
		},
	}
	pkgs, _, err := NewPipCollector(runner).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	want := []model.Package{{Name: "ruff", Version: "0.3.4", Source: model.SourcePip, Description: "Fast Python linter", UpdatedAt: "2024-08-05", UsedBy: "N"}}
	if !reflect.DeepEqual(pkgs, want) {
		t.Fatalf("Collect() = %#v, want %#v", pkgs, want)
	}
}

func TestMetadataHelpers(t *testing.T) {
	if got := formatUnixDate(1704067200); got != "2024-01-01" {
		t.Fatalf("formatUnixDate() = %q, want %q", got, "2024-01-01")
	}
	if got := formatUnixDate(0); got != "" {
		t.Fatalf("formatUnixDate(0) = %q, want empty", got)
	}
	if got := brewActionLabel(true, false, false); got != "update" {
		t.Fatalf("brewActionLabel(update) = %q", got)
	}
	if got := brewActionLabel(false, true, false); got != "attention" {
		t.Fatalf("brewActionLabel(deprecated) = %q", got)
	}
	if got := brewActionLabel(false, false, true); got != "attention" {
		t.Fatalf("brewActionLabel(disabled) = %q", got)
	}
	if got := brewActionLabel(false, false, false); got != "current" {
		t.Fatalf("brewActionLabel(current) = %q", got)
	}

	info := parsePipShowOutput("Name: openai\nSummary: Official SDK\nLocation: /tmp/site\n---\nName: anthropic\nSummary: Anthropic SDK\nLocation: /tmp/site")
	if got := info["openai"].Summary; got != "Official SDK" {
		t.Fatalf("parsePipShowOutput(summary) = %q", got)
	}
	if got := info["anthropic"].Location; got != "/tmp/site" {
		t.Fatalf("parsePipShowOutput(location) = %q", got)
	}

	if got := formatFileDate(filepath.Join(t.TempDir(), "missing")); got != "" {
		t.Fatalf("formatFileDate(missing) = %q, want empty", got)
	}
	if got := joinDetails("", " one ", "", "two"); got != "one; two" {
		t.Fatalf("joinDetails() = %q, want %q", got, "one; two")
	}

	pkgs := []model.Package{{Name: "gh", Source: model.SourceHomebrew}}
	if got := enrichBrewFormulae(pkgs, "not-json"); !reflect.DeepEqual(got, pkgs) {
		t.Fatalf("enrichBrewFormulae(invalid) = %#v, want unchanged", got)
	}
	if got := enrichBrewFormulae(pkgs, `{"formulae":[{"name":"other","desc":"Other"}]}`); !reflect.DeepEqual(got, pkgs) {
		t.Fatalf("enrichBrewFormulae(missing) = %#v, want unchanged", got)
	}

	casks := []model.Package{{Name: "ghostty", Source: model.SourceHomebrewCask}}
	if got := enrichBrewCasks(casks, "not-json"); !reflect.DeepEqual(got, casks) {
		t.Fatalf("enrichBrewCasks(invalid) = %#v, want unchanged", got)
	}
	if got := enrichBrewCasks(casks, `{"casks":[{"token":"other","desc":"Other"}]}`); !reflect.DeepEqual(got, casks) {
		t.Fatalf("enrichBrewCasks(missing) = %#v, want unchanged", got)
	}

	if got := findPipUpdatedAt("", "openai"); got != "" {
		t.Fatalf("findPipUpdatedAt(empty) = %q, want empty", got)
	}
	if got := findPipUpdatedAt(t.TempDir(), "missing"); got != "" {
		t.Fatalf("findPipUpdatedAt(missing) = %q, want empty", got)
	}
}

func TestMetadataCommandsGracefullyFallback(t *testing.T) {
	baseFormulae := []model.Package{{Name: "gh", Source: model.SourceHomebrew}}
	if got := NewBrewCollector(stubRunner{}).addFormulaMetadata(context.Background(), nil); got != nil {
		t.Fatalf("addFormulaMetadata(nil) = %#v, want nil", got)
	}
	formulaRunner := stubRunner{errs: map[string]error{"brew info --json=v2 gh": context.DeadlineExceeded}}
	if got := NewBrewCollector(formulaRunner).addFormulaMetadata(context.Background(), append([]model.Package(nil), baseFormulae...)); !reflect.DeepEqual(got, baseFormulae) {
		t.Fatalf("addFormulaMetadata(err) = %#v, want unchanged", got)
	}

	baseCasks := []model.Package{{Name: "ghostty", Source: model.SourceHomebrewCask}}
	if got := NewBrewCollector(stubRunner{}).addCaskMetadata(context.Background(), nil); got != nil {
		t.Fatalf("addCaskMetadata(nil) = %#v, want nil", got)
	}
	caskErrRunner := stubRunner{errs: map[string]error{"brew info --json=v2 --cask ghostty": context.DeadlineExceeded}}
	if got := NewBrewCollector(caskErrRunner).addCaskMetadata(context.Background(), append([]model.Package(nil), baseCasks...)); !reflect.DeepEqual(got, baseCasks) {
		t.Fatalf("addCaskMetadata(err) = %#v, want unchanged", got)
	}
	caskRunner := stubRunner{results: map[string]Result{"brew info --json=v2 --cask ghostty": {ExitCode: 1}}}
	if got := NewBrewCollector(caskRunner).addCaskMetadata(context.Background(), append([]model.Package(nil), baseCasks...)); !reflect.DeepEqual(got, baseCasks) {
		t.Fatalf("addCaskMetadata(exit) = %#v, want unchanged", got)
	}

	if got := NewPipCollector(stubRunner{}).addMetadata(context.Background(), "pip", nil); got != nil {
		t.Fatalf("addMetadata(nil) = %#v, want nil", got)
	}
	pipRunner := stubRunner{errs: map[string]error{"pip show ruff": context.DeadlineExceeded}}
	basePip := []model.Package{{Name: "ruff", Source: model.SourcePip}}
	if got := NewPipCollector(pipRunner).addMetadata(context.Background(), "pip", append([]model.Package(nil), basePip...)); !reflect.DeepEqual(got, basePip) {
		t.Fatalf("addMetadata(err) = %#v, want unchanged", got)
	}
	pipExitRunner := stubRunner{results: map[string]Result{"pip show ruff": {ExitCode: 1}}}
	if got := NewPipCollector(pipExitRunner).addMetadata(context.Background(), "pip", append([]model.Package(nil), basePip...)); !reflect.DeepEqual(got, basePip) {
		t.Fatalf("addMetadata(exit) = %#v, want unchanged", got)
	}

	brokenDir := t.TempDir()
	brokenPath := filepath.Join(brokenDir, "openai-1.0.0.dist-info")
	if err := os.Symlink(filepath.Join(brokenDir, "missing-target"), brokenPath); err == nil {
		if got := findPipUpdatedAt(brokenDir, "openai"); got != "" {
			t.Fatalf("findPipUpdatedAt(broken) = %q, want empty", got)
		}
	}
}

func TestPipCollectorAvailable(t *testing.T) {
	if !NewPipCollector(stubRunner{}).Available(context.Background()) {
		t.Fatal("Available() = false, want true")
	}
}

func TestPipCollectorMissingBinary(t *testing.T) {
	runner := stubRunner{paths: map[string]error{"pip": exec.ErrNotFound, "pip3": exec.ErrNotFound}}
	_, status, err := NewPipCollector(runner).Collect(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Collect() error = %v, want ErrUnavailable", err)
	}
	if status.State != model.CollectorStateMissing {
		t.Fatalf("status.State = %q", status.State)
	}
}

func TestPipCollectorInvalidJSON(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"pip list --not-required --format=json": {Stdout: `oops`},
		},
	}
	_, status, err := NewPipCollector(runner).Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() error = nil, want error")
	}
	if status.State != model.CollectorStateError {
		t.Fatalf("status.State = %q", status.State)
	}
}

func TestPipCollectorTimeout(t *testing.T) {
	runner := stubRunner{
		errs: map[string]error{
			"pip list --not-required --format=json": context.DeadlineExceeded,
		},
	}
	_, status, err := NewPipCollector(runner).Collect(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Collect() error = %v", err)
	}
	if status.State != model.CollectorStateTimeout {
		t.Fatalf("status.State = %q", status.State)
	}
}

func TestPipCollectorExitError(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"pip list --not-required --format=json": {ExitCode: 1, Stderr: "pip failed"},
		},
	}
	_, status, err := NewPipCollector(runner).Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() error = nil, want error")
	}
	if status.Details != "pip failed" {
		t.Fatalf("status.Details = %q, want %q", status.Details, "pip failed")
	}
}

func TestPipCollectorSortsPackagesByName(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"pip list --not-required --format=json": {Stdout: `[{"name":"zeta","version":"1.0.0"},{"name":"alpha","version":"2.0.0"}]`},
		},
	}
	pkgs, _, err := NewPipCollector(runner).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if pkgs[0].Name != "alpha" {
		t.Fatalf("packages[0].Name = %q, want %q", pkgs[0].Name, "alpha")
	}
}

func TestCollectAllSortsStatusesAndPackages(t *testing.T) {
	collectors := []Collector{
		staticCollector{
			source:   model.SourcePip,
			packages: []model.Package{{Name: "ruff", Source: model.SourcePip}},
			status:   model.CollectorStatus{Source: model.SourcePip, State: model.CollectorStateReady},
		},
		staticCollector{
			source:   model.SourceHomebrew,
			packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
			status:   model.CollectorStatus{Source: model.SourceHomebrew, State: model.CollectorStateReady},
		},
	}

	result := CollectAll(context.Background(), collectors)
	if got, want := result.Packages[0].Name, "gh"; got != want {
		t.Fatalf("result.Packages[0].Name = %q, want %q", got, want)
	}
	if got, want := result.Statuses[0].Source, model.SourceHomebrew; got != want {
		t.Fatalf("result.Statuses[0].Source = %q, want %q", got, want)
	}
}

func TestCollectAllOrdersSameNameBySource(t *testing.T) {
	result := CollectAll(context.Background(), []Collector{
		staticCollector{
			source:   model.SourcePip,
			packages: []model.Package{{Name: "gh", Source: model.SourcePip}},
			status:   model.CollectorStatus{Source: model.SourcePip, State: model.CollectorStateReady},
		},
		staticCollector{
			source:   model.SourceHomebrew,
			packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
			status:   model.CollectorStatus{Source: model.SourceHomebrew, State: model.CollectorStateReady},
		},
	})
	if got, want := result.Packages[0].Source, model.SourceHomebrew; got != want {
		t.Fatalf("result.Packages[0].Source = %q, want %q", got, want)
	}
}

func TestCollectAllBuildsFallbackErrorStatus(t *testing.T) {
	result := CollectAll(context.Background(), []Collector{
		staticCollector{
			source: model.SourceNPM,
			err:    errors.New("boom"),
		},
	})
	if result.Statuses[0].State != model.CollectorStateError {
		t.Fatalf("result.Statuses[0].State = %q", result.Statuses[0].State)
	}
	if result.Statuses[0].Details != "boom" {
		t.Fatalf("result.Statuses[0].Details = %q", result.Statuses[0].Details)
	}
}

type staticCollector struct {
	source   model.Source
	packages []model.Package
	status   model.CollectorStatus
	err      error
}

func (s staticCollector) Name() model.Source             { return s.source }
func (s staticCollector) Available(context.Context) bool { return true }
func (s staticCollector) Collect(context.Context) ([]model.Package, model.CollectorStatus, error) {
	return s.packages, s.status, s.err
}
