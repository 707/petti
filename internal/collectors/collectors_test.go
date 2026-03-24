package collectors

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"testing"

	"github.com/nad/pkgview/internal/model"
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
		{Name: "gh", Version: "2.47.0", Source: model.SourceHomebrew},
		{Name: "fzf", Version: "0.54.0", Source: model.SourceHomebrew},
		{Name: "font-hack-nerd-font", Version: "3.4.0", Source: model.SourceHomebrewCask},
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
	want := []model.Package{{Name: "gh", Version: "2.47.0", Source: model.SourceHomebrew}}
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

func TestNPMCollectorStripsNPM(t *testing.T) {
	runner := stubRunner{
		results: map[string]Result{
			"npm list -g --depth=0 --json": {Stdout: `{"dependencies":{"npm":{"version":"10.0.0"},"typescript":{"version":"5.4.3"}}}`},
		},
	}
	pkgs, _, err := NewNPMCollector(runner).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	want := []model.Package{{Name: "typescript", Version: "5.4.3", Source: model.SourceNPM}}
	if !reflect.DeepEqual(pkgs, want) {
		t.Fatalf("Collect() = %#v, want %#v", pkgs, want)
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
			"npm list -g --depth=0 --json": {Stdout: "{not-json"},
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
			"npm list -g --depth=0 --json": context.DeadlineExceeded,
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
			"npm list -g --depth=0 --json": {ExitCode: 1, Stderr: "npm failed"},
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
	runner := stubRunner{
		paths: map[string]error{"pip": exec.ErrNotFound},
		results: map[string]Result{
			"pip3 list --not-required --format=json": {Stdout: `[{"name":"ruff","version":"0.3.4"}]`},
		},
	}
	pkgs, _, err := NewPipCollector(runner).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	want := []model.Package{{Name: "ruff", Version: "0.3.4", Source: model.SourcePip}}
	if !reflect.DeepEqual(pkgs, want) {
		t.Fatalf("Collect() = %#v, want %#v", pkgs, want)
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
