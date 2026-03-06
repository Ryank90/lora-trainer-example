package runpod

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
	return "runpod"
}

func (p *Provider) CreateInstance(ctx context.Context, req provider.InstanceRequest) (*provider.Instance, error) {
	resp, err := p.client.CreatePod(ctx, createPodRequest{
		Name:            fmt.Sprintf("lora-trainer-%d", time.Now().Unix()),
		ImageName:       req.Image,
		GPUTypeID:       mapGPUType(req.GPUType),
		GPUCount:        req.GPUCount,
		VolumeInGB:      req.DiskGB,
		ContainerDiskGB: 50,
		VolumeMountPath: req.VolumeMount,
		Env:             req.Env,
	})
	if err != nil {
		return nil, fmt.Errorf("runpod create instance: %w", err)
	}

	return mapInstance(resp), nil
}

func (p *Provider) GetInstance(ctx context.Context, instanceID string) (*provider.Instance, error) {
	resp, err := p.client.GetPod(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("runpod get instance: %w", err)
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
				return nil, fmt.Errorf("pod %s failed", instanceID)
			}
		}
	}
}

func (p *Provider) TerminateInstance(ctx context.Context, instanceID string) error {
	return p.client.DeletePod(ctx, instanceID)
}

func (p *Provider) AvailableGPUs(ctx context.Context) ([]provider.GPUAvailability, error) {
	gpuTypes, err := p.client.GetGPUTypes(ctx)
	if err != nil {
		return nil, err
	}

	var result []provider.GPUAvailability
	for _, gt := range gpuTypes {
		gpuType := reverseMapGPUType(gt.ID)
		if gpuType == "" {
			continue
		}
		result = append(result, provider.GPUAvailability{
			GPUType:   gpuType,
			Available: gt.SecureCount + gt.CommunityCount,
			CostPerHr: gt.SecurePrice,
		})
	}
	return result, nil
}

func mapGPUType(t provider.GPUType) string {
	switch t {
	case provider.GPUA100_80GB:
		return "NVIDIA A100 80GB PCIe"
	case provider.GPUA100_40GB:
		return "NVIDIA A100-SXM4-40GB"
	case provider.GPUH100:
		return "NVIDIA H100 80GB HBM3"
	case provider.GPUL40S:
		return "NVIDIA L40S"
	case provider.GPUA6000:
		return "NVIDIA RTX A6000"
	default:
		return string(t)
	}
}

func reverseMapGPUType(id string) provider.GPUType {
	switch id {
	case "NVIDIA A100 80GB PCIe":
		return provider.GPUA100_80GB
	case "NVIDIA A100-SXM4-40GB":
		return provider.GPUA100_40GB
	case "NVIDIA H100 80GB HBM3":
		return provider.GPUH100
	case "NVIDIA L40S":
		return provider.GPUL40S
	case "NVIDIA RTX A6000":
		return provider.GPUA6000
	default:
		return ""
	}
}

func mapInstance(resp *podResponse) *provider.Instance {
	status := provider.InstanceStatusPending
	switch resp.Status {
	case "RUNNING":
		status = provider.InstanceStatusRunning
	case "EXITED", "TERMINATED":
		status = provider.InstanceStatusStopped
	case "FAILED":
		status = provider.InstanceStatusFailed
	}

	inst := &provider.Instance{
		ID:        resp.ID,
		Provider:  "runpod",
		Status:    status,
		GPUType:   reverseMapGPUType(resp.GPUType),
		GPUCount:  resp.GPUCount,
		SSHUser:   "root",
		SSHPort:   22,
		CostPerHr: resp.CostPerHr,
		CreatedAt: time.Now(),
	}

	if resp.Runtime != nil {
		for _, port := range resp.Runtime.Ports {
			if port.PrivatePort == 22 {
				inst.IP = port.IP
				inst.SSHPort = port.PublicPort
				break
			}
		}
	}

	return inst
}
