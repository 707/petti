package collectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nad/pkgview/internal/model"
)

type brewFormulaInfoPayload struct {
	Formulae []brewFormulaInfo `json:"formulae"`
}

type brewFormulaInfo struct {
	Name       string              `json:"name"`
	Desc       string              `json:"desc"`
	Outdated   bool                `json:"outdated"`
	Deprecated bool                `json:"deprecated"`
	Disabled   bool                `json:"disabled"`
	Installed  []brewInstalledInfo `json:"installed"`
}

type brewInstalledInfo struct {
	Time int64 `json:"time"`
}

type brewCaskInfoPayload struct {
	Casks []brewCaskInfo `json:"casks"`
}

type brewCaskInfo struct {
	Token         string `json:"token"`
	Desc          string `json:"desc"`
	Outdated      bool   `json:"outdated"`
	Deprecated    bool   `json:"deprecated"`
	Disabled      bool   `json:"disabled"`
	InstalledTime int64  `json:"installed_time"`
}

type pipShowInfo struct {
	Summary  string
	Location string
}

func enrichBrewFormulae(packages []model.Package, stdout string) []model.Package {
	var payload brewFormulaInfoPayload
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		return packages
	}

	metadata := map[string]brewFormulaInfo{}
	for _, info := range payload.Formulae {
		metadata[info.Name] = info
	}
	for index := range packages {
		info, ok := metadata[packages[index].Name]
		if !ok {
			continue
		}
		packages[index].Description = strings.TrimSpace(info.Desc)
		packages[index].ActionRequired = brewActionLabel(info.Outdated, info.Deprecated, info.Disabled)
		if len(info.Installed) > 0 {
			packages[index].UpdatedAt = formatUnixDate(info.Installed[len(info.Installed)-1].Time)
		}
	}
	return packages
}

func enrichBrewCasks(packages []model.Package, stdout string) []model.Package {
	var payload brewCaskInfoPayload
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		return packages
	}

	metadata := map[string]brewCaskInfo{}
	for _, info := range payload.Casks {
		metadata[info.Token] = info
	}
	for index := range packages {
		info, ok := metadata[packages[index].Name]
		if !ok {
			continue
		}
		packages[index].Description = strings.TrimSpace(info.Desc)
		packages[index].ActionRequired = brewActionLabel(info.Outdated, info.Deprecated, info.Disabled)
		packages[index].UpdatedAt = formatUnixDate(info.InstalledTime)
	}
	return packages
}

func brewActionLabel(outdated, deprecated, disabled bool) string {
	switch {
	case deprecated, disabled:
		return "attention"
	case outdated:
		return "update"
	default:
		return "current"
	}
}

func formatUnixDate(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format("2006-01-02")
}

func formatFileDate(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return info.ModTime().UTC().Format("2006-01-02")
}

func parsePipShowOutput(output string) map[string]pipShowInfo {
	sections := strings.Split(strings.TrimSpace(output), "\n---")
	metadata := make(map[string]pipShowInfo, len(sections))
	for _, section := range sections {
		lines := strings.Split(strings.TrimSpace(section), "\n")
		var name string
		var info pipShowInfo
		for _, line := range lines {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "Name: "):
				name = strings.TrimSpace(strings.TrimPrefix(line, "Name: "))
			case strings.HasPrefix(line, "Summary: "):
				info.Summary = strings.TrimSpace(strings.TrimPrefix(line, "Summary: "))
			case strings.HasPrefix(line, "Location: "):
				info.Location = strings.TrimSpace(strings.TrimPrefix(line, "Location: "))
			}
		}
		if name != "" {
			metadata[strings.ToLower(name)] = info
		}
	}
	return metadata
}

func findPipUpdatedAt(location, name string) string {
	if strings.TrimSpace(location) == "" || strings.TrimSpace(name) == "" {
		return ""
	}

	for _, pattern := range pipDistributionPatterns(name) {
		matches, err := filepath.Glob(filepath.Join(location, pattern))
		if err != nil || len(matches) == 0 {
			continue
		}
		latest := ""
		latestTime := time.Time{}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if latest == "" || info.ModTime().After(latestTime) {
				latest = match
				latestTime = info.ModTime()
			}
		}
		if latest != "" {
			return latestTime.UTC().Format("2006-01-02")
		}
	}
	return ""
}

func pipDistributionPatterns(name string) []string {
	underscore := normalizeDistributionName(name, "_")
	hyphen := normalizeDistributionName(name, "-")
	return []string{underscore + "-*.dist-info", hyphen + "-*.dist-info"}
}

func normalizeDistributionName(name, separator string) string {
	value := strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer("-", separator, "_", separator, ".", separator, " ", separator)
	return replacer.Replace(value)
}
