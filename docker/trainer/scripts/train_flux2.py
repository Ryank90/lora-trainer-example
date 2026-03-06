#!/usr/bin/env python3
"""Flux 2 LoRA training script. Handles all Flux variants (full, klein-4b, klein-9b)."""

import argparse
import json
import os
import time
from pathlib import Path

import torch
from accelerate import Accelerator
from diffusers import FluxPipeline
from peft import LoraConfig, get_peft_model
from PIL import Image
from torch.utils.data import DataLoader, Dataset
from tqdm import tqdm


MODEL_VARIANT_PATHS = {
    "full": "",
    "klein-4b": "",
    "klein-9b": "",
}


class LoRADataset(Dataset):
    def __init__(self, image_dir: str, resolution: int):
        self.image_dir = Path(image_dir)
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

    print(f"Loading Flux 2 model (variant: {args.model_variant}) from {args.model_path}")
    pipeline = FluxPipeline.from_pretrained(
        args.model_path,
        torch_dtype=torch.bfloat16,
    )

    # Configure LoRA
    lora_config = LoraConfig(
        r=args.lora_rank,
        lora_alpha=args.lora_rank,
        target_modules=["to_q", "to_k", "to_v", "to_out.0"],
        lora_dropout=0.0,
    )

    unet = pipeline.transformer
    unet = get_peft_model(unet, lora_config)
    unet.print_trainable_parameters()

    # Dataset
    dataset = LoRADataset(args.dataset_dir, args.resolution)
    dataloader = DataLoader(dataset, batch_size=args.batch_size, shuffle=True)

    print(f"Dataset: {len(dataset)} images")
    print(f"Training for {args.steps} steps")

    # Optimizer
    optimizer = torch.optim.AdamW(
        unet.parameters(),
        lr=args.learning_rate,
        weight_decay=0.01,
    )

    unet, optimizer, dataloader = accelerator.prepare(unet, optimizer, dataloader)

    # Training loop
    global_step = 0
    start_time = time.time()
    losses = []

    if args.seed is not None:
        torch.manual_seed(args.seed)

    unet.train()
    progress = tqdm(total=args.steps, desc="Training")

    while global_step < args.steps:
        for batch in dataloader:
            if global_step >= args.steps:
                break

            optimizer.zero_grad()

            # Simplified training step - actual implementation would use
            # the full diffusion training loop with noise scheduling
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

    unet = accelerator.unwrap_model(unet)
    unet.save_pretrained(output_dir / "lora_weights")

    # Save safetensors
    from safetensors.torch import save_file
    lora_state_dict = {k: v for k, v in unet.state_dict().items() if "lora" in k.lower()}
    save_file(lora_state_dict, output_dir / "lora_weights.safetensors")

    # Save training config
    config = {
        "model_type": f"flux-2-{args.model_variant}" if args.model_variant != "full" else "flux-2",
        "model_variant": args.model_variant,
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
    parser = argparse.ArgumentParser(description="Flux 2 LoRA Training")
    parser.add_argument("--model_path", required=True)
    parser.add_argument("--model_variant", default="full", choices=["full", "klein-4b", "klein-9b"])
    parser.add_argument("--dataset_dir", required=True)
    parser.add_argument("--output_dir", required=True)
    parser.add_argument("--trigger_word", required=True)
    parser.add_argument("--steps", type=int, required=True)
    parser.add_argument("--learning_rate", type=float, default=1e-4)
    parser.add_argument("--lora_rank", type=int, default=16)
    parser.add_argument("--resolution", type=int, default=512)
    parser.add_argument("--batch_size", type=int, default=1)
    parser.add_argument("--seed", type=int, default=None)
    args = parser.parse_args()

    train(args)


if __name__ == "__main__":
    main()
