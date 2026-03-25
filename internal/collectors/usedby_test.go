package collectors

import (
	"testing"

	"github.com/nad/pkgview/internal/model"
)

func TestEnrichBrewMetadataSetsDependencySafetyDefaults(t *testing.T) {
	formulae := enrichBrewFormulae([]model.Package{{Name: "gh", Source: model.SourceHomebrew}}, `{"formulae":[{"name":"gh","desc":"GitHub CLI","outdated":false,"installed":[{"time":1704067200}]}],"casks":[]}`)
	if formulae[0].UsedBy != "-" {
		t.Fatalf("UsedBy = %q, want %q", formulae[0].UsedBy, "-")
	}

	casks := enrichBrewCasks([]model.Package{{Name: "ghostty", Source: model.SourceHomebrewCask}}, `{"formulae":[],"casks":[{"token":"ghostty","desc":"Terminal","outdated":false,"installed_time":1709251200}]}`)
	if casks[0].UsedBy != "-" {
		t.Fatalf("UsedBy = %q, want %q", casks[0].UsedBy, "-")
	}
}
