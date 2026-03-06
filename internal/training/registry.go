package training

import (
	"fmt"

	"github.com/ryank90/lora-trainer-example/internal/domain"
	"github.com/ryank90/lora-trainer-example/internal/provider"
)

type ModelConfig struct {
	ModelType    domain.ModelType
	DockerImage  string
	GPUType      provider.GPUType
	GPUCount     int
	DiskGB       int
	VolumeMounts []string
	ExtraEnv     map[string]string
}

type Registry struct {
	configs map[domain.ModelType]ModelConfig
}

func NewRegistry() *Registry {
	r := &Registry{
		configs: make(map[domain.ModelType]ModelConfig),
	}
	r.registerAll()
	return r
}

func (r *Registry) Get(modelType domain.ModelType) (ModelConfig, error) {
	config, ok := r.configs[modelType]
	if !ok {
		return ModelConfig{}, fmt.Errorf("no config registered for model type: %s", modelType)
	}
	return config, nil
}

func (r *Registry) registerAll() {
	r.configs[domain.ModelTypeFlux2Trainer] = ModelConfig{
		ModelType:   domain.ModelTypeFlux2Trainer,
		DockerImage: "ghcr.io/rk/lora-trainer/flux2:latest",
		GPUType:     provider.GPUA100_80GB,
		GPUCount:    1,
		DiskGB:      100,
		VolumeMounts: []string{
			"/models/flux2:/models:ro",
			"/tmp/outputs:/outputs",
		},
		ExtraEnv: map[string]string{
			"MODEL_PATH": "/models",
		},
	}

	r.configs[domain.ModelTypeFlux2Klein4BBase] = ModelConfig{
		ModelType:   domain.ModelTypeFlux2Klein4BBase,
		DockerImage: "ghcr.io/rk/lora-trainer/flux2:latest",
		GPUType:     provider.GPUL40S,
		GPUCount:    1,
		DiskGB:      50,
		VolumeMounts: []string{
			"/models/flux2-klein-4b:/models:ro",
			"/tmp/outputs:/outputs",
		},
		ExtraEnv: map[string]string{
			"MODEL_PATH": "/models",
		},
	}

	r.configs[domain.ModelTypeFlux2Klein9BBase] = ModelConfig{
		ModelType:   domain.ModelTypeFlux2Klein9BBase,
		DockerImage: "ghcr.io/rk/lora-trainer/flux2:latest",
		GPUType:     provider.GPUA100_40GB,
		GPUCount:    1,
		DiskGB:      80,
		VolumeMounts: []string{
			"/models/flux2-klein-9b:/models:ro",
			"/tmp/outputs:/outputs",
		},
		ExtraEnv: map[string]string{
			"MODEL_PATH": "/models",
		},
	}

	r.configs[domain.ModelTypeQwenImage2512Trainer] = ModelConfig{
		ModelType:   domain.ModelTypeQwenImage2512Trainer,
		DockerImage: "ghcr.io/rk/lora-trainer/qwen:latest",
		GPUType:     provider.GPUA100_80GB,
		GPUCount:    1,
		DiskGB:      100,
		VolumeMounts: []string{
			"/models/qwen-image-2512:/models:ro",
			"/tmp/outputs:/outputs",
		},
		ExtraEnv: map[string]string{
			"MODEL_PATH": "/models",
		},
	}
}
