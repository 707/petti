package collectors

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/707/petti/internal/model"
)

func TestPackageInspectorInspectBrew(t *testing.T) {
	inspector := NewPackageInspector(stubRunner{
		results: map[string]Result{
			"brew info --json=v2 gh":             {Stdout: `{"formulae":[{"name":"gh","homepage":"https://cli.github.com","dependencies":["git","openssl"],"installed":[{"installed_size":"12 MB"}]}],"casks":[]}`},
			"brew uses --installed gh":           {Stdout: "hub\n"},
			"brew info --json=v2 --cask ghostty": {Stdout: `{"formulae":[],"casks":[{"token":"ghostty","homepage":"https://ghostty.org","depends_on":{"formula":["fontconfig","libpng"]}}]}`},
			"brew uses --installed ghostty":      {Stdout: ""},
		},
	})

	details, err := inspector.Inspect(context.Background(), model.Package{Name: "gh", Source: model.SourceHomebrew})
	if err != nil {
		t.Fatalf("Inspect(brew) error = %v", err)
	}
	if details.Homepage != "https://cli.github.com" || details.Size != "12 MB" {
		t.Fatalf("brew details = %#v", details)
	}
	if !reflect.DeepEqual(details.Dependencies, []string{"git", "openssl"}) {
		t.Fatalf("Dependencies = %#v", details.Dependencies)
	}
	if !reflect.DeepEqual(details.Dependents, []string{"hub"}) {
		t.Fatalf("Dependents = %#v", details.Dependents)
	}

	details, err = inspector.Inspect(context.Background(), model.Package{Name: "ghostty", Source: model.SourceHomebrewCask})
	if err != nil {
		t.Fatalf("Inspect(cask) error = %v", err)
	}
	if details.Homepage != "https://ghostty.org" {
		t.Fatalf("Homepage = %q", details.Homepage)
	}
	if !reflect.DeepEqual(details.Dependencies, []string{"fontconfig", "libpng"}) {
		t.Fatalf("Dependencies = %#v", details.Dependencies)
	}
	if len(details.Dependents) != 0 {
		t.Fatalf("Dependents = %#v, want empty", details.Dependents)
	}
}

func TestPackageInspectorInspectNPMAndPip(t *testing.T) {
	inspector := NewPackageInspector(stubRunner{
		results: map[string]Result{
			"npm view typescript --json": {Stdout: `{"homepage":"https://www.typescriptlang.org","repository":{"url":"https://github.com/microsoft/TypeScript"},"dependencies":{"chalk":"^5.0.0","minimist":"^1.2.8"}}`},
			"pip show ruff":              {Stdout: "Name: ruff\nHome-page: https://docs.astral.sh/ruff\nRequires: click, tomli\nLocation: /tmp/site\nRequired-by: "},
		},
	})

	npmDetails, err := inspector.Inspect(context.Background(), model.Package{Name: "typescript", Source: model.SourceNPM})
	if err != nil {
		t.Fatalf("Inspect(npm) error = %v", err)
	}
	if npmDetails.Homepage != "https://www.typescriptlang.org" || npmDetails.Repository != "https://github.com/microsoft/TypeScript" {
		t.Fatalf("npm details = %#v", npmDetails)
	}
	if !reflect.DeepEqual(npmDetails.Dependencies, []string{"chalk", "minimist"}) {
		t.Fatalf("npm dependencies = %#v", npmDetails.Dependencies)
	}

	pipDetails, err := inspector.Inspect(context.Background(), model.Package{Name: "ruff", Source: model.SourcePip})
	if err != nil {
		t.Fatalf("Inspect(pip) error = %v", err)
	}
	if pipDetails.Homepage != "https://docs.astral.sh/ruff" || pipDetails.Location != "/tmp/site" {
		t.Fatalf("pip details = %#v", pipDetails)
	}
	if !reflect.DeepEqual(pipDetails.Dependencies, []string{"click", "tomli"}) {
		t.Fatalf("pip dependencies = %#v", pipDetails.Dependencies)
	}
}

func TestPackageInspectorFallbacksAndErrors(t *testing.T) {
	inspector := NewPackageInspector(stubRunner{
		paths: map[string]error{"pip": ErrUnavailable, "pip3": ErrUnavailable},
		errs:  map[string]error{"brew info --json=v2 gh": context.DeadlineExceeded},
		results: map[string]Result{
			"npm view broken --json": {Stdout: `{bad json`},
		},
	})

	if _, err := inspector.Inspect(context.Background(), model.Package{Name: "gh", Source: model.SourceHomebrew}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Inspect(brew error) = %v, want deadline exceeded", err)
	}
	if _, err := inspector.Inspect(context.Background(), model.Package{Name: "broken", Source: model.SourceNPM}); err == nil {
		t.Fatal("Inspect(npm invalid json) error = nil, want error")
	}
	if _, err := inspector.Inspect(context.Background(), model.Package{Name: "ruff", Source: model.SourcePip}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Inspect(pip unavailable) = %v, want ErrUnavailable", err)
	}

	parts := splitCSVDetails(" a, b ,, c ")
	if !reflect.DeepEqual(parts, []string{"a", "b", "c"}) {
		t.Fatalf("splitCSVDetails() = %#v", parts)
	}
	if got := normalizeDetailValue("   "); got != "" {
		t.Fatalf("normalizeDetailValue() = %q, want empty", got)
	}
}

