package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/nad/pkgview/internal/model"
)

type NPMCollector struct {
	runner Runner
}

func NewNPMCollector(runner Runner) NPMCollector {
	return NPMCollector{runner: runner}
}

func (c NPMCollector) Name() model.Source {
	return model.SourceNPM
}

func (c NPMCollector) Available(context.Context) bool {
	_, err := c.runner.LookPath("npm")
	return err == nil
}

func (c NPMCollector) Collect(ctx context.Context) ([]model.Package, model.CollectorStatus, error) {
	if !c.Available(ctx) {
		return nil, model.CollectorStatus{
			Source:  model.SourceNPM,
			Label:   "npm",
			State:   model.CollectorStateMissing,
			Details: "npm not found on PATH",
		}, ErrUnavailable
	}

	result, err := c.runner.Run(ctx, "npm", "list", "-g", "--depth=0", "--json", "-l")
	if err != nil {
		return nil, timeoutStatus(model.SourceNPM, "npm", err), err
	}
	if result.ExitCode != 0 {
		return nil, exitStatus(model.SourceNPM, "npm", result), fmt.Errorf("npm list failed")
	}
	return c.collectDetailed(result.Stdout)
}

func (c NPMCollector) collectDetailed(stdout string) ([]model.Package, model.CollectorStatus, error) {
	type dependency struct {
		Version     string `json:"version"`
		Description string `json:"description"`
		Path        string `json:"path"`
	}
	type payload struct {
		Dependencies map[string]dependency `json:"dependencies"`
	}

	var out payload
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		return nil, model.CollectorStatus{
			Source:  model.SourceNPM,
			Label:   "npm",
			State:   model.CollectorStateError,
			Details: err.Error(),
		}, err
	}

	names := make([]string, 0, len(out.Dependencies))
	for name := range out.Dependencies {
		if name == "npm" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	packages := make([]model.Package, 0, len(names))
	for _, name := range names {
		dependency := out.Dependencies[name]
		packages = append(packages, model.Package{
			Name:        name,
			Version:     dependency.Version,
			Source:      model.SourceNPM,
			Description: dependency.Description,
			UpdatedAt:   formatFileDate(dependency.Path),
		})
	}

	return packages, model.CollectorStatus{
		Source: model.SourceNPM,
		Label:  "npm",
		State:  model.CollectorStateReady,
	}, nil
}
