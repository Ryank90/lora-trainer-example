package flux2klein4b

import "github.com/ryank90/lora-trainer-example/internal/domain"

var DefaultHyperparams = map[string]string{
	"learning_rate":    "2e-4",
	"lora_rank":        "16",
	"lora_alpha":       "16",
	"resolution":       "512",
	"batch_size":       "2",
	"gradient_accum":   "2",
	"warmup_steps":     "50",
	"optimizer":        "adamw",
	"weight_decay":     "0.01",
	"lr_scheduler":     "cosine",
	"mixed_precision":  "bf16",
	"gradient_checkpointing": "true",
}

var ModelType = domain.ModelTypeFlux2Klein4BBase
var ModelWeightPath = "/models/flux2-klein-4b"
var DockerImage = "ghcr.io/rk/lora-trainer/flux2:latest"
