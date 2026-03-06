#!/bin/bash
set -euo pipefail

REGISTRY="${REGISTRY:-ghcr.io/rk/lora-trainer}"
TAG="${TAG:-latest}"

echo "=== Building trainer images ==="
echo "Registry: ${REGISTRY}"
echo "Tag: ${TAG}"

# Build base image
echo "Building base image..."
docker build \
    -t "${REGISTRY}/base:${TAG}" \
    -f docker/trainer/Dockerfile.base \
    docker/trainer/

# Build Flux 2 image (handles all Flux variants)
echo "Building Flux 2 trainer image..."
docker build \
    -t "${REGISTRY}/flux2:${TAG}" \
    -f docker/trainer/Dockerfile.flux2 \
    docker/trainer/

# Build Qwen image
echo "Building Qwen Image 2512 trainer image..."
docker build \
    -t "${REGISTRY}/qwen:${TAG}" \
    -f docker/trainer/Dockerfile.qwen \
    docker/trainer/

echo "=== Pushing images ==="
docker push "${REGISTRY}/base:${TAG}"
docker push "${REGISTRY}/flux2:${TAG}"
docker push "${REGISTRY}/qwen:${TAG}"

echo "=== Done ==="
