package flux2

import "github.com/ryank90/lora-trainer-example/internal/domain"

var DefaultHyperparams = map[string]string{
	"learning_rate":    "1e-4",
	"lora_rank":        "16",
	"lora_alpha":       "16",
	"resolution":       "512",
	"batch_size":       "1",
	"gradient_accum":   "4",
	"warmup_steps":     "100",
	"optimizer":        "adamw",
	"weight_decay":     "0.01",
	"lr_scheduler":     "cosine",
	"mixed_precision":  "bf16",
	"gradient_checkpointing": "true",
}

var ModelType = domain.ModelTypeFlux2Trainer
var ModelWeightPath = "/models/flux2"
var DockerImage = "ghcr.io/rk/lora-trainer/flux2:latest"
