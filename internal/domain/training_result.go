package domain

import "time"

type FileRef struct {
	Key          string `json:"key"`
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	ContentType  string `json:"content_type"`
	PresignedURL string `json:"presigned_url,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

type TrainingResult struct {
	Files       []FileRef         `json:"files"`
	Metrics     *TrainingMetrics  `json:"metrics,omitempty"`
	ProviderRef string            `json:"provider_ref,omitempty"`
	GPUType     string            `json:"gpu_type,omitempty"`
}

type TrainingMetrics struct {
	FinalLoss       float64 `json:"final_loss"`
	TotalSteps      int     `json:"total_steps"`
	TrainingSeconds float64 `json:"training_seconds"`
	PeakVRAMGB      float64 `json:"peak_vram_gb,omitempty"`
}
