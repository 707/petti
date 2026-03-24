package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

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
		})
	}

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
