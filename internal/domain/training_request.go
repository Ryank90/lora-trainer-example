package domain

import (
	"fmt"
	"strings"
)

type TrainingRequest struct {
	ModelType    ModelType         `json:"model_type"`
	TriggerWord  string            `json:"trigger_word"`
	DatasetURL   string            `json:"dataset_url"`
	Steps        int               `json:"steps"`
	LearningRate float64           `json:"learning_rate,omitempty"`
	LoRARank     int               `json:"lora_rank,omitempty"`
	Resolution   int               `json:"resolution,omitempty"`
	BatchSize    int               `json:"batch_size,omitempty"`
	Seed         *int64            `json:"seed,omitempty"`
	WebhookURL   string            `json:"webhook_url,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func (r *TrainingRequest) Validate() error {
	var errs []string

	if _, err := ParseModelType(string(r.ModelType)); err != nil {
		errs = append(errs, fmt.Sprintf("model_type: %v", err))
	}

	if strings.TrimSpace(r.TriggerWord) == "" {
		errs = append(errs, "trigger_word: required")
	}

	if strings.TrimSpace(r.DatasetURL) == "" {
		errs = append(errs, "dataset_url: required")
	}

	if r.Steps <= 0 {
		errs = append(errs, "steps: must be positive")
	}
	if r.Steps > 10000 {
		errs = append(errs, "steps: must be <= 10000")
	}

	if r.LearningRate < 0 {
		errs = append(errs, "learning_rate: must be non-negative")
	}

	if r.LoRARank < 0 {
		errs = append(errs, "lora_rank: must be non-negative")
	}
	if r.LoRARank > 128 {
		errs = append(errs, "lora_rank: must be <= 128")
	}

	if r.Resolution < 0 {
		errs = append(errs, "resolution: must be non-negative")
	}

	if r.BatchSize < 0 {
		errs = append(errs, "batch_size: must be non-negative")
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidRequest, strings.Join(errs, "; "))
	}

	return nil
}

func (r *TrainingRequest) ApplyDefaults() {
	if r.LearningRate == 0 {
		r.LearningRate = 1e-4
	}
	if r.LoRARank == 0 {
		r.LoRARank = 16
	}
	if r.Resolution == 0 {
		r.Resolution = 512
	}
	if r.BatchSize == 0 {
		r.BatchSize = 1
	}
}
