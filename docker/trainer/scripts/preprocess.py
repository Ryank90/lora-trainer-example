#!/usr/bin/env python3
"""Dataset preprocessing: download, validate images, generate captions, inject trigger words."""

import argparse
import os
import zipfile
import tarfile
import shutil
from pathlib import Path

import requests
from PIL import Image
from tqdm import tqdm


def download_dataset(url: str, output_path: str) -> str:
    """Download dataset archive from URL."""
    print(f"Downloading dataset from {url}")
    local_path = os.path.join(output_path, "dataset_archive")
    os.makedirs(output_path, exist_ok=True)

    response = requests.get(url, stream=True, timeout=600)
    response.raise_for_status()

    total_size = int(response.headers.get("content-length", 0))
    with open(local_path, "wb") as f:
        with tqdm(total=total_size, unit="B", unit_scale=True) as pbar:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)
                pbar.update(len(chunk))

    return local_path


def extract_archive(archive_path: str, output_dir: str) -> str:
    """Extract zip or tar archive."""
    extract_dir = os.path.join(output_dir, "images")
    os.makedirs(extract_dir, exist_ok=True)

    if zipfile.is_zipfile(archive_path):
        with zipfile.ZipFile(archive_path, "r") as z:
            z.extractall(extract_dir)
    elif tarfile.is_tarfile(archive_path):
        with tarfile.open(archive_path, "r:*") as t:
            t.extractall(extract_dir)
    else:
        # Assume it's a directory of images already
        shutil.copytree(archive_path, extract_dir, dirs_exist_ok=True)

    os.remove(archive_path)
    return extract_dir


def validate_and_resize_images(image_dir: str, resolution: int) -> list[str]:
    """Validate images and resize to target resolution."""
    valid_extensions = {".jpg", ".jpeg", ".png", ".webp", ".bmp"}
    valid_images = []

    for root, _, files in os.walk(image_dir):
        for fname in sorted(files):
            ext = Path(fname).suffix.lower()
            if ext not in valid_extensions:
                continue

            filepath = os.path.join(root, fname)
            try:
                with Image.open(filepath) as img:
                    img.verify()

                with Image.open(filepath) as img:
                    img = img.convert("RGB")
                    # Resize maintaining aspect ratio, then center crop
                    w, h = img.size
                    scale = resolution / min(w, h)
                    new_w, new_h = int(w * scale), int(h * scale)
                    img = img.resize((new_w, new_h), Image.LANCZOS)

                    # Center crop
                    left = (new_w - resolution) // 2
                    top = (new_h - resolution) // 2
                    img = img.crop((left, top, left + resolution, top + resolution))
                    img.save(filepath)
                    valid_images.append(filepath)

            except Exception as e:
                print(f"Skipping invalid image {fname}: {e}")

    print(f"Validated {len(valid_images)} images")
    return valid_images


def create_captions(image_paths: list[str], trigger_word: str, output_dir: str):
    """Create caption files with trigger word for each image."""
    for img_path in image_paths:
        caption_path = Path(img_path).with_suffix(".txt")
        caption = f"A photo of {trigger_word}"
        caption_path.write_text(caption)

    print(f"Created {len(image_paths)} caption files with trigger word: {trigger_word}")


def main():
    parser = argparse.ArgumentParser(description="Preprocess training dataset")
    parser.add_argument("--dataset_url", required=True, help="URL to dataset archive")
    parser.add_argument("--output_dir", required=True, help="Output directory")
    parser.add_argument("--trigger_word", required=True, help="Trigger word for captions")
    parser.add_argument("--resolution", type=int, default=512, help="Target resolution")
    args = parser.parse_args()

    archive_path = download_dataset(args.dataset_url, args.output_dir)
    image_dir = extract_archive(archive_path, args.output_dir)
    valid_images = validate_and_resize_images(image_dir, args.resolution)

    if len(valid_images) == 0:
        raise ValueError("No valid images found in dataset")

    create_captions(valid_images, args.trigger_word, args.output_dir)
    print(f"Preprocessing complete: {len(valid_images)} images ready")


if __name__ == "__main__":
    main()
