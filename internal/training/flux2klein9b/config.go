package flux2klein9b

import "github.com/ryank90/lora-trainer-example/internal/domain"

var DefaultHyperparams = map[string]string{
	"learning_rate":    "1.5e-4",
	"lora_rank":        "16",
	"lora_alpha":       "16",
	"resolution":       "512",
	"batch_size":       "1",
	"gradient_accum":   "4",
	"warmup_steps":     "80",
	"optimizer":        "adamw",
	"weight_decay":     "0.01",
	"lr_scheduler":     "cosine",
	"mixed_precision":  "bf16",
	"gradient_checkpointing": "true",
}

var ModelType = domain.ModelTypeFlux2Klein9BBase
var ModelWeightPath = "/models/flux2-klein-9b"
var DockerImage = "ghcr.io/rk/lora-trainer/flux2:latest"
