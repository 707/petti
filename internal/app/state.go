package app

import (
	"sort"
	"strings"

	"github.com/707/petti/internal/model"
)

type SortMode int

const (
	SortDefault SortMode = iota
	SortName
	SortVersion
	SortSource
	SortUpdated
)

type ActionFilter string

const (
	ActionFilterAll       ActionFilter = ""
	ActionFilterUpdate    ActionFilter = "update"
	ActionFilterAttention ActionFilter = "attention"
	ActionFilterCurrent   ActionFilter = "current"
	ActionFilterUnknown   ActionFilter = "unknown"
)

type UpdatedFilter string

const (
	UpdatedFilterAll     UpdatedFilter = ""
	UpdatedFilterKnown   UpdatedFilter = "known"
	UpdatedFilterUnknown UpdatedFilter = "unknown"
)

type State struct {
	Packages      []model.Package
	Statuses      []model.CollectorStatus
	Filter        string
	SourceFilter  model.Source
	ActionFilter  ActionFilter
	UpdatedFilter UpdatedFilter
	SortMode      SortMode
	Selected      int
	ShowHelp      bool
	Width         int
	Height        int
	IsLoading     bool
}

func (s State) VisiblePackages() []model.Package {
	filtered := make([]model.Package, 0, len(s.Packages))
	needle := strings.ToLower(strings.TrimSpace(s.Filter))
	for _, pkg := range s.Packages {
		if s.SourceFilter != "" && pkg.Source != s.SourceFilter {
			continue
		}
		if !matchesActionFilter(pkg, s.ActionFilter) {
			continue
		}
		if !matchesUpdatedFilter(pkg, s.UpdatedFilter) {
			continue
		}
		if needle == "" || strings.Contains(strings.ToLower(pkg.Name), needle) {
			filtered = append(filtered, pkg)
		}
	}

	switch s.SortMode {
	case SortUpdated:
		sort.SliceStable(filtered, func(i, j int) bool {
			if filtered[i].UpdatedAt == "" || filtered[j].UpdatedAt == "" {
				if filtered[i].UpdatedAt == filtered[j].UpdatedAt {
					return lessName(filtered[i].Name, filtered[j].Name)
				}
				return filtered[j].UpdatedAt == ""
			}
			if filtered[i].UpdatedAt == filtered[j].UpdatedAt {
				return lessName(filtered[i].Name, filtered[j].Name)
			}
			return filtered[i].UpdatedAt > filtered[j].UpdatedAt
		})
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
	s.SortMode = (s.SortMode + 1) % 5
	s.ClampSelection()
}

func (s *State) CycleSourceFilter() {
	switch s.SourceFilter {
	case "":
		s.SourceFilter = model.SourceHomebrew
	case model.SourceHomebrew:
		s.SourceFilter = model.SourceHomebrewCask
	case model.SourceHomebrewCask:
		s.SourceFilter = model.SourceNPM
	case model.SourceNPM:
		s.SourceFilter = model.SourcePip
	default:
		s.SourceFilter = ""
	}
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

func matchesActionFilter(pkg model.Package, filter ActionFilter) bool {
	switch filter {
	case ActionFilterUpdate:
		return pkg.ActionRequired == string(ActionFilterUpdate)
	case ActionFilterAttention:
		return pkg.ActionRequired == string(ActionFilterAttention)
	case ActionFilterCurrent:
		return pkg.ActionRequired == string(ActionFilterCurrent)
	case ActionFilterUnknown:
		return strings.TrimSpace(pkg.ActionRequired) == ""
	default:
		return true
	}
}

func matchesUpdatedFilter(pkg model.Package, filter UpdatedFilter) bool {
	switch filter {
	case UpdatedFilterKnown:
		return strings.TrimSpace(pkg.UpdatedAt) != ""
	case UpdatedFilterUnknown:
		return strings.TrimSpace(pkg.UpdatedAt) == ""
	default:
		return true
	}
}
