#!/usr/bin/env bash
# init-minio.sh — creates the initial test bucket in MinIO using the mc client.
set -euo pipefail

MINIO_ENDPOINT="${S3COMPAT_ENDPOINT:-http://localhost:9000}"
MINIO_USER="${S3COMPAT_ACCESS_KEY:-minioadmin}"
MINIO_PASS="${S3COMPAT_SECRET_KEY:-minioadmin}"
BUCKET="${MINIO_BUCKET:-test-bucket}"

echo "Initializing MinIO at $MINIO_ENDPOINT..."

docker run --rm \
  --network host \
  quay.io/minio/mc:latest \
  sh -c "
    mc alias set local '$MINIO_ENDPOINT' '$MINIO_USER' '$MINIO_PASS' &&
    mc mb --ignore-existing local/$BUCKET
  "

echo "MinIO initialization complete. Bucket: $BUCKET"
