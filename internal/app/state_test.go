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

func TestCycleSortWraps(t *testing.T) {
	state := State{}
	for i := 0; i < 4; i++ {
		state.CycleSort()
	}
	if state.SortMode != SortDefault {
		t.Fatalf("SortMode = %d, want %d", state.SortMode, SortDefault)
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
