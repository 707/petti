package model

import "testing"

func TestPackageKey(t *testing.T) {
	pkg := Package{Name: "Gh", Source: SourceHomebrew}
	if got, want := pkg.Key(), "homebrew:gh"; got != want {
		t.Fatalf("Key() = %q, want %q", got, want)
	}
}

func TestSourceOrder(t *testing.T) {
	cases := []struct {
		source Source
		want   int
	}{
		{SourceHomebrew, 0},
		{SourceHomebrewCask, 1},
		{SourceNPM, 2},
		{SourcePip, 3},
		{Source("other"), 99},
	}

	for _, tc := range cases {
		if got := SourceOrder(tc.source); got != tc.want {
			t.Fatalf("SourceOrder(%q) = %d, want %d", tc.source, got, tc.want)
		}
	}
}
