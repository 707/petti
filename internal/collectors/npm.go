package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/707/petti/internal/model"
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
	return c.collectDetailed(ctx, result.Stdout)
}

func (c NPMCollector) collectDetailed(ctx context.Context, stdout string) ([]model.Package, model.CollectorStatus, error) {
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
			UsedBy:      "-",
		})
	}
	packages = c.applyDependencySafety(ctx, packages)

	return packages, model.CollectorStatus{
		Source: model.SourceNPM,
		Label:  "npm",
		State:  model.CollectorStateReady,
	}, nil
}

func (c NPMCollector) applyDependencySafety(ctx context.Context, packages []model.Package) []model.Package {
	if len(packages) == 0 {
		return packages
	}
	result, err := c.runner.Run(ctx, "npm", "ls", "-g", "--all", "--json")
	if err != nil || result.ExitCode != 0 {
		return packages
	}
	return markNPMDependencySafety(packages, result.Stdout)
}

func markNPMDependencySafety(packages []model.Package, stdout string) []model.Package {
	type npmTreeNode struct {
		Dependencies map[string]npmTreeNode `json:"dependencies"`
	}

	var root npmTreeNode
	if err := json.Unmarshal([]byte(stdout), &root); err != nil {
		return packages
	}

	topLevel := make(map[string]struct{}, len(packages))
	usedByOther := make(map[string]bool, len(packages))
	for _, pkg := range packages {
		topLevel[strings.ToLower(pkg.Name)] = struct{}{}
	}

	var visit func(owner string, node npmTreeNode)
	visit = func(owner string, node npmTreeNode) {
		for name, child := range node.Dependencies {
			lower := strings.ToLower(name)
			if _, ok := topLevel[lower]; ok && lower != owner {
				usedByOther[lower] = true
			}
			visit(owner, child)
		}
	}

	for name, child := range root.Dependencies {
		visit(strings.ToLower(name), child)
	}

	for index := range packages {
		if usedByOther[strings.ToLower(packages[index].Name)] {
			packages[index].UsedBy = "Y"
		} else {
			packages[index].UsedBy = "N"
		}
	}
	return packages
}
