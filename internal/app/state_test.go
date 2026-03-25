package app

import (
	"reflect"
	"testing"

	"github.com/nad/pkgview/internal/model"
)

func TestVisiblePackagesFilterAndSortBySource(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "ruff", Source: model.SourcePip},
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "npm-check", Source: model.SourceNPM},
		},
		Filter:   "",
		SortMode: SortSource,
	}

	got := state.VisiblePackages()
	want := []model.Package{
		{Name: "gh", Source: model.SourceHomebrew},
		{Name: "npm-check", Source: model.SourceNPM},
		{Name: "ruff", Source: model.SourcePip},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("VisiblePackages() = %#v, want %#v", got, want)
	}
}

func TestVisiblePackagesSortsNamesCaseInsensitively(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "Zulu", Source: model.SourceHomebrew},
			{Name: "alpha", Source: model.SourceHomebrew},
		},
	}

	got := state.VisiblePackages()
	if got[0].Name != "alpha" {
		t.Fatalf("VisiblePackages()[0].Name = %q, want %q", got[0].Name, "alpha")
	}
}

func TestVisiblePackagesSortByVersion(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "gh", Version: "2.0.0", Source: model.SourceHomebrew},
			{Name: "bat", Version: "1.0.0", Source: model.SourceHomebrew},
		},
		SortMode: SortVersion,
	}

	got := state.VisiblePackages()
	if got[0].Name != "bat" {
		t.Fatalf("VisiblePackages()[0].Name = %q, want %q", got[0].Name, "bat")
	}
}

func TestVisiblePackagesSortByVersionPlacesEmptyVersionsLast(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "gh", Version: "", Source: model.SourceHomebrew},
			{Name: "bat", Version: "1.0.0", Source: model.SourceHomebrew},
		},
		SortMode: SortVersion,
	}

	got := state.VisiblePackages()
	if got[0].Name != "bat" {
		t.Fatalf("VisiblePackages()[0].Name = %q, want %q", got[0].Name, "bat")
	}
}

func TestVisiblePackagesSortByVersionUsesNameTiebreaker(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "Zulu", Version: "1.0.0", Source: model.SourceHomebrew},
			{Name: "alpha", Version: "1.0.0", Source: model.SourceHomebrew},
		},
		SortMode: SortVersion,
	}

	got := state.VisiblePackages()
	if got[0].Name != "alpha" {
		t.Fatalf("VisiblePackages()[0].Name = %q, want %q", got[0].Name, "alpha")
	}
}

func TestVisiblePackagesSortByVersionUsesNameTiebreakerWhenVersionsEmpty(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "Zulu", Version: "", Source: model.SourceHomebrew},
			{Name: "alpha", Version: "", Source: model.SourceHomebrew},
		},
		SortMode: SortVersion,
	}

	got := state.VisiblePackages()
	if got[0].Name != "alpha" {
		t.Fatalf("VisiblePackages()[0].Name = %q, want %q", got[0].Name, "alpha")
	}
}

func TestVisiblePackagesSortBySourceUsesNameTiebreaker(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "Zulu", Source: model.SourceHomebrew},
			{Name: "alpha", Source: model.SourceHomebrew},
		},
		SortMode: SortSource,
	}

	got := state.VisiblePackages()
	if got[0].Name != "alpha" {
		t.Fatalf("VisiblePackages()[0].Name = %q, want %q", got[0].Name, "alpha")
	}
}

func TestSummaryCountsUsesFilteredPackages(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "ruff", Source: model.SourcePip},
			{Name: "gh", Source: model.SourceHomebrew},
		},
		Filter: "gh",
	}
	counts := state.SummaryCounts()
	if counts[model.SourceHomebrew] != 1 {
		t.Fatalf("homebrew count = %d, want 1", counts[model.SourceHomebrew])
	}
	if counts[model.SourcePip] != 0 {
		t.Fatalf("pip count = %d, want 0", counts[model.SourcePip])
	}
}

func TestVisiblePackagesAppliesSourceFilter(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "gh", Source: model.SourceHomebrew},
			{Name: "ruff", Source: model.SourcePip},
		},
		SourceFilter: model.SourcePip,
	}

	got := state.VisiblePackages()
	if len(got) != 1 || got[0].Name != "ruff" {
		t.Fatalf("VisiblePackages() = %#v, want only ruff", got)
	}
}

func TestVisiblePackagesAppliesActionAndUpdatedFilters(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "gh", ActionRequired: "update", UpdatedAt: "2024-01-01", Source: model.SourceHomebrew},
			{Name: "ruff", ActionRequired: "current", UpdatedAt: "", Source: model.SourcePip},
			{Name: "ghostty", ActionRequired: "", UpdatedAt: "", Source: model.SourceHomebrewCask},
		},
		ActionFilter:  ActionFilterUpdate,
		UpdatedFilter: UpdatedFilterKnown,
	}

	got := state.VisiblePackages()
	if len(got) != 1 || got[0].Name != "gh" {
		t.Fatalf("VisiblePackages() = %#v, want only gh", got)
	}

	state.ActionFilter = ActionFilterUnknown
	state.UpdatedFilter = UpdatedFilterUnknown
	got = state.VisiblePackages()
	if len(got) != 1 || got[0].Name != "ghostty" {
		t.Fatalf("VisiblePackages() = %#v, want only ghostty", got)
	}
}

func TestVisiblePackagesSortByUpdated(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "older", UpdatedAt: "2024-01-01", Source: model.SourceHomebrew},
			{Name: "newer", UpdatedAt: "2024-03-01", Source: model.SourceHomebrew},
			{Name: "unknown", UpdatedAt: "", Source: model.SourceHomebrew},
		},
		SortMode: SortUpdated,
	}

	got := state.VisiblePackages()
	if got[0].Name != "newer" || got[1].Name != "older" || got[2].Name != "unknown" {
		t.Fatalf("VisiblePackages() = %#v, want newer/older/unknown", got)
	}
}

