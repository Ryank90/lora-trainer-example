package provider

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/ryank90/lora-trainer-example/internal/domain"
)

type Strategy string

const (
	StrategyCheapest  Strategy = "cheapest"
	StrategyPreferred Strategy = "preferred"
)

type ProviderRouter struct {
	providers  map[string]GPUProvider
	preference []string
	logger     *slog.Logger
}

func NewProviderRouter(providers map[string]GPUProvider, preference []string, logger *slog.Logger) *ProviderRouter {
	return &ProviderRouter{
		providers:  providers,
		preference: preference,
		logger:     logger,
	}
}

func (r *ProviderRouter) SelectProvider(ctx context.Context, gpuType GPUType, strategy Strategy) (GPUProvider, error) {
	switch strategy {
	case StrategyCheapest:
		return r.selectCheapest(ctx, gpuType)
	case StrategyPreferred:
		return r.selectPreferred(ctx, gpuType)
	default:
		return r.selectPreferred(ctx, gpuType)
	}
}

func (r *ProviderRouter) selectPreferred(ctx context.Context, gpuType GPUType) (GPUProvider, error) {
	for _, name := range r.preference {
		p, ok := r.providers[name]
		if !ok {
			continue
		}

		avail, err := p.AvailableGPUs(ctx)
		if err != nil {
			r.logger.Warn("provider availability check failed", "provider", name, "error", err)
			continue
		}

		for _, a := range avail {
			if a.GPUType == gpuType && a.Available > 0 {
				return p, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: no provider has %s available", domain.ErrProviderUnavailable, gpuType)
}

func (r *ProviderRouter) selectCheapest(ctx context.Context, gpuType GPUType) (GPUProvider, error) {
	type candidate struct {
		provider GPUProvider
		cost     float64
	}

	var candidates []candidate

	for _, p := range r.providers {
		avail, err := p.AvailableGPUs(ctx)
		if err != nil {
			r.logger.Warn("provider availability check failed", "provider", p.Name(), "error", err)
			continue
		}
		for _, a := range avail {
			if a.GPUType == gpuType && a.Available > 0 {
				candidates = append(candidates, candidate{provider: p, cost: a.CostPerHr})
			}
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("%w: no provider has %s available", domain.ErrProviderUnavailable, gpuType)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].cost < candidates[j].cost
	})

	return candidates[0].provider, nil
}
