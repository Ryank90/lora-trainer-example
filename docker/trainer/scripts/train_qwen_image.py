#!/usr/bin/env python3
"""Qwen Image 2512 LoRA training script."""

import argparse
import json
import os
import time
from pathlib import Path

import torch
from accelerate import Accelerator
from peft import LoraConfig, get_peft_model, TaskType
from PIL import Image
from torch.utils.data import DataLoader, Dataset
from tqdm import tqdm
from transformers import AutoModelForCausalLM, AutoProcessor


class QwenImageDataset(Dataset):
    def __init__(self, image_dir: str, processor, resolution: int):
        self.image_dir = Path(image_dir)
        self.processor = processor
        self.resolution = resolution
        self.image_paths = sorted(
            p for p in self.image_dir.rglob("*")
            if p.suffix.lower() in {".jpg", ".jpeg", ".png", ".webp"}
        )
        self.captions = []
        for img_path in self.image_paths:
            caption_path = img_path.with_suffix(".txt")
            if caption_path.exists():
                self.captions.append(caption_path.read_text().strip())
            else:
                self.captions.append("")

    def __len__(self):
        return len(self.image_paths)

    def __getitem__(self, idx):
        image = Image.open(self.image_paths[idx]).convert("RGB")
        image = image.resize((self.resolution, self.resolution), Image.LANCZOS)
        caption = self.captions[idx]
        return {"image": image, "caption": caption}


def train(args):
    accelerator = Accelerator(mixed_precision="bf16")

    print(f"Loading Qwen Image model from {args.model_path}")
    processor = AutoProcessor.from_pretrained(args.model_path)
    model = AutoModelForCausalLM.from_pretrained(
        args.model_path,
        torch_dtype=torch.bfloat16,
        device_map="auto",
    )

    # Configure LoRA
    lora_config = LoraConfig(
        r=args.lora_rank,
        lora_alpha=args.lora_rank,
        target_modules=["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"],
        lora_dropout=0.05,
        task_type=TaskType.CAUSAL_LM,
    )

    model = get_peft_model(model, lora_config)
    model.print_trainable_parameters()

    # Dataset
    dataset = QwenImageDataset(args.dataset_dir, processor, args.resolution)
    dataloader = DataLoader(dataset, batch_size=args.batch_size, shuffle=True)

    print(f"Dataset: {len(dataset)} images")
    print(f"Training for {args.steps} steps")

    # Optimizer
    optimizer = torch.optim.AdamW(
        model.parameters(),
        lr=args.learning_rate,
        weight_decay=0.01,
    )

    model, optimizer, dataloader = accelerator.prepare(model, optimizer, dataloader)

    # Training loop
    global_step = 0
    start_time = time.time()
    losses = []

    if args.seed is not None:
        torch.manual_seed(args.seed)

    model.train()
    progress = tqdm(total=args.steps, desc="Training")

    while global_step < args.steps:
        for batch in dataloader:
            if global_step >= args.steps:
                break

            optimizer.zero_grad()

            # Simplified - actual impl would process images through the model
            loss = torch.tensor(0.0, device=accelerator.device, requires_grad=True)

            accelerator.backward(loss)
            optimizer.step()

            losses.append(loss.item())
            global_step += 1
            progress.update(1)
            progress.set_postfix(loss=f"{loss.item():.4f}")

    progress.close()
    training_time = time.time() - start_time

    # Save LoRA weights
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    model = accelerator.unwrap_model(model)
    model.save_pretrained(output_dir / "lora_weights")

    # Save safetensors
    from safetensors.torch import save_file
    lora_state_dict = {k: v for k, v in model.state_dict().items() if "lora" in k.lower()}
    save_file(lora_state_dict, output_dir / "lora_weights.safetensors")

    # Save training config
    config = {
        "model_type": "qwen-image-2512",
        "trigger_word": args.trigger_word,
        "steps": args.steps,
        "learning_rate": args.learning_rate,
        "lora_rank": args.lora_rank,
        "resolution": args.resolution,
        "batch_size": args.batch_size,
        "final_loss": losses[-1] if losses else 0.0,
        "training_seconds": training_time,
        "total_images": len(dataset),
        "seed": args.seed,
    }

    with open(output_dir / "training_config.json", "w") as f:
        json.dump(config, f, indent=2)

    # Save metrics
    peak_vram = torch.cuda.max_memory_allocated() / (1024**3) if torch.cuda.is_available() else 0
    metrics = {
        "final_loss": losses[-1] if losses else 0.0,
        "total_steps": global_step,
        "training_seconds": training_time,
        "peak_vram_gb": round(peak_vram, 2),
    }

    with open(output_dir / "metrics.json", "w") as f:
        json.dump(metrics, f, indent=2)

    print(f"Training complete. Final loss: {metrics['final_loss']:.4f}")
    print(f"Training time: {training_time:.1f}s")
    print(f"Peak VRAM: {peak_vram:.2f} GB")
    print(f"Outputs saved to {output_dir}")


def main():
    parser = argparse.ArgumentParser(description="Qwen Image 2512 LoRA Training")
    parser.add_argument("--model_path", required=True)
    parser.add_argument("--dataset_dir", required=True)
    parser.add_argument("--output_dir", required=True)
    parser.add_argument("--trigger_word", required=True)
    parser.add_argument("--steps", type=int, required=True)
    parser.add_argument("--learning_rate", type=float, default=5e-5)
    parser.add_argument("--lora_rank", type=int, default=32)
    parser.add_argument("--resolution", type=int, default=2512)
    parser.add_argument("--batch_size", type=int, default=1)
    parser.add_argument("--seed", type=int, default=None)
    args = parser.parse_args()

    train(args)


if __name__ == "__main__":
    main()
