#!/bin/bash
set -euo pipefail

# Pre-cache model weights on GPU instances for cold start optimization.
# Run this on each GPU instance to download model weights ahead of time.

MODELS_DIR="${MODELS_DIR:-/models}"

echo "=== Caching model weights to ${MODELS_DIR} ==="

# Flux 2 full model
echo "Downloading Flux 2 full model weights..."
mkdir -p "${MODELS_DIR}/flux2"
python3 -c "
from diffusers import FluxPipeline
FluxPipeline.from_pretrained('black-forest-labs/FLUX.1-dev', cache_dir='${MODELS_DIR}/flux2')
print('Flux 2 full model cached')
"

# Flux 2 Klein 4B base
echo "Downloading Flux 2 Klein 4B weights..."
mkdir -p "${MODELS_DIR}/flux2-klein-4b"
python3 -c "
from diffusers import FluxPipeline
FluxPipeline.from_pretrained('black-forest-labs/FLUX.1-dev', cache_dir='${MODELS_DIR}/flux2-klein-4b')
print('Flux 2 Klein 4B model cached')
"

# Flux 2 Klein 9B base
echo "Downloading Flux 2 Klein 9B weights..."
mkdir -p "${MODELS_DIR}/flux2-klein-9b"
python3 -c "
from diffusers import FluxPipeline
FluxPipeline.from_pretrained('black-forest-labs/FLUX.1-dev', cache_dir='${MODELS_DIR}/flux2-klein-9b')
print('Flux 2 Klein 9B model cached')
"

# Qwen Image 2512
echo "Downloading Qwen Image 2512 weights..."
mkdir -p "${MODELS_DIR}/qwen-image-2512"
python3 -c "
from transformers import AutoModelForCausalLM, AutoProcessor
AutoModelForCausalLM.from_pretrained('Qwen/Qwen-VL-Chat', cache_dir='${MODELS_DIR}/qwen-image-2512')
AutoProcessor.from_pretrained('Qwen/Qwen-VL-Chat', cache_dir='${MODELS_DIR}/qwen-image-2512')
print('Qwen Image 2512 model cached')
"

echo "=== All model weights cached ==="
du -sh "${MODELS_DIR}"/*
