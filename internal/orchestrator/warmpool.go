package orchestrator

import (
	"log/slog"
	"sync"
	"time"

	"github.com/ryank90/lora-trainer-example/internal/config"
	"github.com/ryank90/lora-trainer-example/internal/provider"
)

type poolEntry struct {
	instance *provider.Instance
	addedAt  time.Time
}

type WarmPool struct {
	cfg     config.WarmPoolConfig
	entries map[provider.GPUType][]poolEntry
	mu      sync.Mutex
	logger  *slog.Logger
}

func NewWarmPool(cfg config.WarmPoolConfig, logger *slog.Logger) *WarmPool {
	return &WarmPool{
		cfg:     cfg,
		entries: make(map[provider.GPUType][]poolEntry),
		logger:  logger,
	}
}

func (p *WarmPool) Acquire(gpuType provider.GPUType) *provider.Instance {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries, ok := p.entries[gpuType]
	if !ok || len(entries) == 0 {
		return nil
	}

	// Take the most recently added instance (warmest cache)
	entry := entries[len(entries)-1]
	p.entries[gpuType] = entries[:len(entries)-1]

	if time.Since(entry.addedAt) > p.cfg.MaxIdleTime {
		p.logger.Info("warm pool entry expired", "instance_id", entry.instance.ID)
		return nil
	}

	p.logger.Info("acquired from warm pool", "instance_id", entry.instance.ID, "gpu_type", gpuType)
	return entry.instance
}

func (p *WarmPool) Return(instance *provider.Instance) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries := p.entries[instance.GPUType]
	if len(entries) >= p.cfg.MaxSize {
		p.logger.Info("warm pool full, not returning instance", "instance_id", instance.ID)
		return
	}

	p.entries[instance.GPUType] = append(entries, poolEntry{
		instance: instance,
		addedAt:  time.Now(),
	})

	p.logger.Info("returned to warm pool", "instance_id", instance.ID, "gpu_type", instance.GPUType)
}

func (p *WarmPool) CanReturn(instance *provider.Instance) bool {
	if !p.cfg.Enabled {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.entries[instance.GPUType]) < p.cfg.MaxSize
}

func (p *WarmPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	total := 0
	for _, entries := range p.entries {
		total += len(entries)
	}
	return total
}

func (p *WarmPool) Cleanup() []provider.Instance {
	p.mu.Lock()
	defer p.mu.Unlock()

	var expired []provider.Instance

	for gpuType, entries := range p.entries {
		var active []poolEntry
		for _, entry := range entries {
			if time.Since(entry.addedAt) > p.cfg.MaxIdleTime {
				expired = append(expired, *entry.instance)
			} else {
				active = append(active, entry)
			}
		}
		p.entries[gpuType] = active
	}

	return expired
}
