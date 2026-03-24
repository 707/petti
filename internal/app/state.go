package app

import (
	"sort"
	"strings"

	"github.com/nad/pkgview/internal/model"
)

type SortMode int

const (
	SortDefault SortMode = iota
	SortName
	SortVersion
	SortSource
)

type State struct {
	Packages  []model.Package
	Statuses  []model.CollectorStatus
	Filter    string
	SortMode  SortMode
	Selected  int
	ShowHelp  bool
	Width     int
	Height    int
	IsLoading bool
}

func (s State) VisiblePackages() []model.Package {
	filtered := make([]model.Package, 0, len(s.Packages))
	needle := strings.ToLower(strings.TrimSpace(s.Filter))
	for _, pkg := range s.Packages {
		if needle == "" || strings.Contains(strings.ToLower(pkg.Name), needle) {
			filtered = append(filtered, pkg)
		}
	}

	switch s.SortMode {
	case SortVersion:
		sort.SliceStable(filtered, func(i, j int) bool {
			if filtered[i].Version == "" || filtered[j].Version == "" {
				if filtered[i].Version == filtered[j].Version {
					return lessName(filtered[i].Name, filtered[j].Name)
				}
				return filtered[j].Version == ""
			}
			if filtered[i].Version == filtered[j].Version {
				return lessName(filtered[i].Name, filtered[j].Name)
			}
			return filtered[i].Version < filtered[j].Version
		})
	case SortSource:
		sort.SliceStable(filtered, func(i, j int) bool {
			if filtered[i].Source == filtered[j].Source {
				return lessName(filtered[i].Name, filtered[j].Name)
			}
			return model.SourceOrder(filtered[i].Source) < model.SourceOrder(filtered[j].Source)
		})
	default:
		sort.SliceStable(filtered, func(i, j int) bool {
			return lessName(filtered[i].Name, filtered[j].Name)
		})
	}

	return filtered
}

func (s State) SummaryCounts() map[model.Source]int {
	counts := map[model.Source]int{}
	for _, pkg := range s.VisiblePackages() {
		counts[pkg.Source]++
	}
	return counts
}

func (s *State) ClampSelection() {
	visible := s.VisiblePackages()
	if len(visible) == 0 {
		s.Selected = 0
		return
	}
	if s.Selected < 0 {
		s.Selected = 0
	}
	if s.Selected >= len(visible) {
		s.Selected = len(visible) - 1
	}
}

func (s *State) CycleSort() {
	s.SortMode = (s.SortMode + 1) % 4
	s.ClampSelection()
}

func (s State) TooSmall() bool {
	return s.Width > 0 && s.Height > 0 && (s.Width < 80 || s.Height < 24)
}

func lessName(left, right string) bool {
	leftLower := strings.ToLower(left)
	rightLower := strings.ToLower(right)
	if leftLower == rightLower {
		return left < right
	}
	return leftLower < rightLower
}