func TestVisiblePackagesSortByUpdatedUsesNameTiebreaker(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "Zulu", UpdatedAt: "2024-03-01", Source: model.SourceHomebrew},
			{Name: "alpha", UpdatedAt: "2024-03-01", Source: model.SourceHomebrew},
		},
		SortMode: SortUpdated,
	}

	got := state.VisiblePackages()
	if got[0].Name != "alpha" {
		t.Fatalf("VisiblePackages()[0].Name = %q, want %q", got[0].Name, "alpha")
	}
}

func TestVisiblePackagesSortByUpdatedUsesNameTiebreakerWhenDatesMissing(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "Zulu", UpdatedAt: "", Source: model.SourceHomebrew},
			{Name: "alpha", UpdatedAt: "", Source: model.SourceHomebrew},
		},
		SortMode: SortUpdated,
	}

	got := state.VisiblePackages()
	if got[0].Name != "alpha" {
		t.Fatalf("VisiblePackages()[0].Name = %q, want %q", got[0].Name, "alpha")
	}
}

func TestVisiblePackagesAppliesCurrentActionFilter(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "gh", ActionRequired: "current", Source: model.SourceHomebrew},
			{Name: "ruff", ActionRequired: "update", Source: model.SourcePip},
		},
		ActionFilter: ActionFilterCurrent,
	}

	got := state.VisiblePackages()
	if len(got) != 1 || got[0].Name != "gh" {
		t.Fatalf("VisiblePackages() = %#v, want only gh", got)
	}
}

func TestVisiblePackagesSortByNameWithCombinedFilters(t *testing.T) {
	state := State{
		Packages: []model.Package{
			{Name: "Zulu", ActionRequired: "update", UpdatedAt: "2024-02-01", Source: model.SourceHomebrew},
			{Name: "alpha", ActionRequired: "update", UpdatedAt: "2024-03-01", Source: model.SourceHomebrew},
			{Name: "alpha-pip", ActionRequired: "update", UpdatedAt: "2024-03-01", Source: model.SourcePip},
			{Name: "beta", ActionRequired: "current", UpdatedAt: "2024-03-01", Source: model.SourceHomebrew},
			{Name: "gamma", ActionRequired: "update", UpdatedAt: "", Source: model.SourceHomebrew},
		},
		Filter:        "",
		SourceFilter:  model.SourceHomebrew,
		ActionFilter:  ActionFilterUpdate,
		UpdatedFilter: UpdatedFilterKnown,
		SortMode:      SortName,
	}

	got := state.VisiblePackages()
	want := []string{"alpha", "Zulu"}
	if len(got) != len(want) {
		t.Fatalf("len(VisiblePackages()) = %d, want %d", len(got), len(want))
	}
	for index, name := range want {
		if got[index].Name != name {
			t.Fatalf("VisiblePackages()[%d].Name = %q, want %q", index, got[index].Name, name)
		}
	}
}

func TestMatchesActionFilter(t *testing.T) {
	pkg := model.Package{Name: "gh", ActionRequired: "attention"}
	if !matchesActionFilter(pkg, ActionFilterAttention) {
		t.Fatal("attention package should match attention filter")
	}
	pkg.ActionRequired = "current"
	if !matchesActionFilter(pkg, ActionFilterCurrent) {
		t.Fatal("current package should match current filter")
	}
	if !matchesActionFilter(model.Package{}, ActionFilterAll) {
		t.Fatal("all filter should match any package")
	}
}

func TestCycleSortWraps(t *testing.T) {
	state := State{}
	for i := 0; i < 5; i++ {
		state.CycleSort()
	}
	if state.SortMode != SortDefault {
		t.Fatalf("SortMode = %d, want %d", state.SortMode, SortDefault)
	}
}

func TestCycleSourceFilter(t *testing.T) {
	state := State{}
	state.CycleSourceFilter()
	if state.SourceFilter != model.SourceHomebrew {
		t.Fatalf("SourceFilter = %q, want %q", state.SourceFilter, model.SourceHomebrew)
	}
	for i := 0; i < 4; i++ {
		state.CycleSourceFilter()
	}
	if state.SourceFilter != "" {
		t.Fatalf("SourceFilter = %q, want all", state.SourceFilter)
	}
}

func TestTooSmall(t *testing.T) {
	state := State{Width: 79, Height: 24}
	if !state.TooSmall() {
		t.Fatal("TooSmall() = false, want true")
	}
}

func TestClampSelection(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		state := State{Selected: 3}
		state.ClampSelection()
		if state.Selected != 0 {
			t.Fatalf("Selected = %d, want 0", state.Selected)
		}
	})

	t.Run("negative", func(t *testing.T) {
		state := State{
			Packages: []model.Package{{Name: "gh", Source: model.SourceHomebrew}},
			Selected: -1,
		}
		state.ClampSelection()
		if state.Selected != 0 {
			t.Fatalf("Selected = %d, want 0", state.Selected)
		}
	})

	t.Run("upper bound", func(t *testing.T) {
		state := State{
			Packages: []model.Package{
				{Name: "gh", Source: model.SourceHomebrew},
				{Name: "bat", Source: model.SourceHomebrew},
			},
			Selected: 9,
		}
		state.ClampSelection()
		if state.Selected != 1 {
			t.Fatalf("Selected = %d, want 1", state.Selected)
		}
	})
}

func TestLessNameTiebreaker(t *testing.T) {
	if !lessName("Alpha", "alpha") {
		t.Fatal("lessName should use original string as tiebreaker when lowercase forms match")
	}
}
