package orchestrator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ryank90/lora-trainer-example/internal/provider"
	"github.com/ryank90/lora-trainer-example/internal/training"
)

type Dispatcher struct {
	router *provider.ProviderRouter
	logger *slog.Logger
}

func NewDispatcher(router *provider.ProviderRouter, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		router: router,
		logger: logger,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, config training.ModelConfig) (*provider.Instance, error) {
	p, err := d.router.SelectProvider(ctx, config.GPUType, provider.StrategyPreferred)
	if err != nil {
		return nil, fmt.Errorf("selecting provider: %w", err)
	}

	d.logger.Info("dispatching to provider",
		"provider", p.Name(),
		"gpu_type", config.GPUType,
		"docker_image", config.DockerImage,
	)

	req := provider.InstanceRequest{
		GPUType:  config.GPUType,
		GPUCount: config.GPUCount,
		DiskGB:   config.DiskGB,
		Image:    config.DockerImage,
		Labels: map[string]string{
			"service":    "lora-trainer",
			"model_type": string(config.ModelType),
		},
	}

	instance, err := p.CreateInstance(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating instance on %s: %w", p.Name(), err)
	}

	d.logger.Info("waiting for instance ready",
		"provider", p.Name(),
		"instance_id", instance.ID,
	)

	instance, err = p.WaitForReady(ctx, instance.ID)
	if err != nil {
		// Attempt cleanup
		p.TerminateInstance(context.Background(), instance.ID)
		return nil, fmt.Errorf("waiting for instance %s: %w", instance.ID, err)
	}

	return instance, nil
}
