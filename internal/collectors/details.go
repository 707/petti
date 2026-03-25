package collectors

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/nad/pkgview/internal/model"
)

type PackageInspector struct {
	runner Runner
}

func NewPackageInspector(runner Runner) PackageInspector {
	return PackageInspector{runner: runner}
}

func (i PackageInspector) Inspect(ctx context.Context, pkg model.Package) (model.PackageDetails, error) {
	switch pkg.Source {
	case model.SourceHomebrew, model.SourceHomebrewCask:
		return i.inspectBrew(ctx, pkg)
	case model.SourceNPM:
		return i.inspectNPM(ctx, pkg)
	default:
		return i.inspectPip(ctx, pkg)
	}
}

func (i PackageInspector) inspectBrew(ctx context.Context, pkg model.Package) (model.PackageDetails, error) {
	args := []string{"info", "--json=v2"}
	if pkg.Source == model.SourceHomebrewCask {
		args = append(args, "--cask")
	}
	args = append(args, pkg.Name)
	result, err := i.runner.Run(ctx, "brew", args...)
	if err != nil {
		return model.PackageDetails{}, err
	}
	if result.ExitCode != 0 {
		return model.PackageDetails{}, errors.New("brew info failed")
	}

	details, err := parseBrewPackageDetails(result.Stdout, pkg.Source)
	if err != nil {
		return model.PackageDetails{}, err
	}

	usesResult, usesErr := i.runner.Run(ctx, "brew", "uses", "--installed", pkg.Name)
	if usesErr == nil && usesResult.ExitCode == 0 {
		details.Dependents = splitLineDetails(usesResult.Stdout)
	}
	return details, nil
}

func (i PackageInspector) inspectNPM(ctx context.Context, pkg model.Package) (model.PackageDetails, error) {
	result, err := i.runner.Run(ctx, "npm", "view", pkg.Name, "--json")
	if err != nil {
		return model.PackageDetails{}, err
	}
	if result.ExitCode != 0 {
		return model.PackageDetails{}, errors.New("npm view failed")
	}
	return parseNPMPackageDetails(result.Stdout)
}

func (i PackageInspector) inspectPip(ctx context.Context, pkg model.Package) (model.PackageDetails, error) {
	command, err := PipCollector{runner: i.runner}.command()
	if err != nil {
		return model.PackageDetails{}, err
	}
	result, err := i.runner.Run(ctx, command, "show", pkg.Name)
	if err != nil {
		return model.PackageDetails{}, err
	}
	if result.ExitCode != 0 {
		return model.PackageDetails{}, errors.New("pip show failed")
	}
	return parsePipPackageDetails(result.Stdout), nil
}

func parseBrewPackageDetails(stdout string, source model.Source) (model.PackageDetails, error) {
	type installedInfo struct {
		InstalledSize string `json:"installed_size"`
	}
	type formulaInfo struct {
		Name         string          `json:"name"`
		Homepage     string          `json:"homepage"`
		Dependencies []string        `json:"dependencies"`
		Installed    []installedInfo `json:"installed"`
	}
	type caskDepends struct {
		Formula []string `json:"formula"`
	}
	type caskInfo struct {
		Token     string      `json:"token"`
		Homepage  string      `json:"homepage"`
		DependsOn caskDepends `json:"depends_on"`
	}
	type payload struct {
		Formulae []formulaInfo `json:"formulae"`
		Casks    []caskInfo    `json:"casks"`
	}

	var out payload
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		return model.PackageDetails{}, err
	}

	if source == model.SourceHomebrewCask {
		if len(out.Casks) == 0 {
			return model.PackageDetails{}, nil
		}
		return model.PackageDetails{
			Homepage:     normalizeDetailValue(out.Casks[0].Homepage),
			Dependencies: out.Casks[0].DependsOn.Formula,
		}, nil
	}
	if len(out.Formulae) == 0 {
		return model.PackageDetails{}, nil
	}
	details := model.PackageDetails{
		Homepage:     normalizeDetailValue(out.Formulae[0].Homepage),
		Dependencies: out.Formulae[0].Dependencies,
	}
	if len(out.Formulae[0].Installed) > 0 {
		details.Size = normalizeDetailValue(out.Formulae[0].Installed[0].InstalledSize)
	}
	return details, nil
}

func parseNPMPackageDetails(stdout string) (model.PackageDetails, error) {
	type repositoryObject struct {
		URL string `json:"url"`
	}
	type payload struct {
		Homepage     string            `json:"homepage"`
		Repository   json.RawMessage   `json:"repository"`
		Dependencies map[string]string `json:"dependencies"`
	}

	var out payload
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		return model.PackageDetails{}, err
	}

	details := model.PackageDetails{Homepage: normalizeDetailValue(out.Homepage)}
	if len(out.Repository) > 0 {
		var repository string
		if err := json.Unmarshal(out.Repository, &repository); err == nil {
			details.Repository = normalizeDetailValue(repository)
		} else {
			var object repositoryObject
			if err := json.Unmarshal(out.Repository, &object); err == nil {
				details.Repository = normalizeDetailValue(object.URL)
			}
		}
	}
	for name := range out.Dependencies {
		details.Dependencies = append(details.Dependencies, name)
	}
	sort.Strings(details.Dependencies)
	return details, nil
}

func parsePipPackageDetails(stdout string) model.PackageDetails {
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		fields[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return model.PackageDetails{
		Homepage:     normalizeDetailValue(fields["Home-page"]),
		Location:     normalizeDetailValue(fields["Location"]),
		Dependencies: splitCSVDetails(fields["Requires"]),
		Dependents:   splitCSVDetails(fields["Required-by"]),
	}
}

func splitCSVDetails(value string) []string {
	parts := strings.Split(value, ",")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeDetailValue(part)
		if part != "" {
			normalized = append(normalized, part)
		}
	}
	return normalized
}

func splitLineDetails(value string) []string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		line = normalizeDetailValue(line)
		if line != "" {
			normalized = append(normalized, line)
		}
	}
	return normalized
}

func normalizeDetailValue(value string) string {
	return strings.TrimSpace(value)
}
