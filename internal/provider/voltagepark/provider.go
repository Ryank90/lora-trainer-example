package voltagepark

import (
	"context"
	"fmt"
	"time"

	"github.com/ryank90/lora-trainer-example/internal/provider"
)

type Provider struct {
	client *Client
}

func NewProvider(endpoint, apiKey string) *Provider {
	return &Provider{
		client: NewClient(endpoint, apiKey),
	}
}

func (p *Provider) Name() string {
	return "voltagepark"
}

func (p *Provider) CreateInstance(ctx context.Context, req provider.InstanceRequest) (*provider.Instance, error) {
	resp, err := p.client.CreateInstance(ctx, createInstanceRequest{
		GPUType:  string(req.GPUType),
		GPUCount: req.GPUCount,
		DiskGB:   req.DiskGB,
		Image:    req.Image,
		Env:      req.Env,
		Labels:   req.Labels,
	})
	if err != nil {
		return nil, fmt.Errorf("voltagepark create instance: %w", err)
	}

	return mapInstance(resp), nil
}

func (p *Provider) GetInstance(ctx context.Context, instanceID string) (*provider.Instance, error) {
	resp, err := p.client.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("voltagepark get instance: %w", err)
	}
	return mapInstance(resp), nil
}

func (p *Provider) WaitForReady(ctx context.Context, instanceID string) (*provider.Instance, error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			inst, err := p.GetInstance(ctx, instanceID)
			if err != nil {
				return nil, err
			}
			if inst.Status == provider.InstanceStatusRunning {
				return inst, nil
			}
			if inst.Status == provider.InstanceStatusFailed {
				return nil, fmt.Errorf("instance %s failed", instanceID)
			}
		}
	}
}

func (p *Provider) TerminateInstance(ctx context.Context, instanceID string) error {
	return p.client.DeleteInstance(ctx, instanceID)
}

func (p *Provider) AvailableGPUs(ctx context.Context) ([]provider.GPUAvailability, error) {
	resp, err := p.client.GetAvailability(ctx)
	if err != nil {
		return nil, err
	}

	var result []provider.GPUAvailability
	for _, gpu := range resp.GPUs {
		result = append(result, provider.GPUAvailability{
			GPUType:   provider.GPUType(gpu.Type),
			Available: gpu.Available,
			CostPerHr: gpu.CostPerHr,
		})
	}
	return result, nil
}

func mapInstance(resp *instanceResponse) *provider.Instance {
	createdAt, _ := time.Parse(time.RFC3339, resp.CreatedAt)
	return &provider.Instance{
		ID:        resp.ID,
		Provider:  "voltagepark",
		Status:    provider.InstanceStatus(resp.Status),
		GPUType:   provider.GPUType(resp.GPUType),
		GPUCount:  resp.GPUCount,
		IP:        resp.IP,
		SSHPort:   resp.SSHPort,
		SSHUser:   "root",
		CostPerHr: resp.CostPerHr,
		CreatedAt: createdAt,
	}
}
