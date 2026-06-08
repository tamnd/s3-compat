#!/usr/bin/env bash
# wait-healthy.sh — polls until clean-buckets succeeds or WAIT_TIMEOUT expires.
set -euo pipefail

TARGET="${S3COMPAT_TARGET:-minio}"
TIMEOUT="${WAIT_TIMEOUT:-60}"
INTERVAL=3

export S3COMPAT_TARGET="$TARGET"

echo "Waiting up to ${TIMEOUT}s for target '$TARGET' to become healthy..."

elapsed=0
while true; do
  if go run ./scripts/clean-buckets 2>/dev/null; then
    echo "Target '$TARGET' is healthy."
    exit 0
  fi
  if [ "$elapsed" -ge "$TIMEOUT" ]; then
    echo "Timed out after ${TIMEOUT}s waiting for '$TARGET'." >&2
    exit 1
  fi
  sleep "$INTERVAL"
  elapsed=$((elapsed + INTERVAL))
done
