package model

import "strings"

type Source string

const (
	SourceHomebrew     Source = "homebrew"
	SourceHomebrewCask Source = "homebrew-cask"
	SourceNPM          Source = "npm"
	SourcePip          Source = "pip"
)

type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  Source `json:"source"`
}

func (p Package) Key() string {
	return string(p.Source) + ":" + strings.ToLower(p.Name)
}

func SourceOrder(source Source) int {
	switch source {
	case SourceHomebrew:
		return 0
	case SourceHomebrewCask:
		return 1
	case SourceNPM:
		return 2
	case SourcePip:
		return 3
	default:
		return 99
	}
}
