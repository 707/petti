package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nad/pkgview/internal/model"
)

type PipCollector struct {
	runner Runner
}

func NewPipCollector(runner Runner) PipCollector {
	return PipCollector{runner: runner}
}

func (c PipCollector) Name() model.Source {
	return model.SourcePip
}

func (c PipCollector) Available(context.Context) bool {
	_, err := c.command()
	return err == nil
}

func (c PipCollector) Collect(ctx context.Context) ([]model.Package, model.CollectorStatus, error) {
	cmd, err := c.command()
	if err != nil {
		return nil, model.CollectorStatus{
			Source:  model.SourcePip,
			Label:   "pip",
			State:   model.CollectorStateMissing,
			Details: "pip/pip3 not found on PATH",
		}, ErrUnavailable
	}

	result, err := c.runner.Run(ctx, cmd, "list", "--not-required", "--format=json")
	if err != nil {
		return nil, timeoutStatus(model.SourcePip, "pip", err), err
	}
	if result.ExitCode != 0 {
		return nil, exitStatus(model.SourcePip, "pip", result), fmt.Errorf("pip list failed")
	}

	type item struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	var raw []item
	if err := json.Unmarshal([]byte(result.Stdout), &raw); err != nil {
		return nil, model.CollectorStatus{
			Source:  model.SourcePip,
			Label:   "pip",
			State:   model.CollectorStateError,
			Details: err.Error(),
		}, err
	}

	sort.Slice(raw, func(i, j int) bool {
		return raw[i].Name < raw[j].Name
	})

	packages := make([]model.Package, 0, len(raw))
	for _, item := range raw {
		packages = append(packages, model.Package{
			Name:    item.Name,
			Version: item.Version,
			Source:  model.SourcePip,
			UsedBy:  "N",
		})
	}
	packages = c.addMetadata(ctx, cmd, packages)

	return packages, model.CollectorStatus{
		Source: model.SourcePip,
		Label:  "pip",
		State:  model.CollectorStateReady,
	}, nil
}

func (c PipCollector) command() (string, error) {
	if _, err := c.runner.LookPath("pip"); err == nil {
		return "pip", nil
	}
	if _, err := c.runner.LookPath("pip3"); err == nil {
		return "pip3", nil
	}
	return "", ErrUnavailable
}

func (c PipCollector) addMetadata(ctx context.Context, command string, packages []model.Package) []model.Package {
	if len(packages) == 0 {
		return packages
	}

	args := []string{"show"}
	for _, pkg := range packages {
		args = append(args, pkg.Name)
	}
	result, err := c.runner.Run(ctx, command, args...)
	if err != nil || result.ExitCode != 0 {
		return packages
	}

	metadata := parsePipShowOutput(result.Stdout)
	for index := range packages {
		info, ok := metadata[strings.ToLower(packages[index].Name)]
		if !ok {
			continue
		}
		packages[index].Description = info.Summary
		packages[index].UpdatedAt = findPipUpdatedAt(info.Location, packages[index].Name)
	}
	return packages
}
