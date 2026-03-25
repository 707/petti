package collectors

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/707/petti/internal/model"
)

type BrewCollector struct {
	runner Runner
}

func NewBrewCollector(runner Runner) BrewCollector {
	return BrewCollector{runner: runner}
}

func (c BrewCollector) Name() model.Source {
	return model.SourceHomebrew
}

func (c BrewCollector) Available(context.Context) bool {
	_, err := c.runner.LookPath("brew")
	return err == nil
}

func (c BrewCollector) Collect(ctx context.Context) ([]model.Package, model.CollectorStatus, error) {
	if !c.Available(ctx) {
		return nil, model.CollectorStatus{
			Source:  model.SourceHomebrew,
			Label:   "homebrew",
			State:   model.CollectorStateMissing,
			Details: "brew not found on PATH",
		}, ErrUnavailable
	}

	formulae, status, err := c.collectFormulae(ctx)
	if err != nil {
		return nil, status, err
	}

	casks, caskStatus, err := c.collectCasks(ctx)
	state := model.CollectorStateReady
	if err != nil && !errors.Is(err, ErrUnavailable) {
		state = caskStatus.State
	}

	all := append(formulae, casks...)
	return all, model.CollectorStatus{
		Source:  model.SourceHomebrew,
		Label:   "homebrew",
		State:   state,
		Details: joinDetails(status.Details, caskStatus.Details),
	}, nil
}

func (c BrewCollector) collectFormulae(ctx context.Context) ([]model.Package, model.CollectorStatus, error) {
	versionResult, err := c.runner.Run(ctx, "brew", "list", "--versions")
	if err != nil {
		return nil, timeoutStatus(model.SourceHomebrew, "homebrew", err), err
	}
	if versionResult.ExitCode != 0 {
		return nil, exitStatus(model.SourceHomebrew, "homebrew", versionResult), fmt.Errorf("brew list --versions failed")
	}

	packages := parseVersionPackages(versionResult.Stdout, model.SourceHomebrew)

	result, err := c.runner.Run(ctx, "brew", "leaves", "--installed-on-request")
	if err != nil {
		return nil, timeoutStatus(model.SourceHomebrew, "homebrew", err), err
	}
	if result.ExitCode != 0 {
		return nil, exitStatus(model.SourceHomebrew, "homebrew", result), fmt.Errorf("brew leaves failed")
	}

	leaves := map[string]struct{}{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		leaves[line] = struct{}{}
	}
	packages = c.addFormulaMetadata(ctx, packages)
	packages = applyBrewDependencySafety(packages, leaves)

	return packages, model.CollectorStatus{
		Source: model.SourceHomebrew,
		Label:  "homebrew",
		State:  model.CollectorStateReady,
	}, nil
}

func (c BrewCollector) collectCasks(ctx context.Context) ([]model.Package, model.CollectorStatus, error) {
	result, err := c.runner.Run(ctx, "brew", "list", "--cask", "--versions")
	if err != nil {
		return nil, timeoutStatus(model.SourceHomebrewCask, "homebrew-cask", err), err
	}
	if result.ExitCode != 0 {
		return nil, exitStatus(model.SourceHomebrewCask, "homebrew-cask", result), fmt.Errorf("brew list --cask --versions failed")
	}

	packages := parseVersionPackages(result.Stdout, model.SourceHomebrewCask)
	packages = c.addCaskMetadata(ctx, packages)
	packages = applyBrewDependencySafety(packages, nil)
	return packages, model.CollectorStatus{
		Source: model.SourceHomebrewCask,
		Label:  "homebrew-cask",
		State:  model.CollectorStateReady,
	}, nil
}

func (c BrewCollector) addFormulaMetadata(ctx context.Context, packages []model.Package) []model.Package {
	if len(packages) == 0 {
		return packages
	}

	args := []string{"info", "--json=v2"}
	for _, pkg := range packages {
		args = append(args, pkg.Name)
	}
	result, err := c.runner.Run(ctx, "brew", args...)
	if err != nil || result.ExitCode != 0 {
		return packages
	}
	return enrichBrewFormulae(packages, result.Stdout)
}

func (c BrewCollector) addCaskMetadata(ctx context.Context, packages []model.Package) []model.Package {
	if len(packages) == 0 {
		return packages
	}

	args := []string{"info", "--json=v2", "--cask"}
	for _, pkg := range packages {
		args = append(args, pkg.Name)
	}
	result, err := c.runner.Run(ctx, "brew", args...)
	if err != nil || result.ExitCode != 0 {
		return packages
	}
	return enrichBrewCasks(packages, result.Stdout)
}

func applyBrewDependencySafety(packages []model.Package, leaves map[string]struct{}) []model.Package {
	for index := range packages {
		if packages[index].Source == model.SourceHomebrewCask {
			packages[index].UsedBy = "-"
			continue
		}
		if _, ok := leaves[packages[index].Name]; ok {
			packages[index].UsedBy = "N"
			continue
		}
		packages[index].UsedBy = "Y"
	}
	return packages
}

func parseVersionLines(input string) map[string]string {
	versions := map[string]string{}
	for _, line := range strings.Split(input, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		versions[fields[0]] = strings.Join(fields[1:], " ")
	}
	return versions
}

func parseVersionPackages(input string, source model.Source) []model.Package {
	packages := []model.Package{}
	for _, line := range strings.Split(input, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		packages = append(packages, model.Package{
			Name:    fields[0],
			Version: strings.Join(fields[1:], " "),
			Source:  source,
		})
	}
	return packages
}

func timeoutStatus(source model.Source, label string, err error) model.CollectorStatus {
	return model.CollectorStatus{
		Source:  source,
		Label:   label,
		State:   model.CollectorStateTimeout,
		Details: err.Error(),
	}
}

func exitStatus(source model.Source, label string, result Result) model.CollectorStatus {
	details := strings.TrimSpace(result.Stderr)
	if details == "" {
		details = fmt.Sprintf("exit code %d", result.ExitCode)
	}
	return model.CollectorStatus{
		Source:  source,
		Label:   label,
		State:   model.CollectorStateError,
		Details: details,
	}
}

func joinDetails(parts ...string) string {
	var filtered []string
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			filtered = append(filtered, strings.TrimSpace(part))
		}
	}
	return strings.Join(filtered, "; ")
}