func TestPackageInspectorCommandExitBranches(t *testing.T) {
	inspector := NewPackageInspector(stubRunner{
		results: map[string]Result{
			"brew info --json=v2 gh": {ExitCode: 1},
			"npm view broken --json": {ExitCode: 1},
			"pip show ruff":          {ExitCode: 1},
		},
	})

	if _, err := inspector.inspectBrew(context.Background(), model.Package{Name: "gh", Source: model.SourceHomebrew}); err == nil {
		t.Fatal("inspectBrew(exit) error = nil, want error")
	}
	if _, err := inspector.inspectNPM(context.Background(), model.Package{Name: "broken", Source: model.SourceNPM}); err == nil {
		t.Fatal("inspectNPM(exit) error = nil, want error")
	}

	pipInspector := NewPackageInspector(stubRunner{
		results: map[string]Result{"pip show ruff": {ExitCode: 1}},
	})
	if _, err := pipInspector.inspectPip(context.Background(), model.Package{Name: "ruff", Source: model.SourcePip}); err == nil {
		t.Fatal("inspectPip(exit) error = nil, want error")
	}
}

func TestPackageInspectorCommandErrorBranches(t *testing.T) {
	inspector := NewPackageInspector(stubRunner{
		errs: map[string]error{
			"brew info --json=v2 gh":   context.DeadlineExceeded,
			"npm view broken --json":   context.DeadlineExceeded,
			"pip show ruff":            context.DeadlineExceeded,
			"brew uses --installed gh": context.DeadlineExceeded,
		},
		results: map[string]Result{
			"brew info --json=v2 gh": {Stdout: `{"formulae":[{"name":"gh","homepage":"https://cli.github.com"}]}`},
		},
	})

	if _, err := inspector.inspectNPM(context.Background(), model.Package{Name: "broken", Source: model.SourceNPM}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("inspectNPM(err) = %v, want deadline exceeded", err)
	}

	pipInspector := NewPackageInspector(stubRunner{
		errs: map[string]error{"pip show ruff": context.DeadlineExceeded},
	})
	if _, err := pipInspector.inspectPip(context.Background(), model.Package{Name: "ruff", Source: model.SourcePip}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("inspectPip(err) = %v, want deadline exceeded", err)
	}

	usesFallback := NewPackageInspector(stubRunner{
		results: map[string]Result{
			"brew info --json=v2 gh": {Stdout: `{"formulae":[{"name":"gh","homepage":"https://cli.github.com"}]}`},
		},
		errs: map[string]error{
			"brew uses --installed gh": context.DeadlineExceeded,
		},
	})
	details, err := usesFallback.inspectBrew(context.Background(), model.Package{Name: "gh", Source: model.SourceHomebrew})
	if err != nil {
		t.Fatalf("inspectBrew(uses fallback) error = %v", err)
	}
	if details.Homepage != "https://cli.github.com" || len(details.Dependents) != 0 {
		t.Fatalf("details = %#v", details)
	}

	usesExit := NewPackageInspector(stubRunner{
		results: map[string]Result{
			"brew info --json=v2 gh":   {Stdout: `{"formulae":[{"name":"gh","homepage":"https://cli.github.com"}]}`},
			"brew uses --installed gh": {ExitCode: 1},
		},
	})
	details, err = usesExit.inspectBrew(context.Background(), model.Package{Name: "gh", Source: model.SourceHomebrew})
	if err != nil {
		t.Fatalf("inspectBrew(uses exit) error = %v", err)
	}
	if len(details.Dependents) != 0 {
		t.Fatalf("Dependents = %#v, want empty", details.Dependents)
	}

	parseFail := NewPackageInspector(stubRunner{
		results: map[string]Result{
			"brew info --json=v2 gh": {Stdout: `{bad json`},
		},
	})
	if _, err := parseFail.inspectBrew(context.Background(), model.Package{Name: "gh", Source: model.SourceHomebrew}); err == nil {
		t.Fatal("inspectBrew(parse error) error = nil, want error")
	}
}

func TestParsePackageDetailHelpersCoverage(t *testing.T) {
	if _, err := parseBrewPackageDetails("not-json", model.SourceHomebrew); err == nil {
		t.Fatal("parseBrewPackageDetails(invalid) error = nil, want error")
	}
	details, err := parseBrewPackageDetails(`{"formulae":[],"casks":[]}`, model.SourceHomebrew)
	if err != nil || !reflect.DeepEqual(details, model.PackageDetails{}) {
		t.Fatalf("parseBrewPackageDetails(empty formulae) = %#v, %v", details, err)
	}
	details, err = parseBrewPackageDetails(`{"formulae":[],"casks":[]}`, model.SourceHomebrewCask)
	if err != nil || !reflect.DeepEqual(details, model.PackageDetails{}) {
		t.Fatalf("parseBrewPackageDetails(empty casks) = %#v, %v", details, err)
	}

	npmDetails, err := parseNPMPackageDetails(`{"repository":"https://github.com/npm/cli"}`)
	if err != nil {
		t.Fatalf("parseNPMPackageDetails(string repo) error = %v", err)
	}
	if npmDetails.Repository != "https://github.com/npm/cli" {
		t.Fatalf("Repository = %q", npmDetails.Repository)
	}

	pipDetails := parsePipPackageDetails("Name: x\nHome-page: https://x\nRequires: \nRequired-by: a, b")
	if !reflect.DeepEqual(pipDetails.Dependents, []string{"a", "b"}) {
		t.Fatalf("Dependents = %#v", pipDetails.Dependents)
	}
	pipDetails = parsePipPackageDetails("garbage\nLocation: /tmp/site")
	if pipDetails.Location != "/tmp/site" {
		t.Fatalf("Location = %q", pipDetails.Location)
	}
	if got := parsePipPackageDetails(""); got.Homepage != "" || got.Location != "" || len(got.Dependencies) != 0 || len(got.Dependents) != 0 {
		t.Fatalf("parsePipPackageDetails(empty) = %#v", got)
	}
}
