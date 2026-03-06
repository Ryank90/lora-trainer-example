package provider

import (
	"context"
	"time"
)

type GPUType string

const (
	GPUA100_80GB GPUType = "A100_80GB"
	GPUA100_40GB GPUType = "A100_40GB"
	GPUH100      GPUType = "H100"
	GPUL40S      GPUType = "L40S"
	GPUA6000     GPUType = "A6000"
)

type InstanceStatus string

const (
	InstanceStatusPending  InstanceStatus = "pending"
	InstanceStatusRunning  InstanceStatus = "running"
	InstanceStatusStopped  InstanceStatus = "stopped"
	InstanceStatusFailed   InstanceStatus = "failed"
)

type InstanceRequest struct {
	GPUType     GPUType           `json:"gpu_type"`
	GPUCount    int               `json:"gpu_count"`
	DiskGB      int               `json:"disk_gb"`
	Image       string            `json:"image"`
	Env         map[string]string `json:"env"`
	VolumeMount string            `json:"volume_mount,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type Instance struct {
	ID         string         `json:"id"`
	Provider   string         `json:"provider"`
	Status     InstanceStatus `json:"status"`
	GPUType    GPUType        `json:"gpu_type"`
	GPUCount   int            `json:"gpu_count"`
	IP         string         `json:"ip"`
	SSHPort    int            `json:"ssh_port"`
	SSHUser    string         `json:"ssh_user"`
	SSHKeyPath string         `json:"ssh_key_path"`
	CostPerHr  float64        `json:"cost_per_hr"`
	CreatedAt  time.Time      `json:"created_at"`
}

type GPUAvailability struct {
	GPUType   GPUType `json:"gpu_type"`
	Available int     `json:"available"`
	CostPerHr float64 `json:"cost_per_hr"`
}

type GPUProvider interface {
	Name() string
	CreateInstance(ctx context.Context, req InstanceRequest) (*Instance, error)
	GetInstance(ctx context.Context, instanceID string) (*Instance, error)
	WaitForReady(ctx context.Context, instanceID string) (*Instance, error)
	TerminateInstance(ctx context.Context, instanceID string) error
	AvailableGPUs(ctx context.Context) ([]GPUAvailability, error)
}
