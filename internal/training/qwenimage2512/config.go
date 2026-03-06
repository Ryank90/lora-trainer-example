package qwenimage2512

import "github.com/ryank90/lora-trainer-example/internal/domain"

var DefaultHyperparams = map[string]string{
	"learning_rate":    "5e-5",
	"lora_rank":        "32",
	"lora_alpha":       "32",
	"resolution":       "2512",
	"batch_size":       "1",
	"gradient_accum":   "8",
	"warmup_steps":     "100",
	"optimizer":        "adamw",
	"weight_decay":     "0.01",
	"lr_scheduler":     "cosine",
	"mixed_precision":  "bf16",
	"gradient_checkpointing": "true",
}

var ModelType = domain.ModelTypeQwenImage2512Trainer
var ModelWeightPath = "/models/qwen-image-2512"
var DockerImage = "ghcr.io/rk/lora-trainer/qwen:latest"
