#!/usr/bin/env python3
"""Upload training outputs to R2 and POST callback to API."""

import argparse
import json
import os
from pathlib import Path

import boto3
import requests


def upload_to_r2(output_dir: str, job_id: str) -> list[dict]:
    """Upload all output files to R2."""
    s3 = boto3.client(
        "s3",
        endpoint_url=os.environ.get("R2_ENDPOINT"),
        aws_access_key_id=os.environ.get("R2_ACCESS_KEY_ID"),
        aws_secret_access_key=os.environ.get("R2_SECRET_ACCESS_KEY"),
        region_name=os.environ.get("R2_REGION", "auto"),
    )

    bucket = os.environ.get("R2_BUCKET", "lora-trainer")
    uploaded_files = []

    output_path = Path(output_dir)
    for filepath in output_path.rglob("*"):
        if filepath.is_dir():
            continue

        relative = filepath.relative_to(output_path)
        key = f"jobs/{job_id}/outputs/{relative}"

        content_type = "application/octet-stream"
        if filepath.suffix == ".json":
            content_type = "application/json"
        elif filepath.suffix == ".safetensors":
            content_type = "application/octet-stream"
        elif filepath.suffix == ".txt":
            content_type = "text/plain"

        print(f"Uploading {relative} -> s3://{bucket}/{key}")
        s3.upload_file(
            str(filepath),
            bucket,
            key,
            ExtraArgs={"ContentType": content_type},
        )

        file_size = filepath.stat().st_size
        uploaded_files.append({
            "key": key,
            "filename": filepath.name,
            "size": file_size,
            "content_type": content_type,
        })

    print(f"Uploaded {len(uploaded_files)} files")
    return uploaded_files


def post_callback(webhook_url: str, job_id: str, files: list[dict], metrics: dict):
    """POST results callback to webhook URL."""
    payload = {
        "job_id": job_id,
        "status": "completed",
        "files": files,
        "metrics": metrics,
    }

    print(f"Posting callback to {webhook_url}")
    response = requests.post(
        webhook_url,
        json=payload,
        timeout=30,
        headers={"Content-Type": "application/json"},
    )
    response.raise_for_status()
    print(f"Callback response: {response.status_code}")


def main():
    parser = argparse.ArgumentParser(description="Upload training results")
    parser.add_argument("--job_id", required=True)
    parser.add_argument("--output_dir", required=True)
    parser.add_argument("--webhook_url", default=None)
    args = parser.parse_args()

    files = upload_to_r2(args.output_dir, args.job_id)

    # Load metrics if available
    metrics_path = Path(args.output_dir) / "metrics.json"
    metrics = {}
    if metrics_path.exists():
        with open(metrics_path) as f:
            metrics = json.load(f)

    if args.webhook_url:
        post_callback(args.webhook_url, args.job_id, files, metrics)


if __name__ == "__main__":
    main()
