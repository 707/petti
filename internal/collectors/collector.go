package collectors

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/nad/pkgview/internal/model"
)

var ErrUnavailable = errors.New("collector unavailable")

type Collector interface {
	Name() model.Source
	Available(context.Context) bool
	Collect(context.Context) ([]model.Package, model.CollectorStatus, error)
}

type CollectResult struct {
	Packages []model.Package
	Statuses []model.CollectorStatus
}

func CollectAll(ctx context.Context, list []Collector) CollectResult {
	type item struct {
		packages []model.Package
		status   model.CollectorStatus
	}

	results := make(chan item, len(list))
	var wg sync.WaitGroup

	for _, collector := range list {
		wg.Add(1)
		go func(c Collector) {
			defer wg.Done()
			pkgs, status, err := c.Collect(ctx)
			if err != nil && status.State == "" {
				status = model.CollectorStatus{
					Source:  c.Name(),
					Label:   string(c.Name()),
					State:   model.CollectorStateError,
					Details: err.Error(),
				}
			}
			results <- item{packages: pkgs, status: status}
		}(collector)
	}

	wg.Wait()
	close(results)

	out := CollectResult{}
	for result := range results {
		out.Packages = append(out.Packages, result.packages...)
		out.Statuses = append(out.Statuses, result.status)
	}

	sort.Slice(out.Packages, func(i, j int) bool {
		left := out.Packages[i]
		right := out.Packages[j]
		if left.Name == right.Name {
			return model.SourceOrder(left.Source) < model.SourceOrder(right.Source)
		}
		return strings.ToLower(left.Name) < strings.ToLower(right.Name)
	})

	sort.Slice(out.Statuses, func(i, j int) bool {
		return model.SourceOrder(out.Statuses[i].Source) < model.SourceOrder(out.Statuses[j].Source)
	})

	return out
}
