#!/bin/bash
set -euo pipefail

echo "=== LoRA Trainer ==="
echo "Job ID: ${JOB_ID}"
echo "Model Type: ${MODEL_TYPE}"
echo "Trigger Word: ${TRIGGER_WORD}"
echo "Steps: ${STEPS}"

# Preprocess dataset
echo "=== Preprocessing dataset ==="
python3 /workspace/preprocess.py \
    --dataset_url "${DATASET_URL}" \
    --output_dir /workspace/dataset \
    --trigger_word "${TRIGGER_WORD}" \
    --resolution "${RESOLUTION:-512}"

# Route to correct training script
case "${MODEL_TYPE}" in
    flux-2-trainer|flux-2-klein-4b-base-trainer|flux-2-klein-9b-base-trainer)
        echo "=== Starting Flux 2 LoRA training (variant: ${MODEL_VARIANT:-full}) ==="
        python3 /workspace/train_flux2.py \
            --model_path "${MODEL_PATH}" \
            --model_variant "${MODEL_VARIANT:-full}" \
            --dataset_dir /workspace/dataset \
            --output_dir /outputs \
            --trigger_word "${TRIGGER_WORD}" \
            --steps "${STEPS}" \
            --learning_rate "${LEARNING_RATE:-1e-4}" \
            --lora_rank "${LORA_RANK:-16}" \
            --resolution "${RESOLUTION:-512}" \
            --batch_size "${BATCH_SIZE:-1}" \
            ${SEED:+--seed "${SEED}"}
        ;;
    qwen-image-2512-trainer)
        echo "=== Starting Qwen Image 2512 LoRA training ==="
        python3 /workspace/train_qwen_image.py \
            --model_path "${MODEL_PATH}" \
            --dataset_dir /workspace/dataset \
            --output_dir /outputs \
            --trigger_word "${TRIGGER_WORD}" \
            --steps "${STEPS}" \
            --learning_rate "${LEARNING_RATE:-5e-5}" \
            --lora_rank "${LORA_RANK:-32}" \
            --resolution "${RESOLUTION:-2512}" \
            --batch_size "${BATCH_SIZE:-1}" \
            ${SEED:+--seed "${SEED}"}
        ;;
    *)
        echo "Unknown model type: ${MODEL_TYPE}"
        exit 1
        ;;
esac

# Upload results
echo "=== Uploading results ==="
python3 /workspace/upload_results.py \
    --job_id "${JOB_ID}" \
    --output_dir /outputs \
    ${WEBHOOK_URL:+--webhook_url "${WEBHOOK_URL}"}

echo "=== Training complete ==="
