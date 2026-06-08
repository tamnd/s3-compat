#!/usr/bin/env bash
# init-garage.sh — bootstraps Garage: layout assign, key create, bucket create.
set -euo pipefail

GARAGE_ENDPOINT="${GARAGE_ADMIN_ENDPOINT:-http://localhost:3903}"
ADMIN_TOKEN="${GARAGE_ADMIN_TOKEN:-test-admin-token}"
BUCKET="${GARAGE_BUCKET:-s3compat-init}"
KEY_NAME="${GARAGE_KEY_NAME:-s3compat}"

echo "Initializing Garage at $GARAGE_ENDPOINT..."

# Step 1: get node ID.
NODE_ID=$(curl -sf -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$GARAGE_ENDPOINT/v1/status" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -z "$NODE_ID" ]; then
  echo "Could not get Garage node ID" >&2
  exit 1
fi
echo "Node ID: $NODE_ID"

# Step 2: assign layout and apply.
curl -sf -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "[{\"id\":\"$NODE_ID\",\"zone\":\"dc1\",\"capacity\":1}]" \
  "$GARAGE_ENDPOINT/v1/layout" > /dev/null

LAYOUT_VER=$(curl -sf -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$GARAGE_ENDPOINT/v1/layout" | grep -o '"version":[0-9]*' | head -1 | cut -d: -f2)
NEXT_VER=$((LAYOUT_VER + 1))

curl -sf -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"version\":$NEXT_VER}" \
  "$GARAGE_ENDPOINT/v1/layout/apply" > /dev/null
echo "Layout applied (version $NEXT_VER)."

# Step 3: create access key.
KEY_RESP=$(curl -sf -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$KEY_NAME\"}" \
  "$GARAGE_ENDPOINT/v1/key")
KEY_ID=$(echo "$KEY_RESP" | grep -o '"accessKeyId":"[^"]*"' | cut -d'"' -f4)
KEY_SECRET=$(echo "$KEY_RESP" | grep -o '"secretAccessKey":"[^"]*"' | cut -d'"' -f4)
echo "Key ID: $KEY_ID"

echo "$KEY_ID" > /tmp/garage-key-id
echo "$KEY_SECRET" > /tmp/garage-key-secret
echo "Credentials written to /tmp/garage-key-id and /tmp/garage-key-secret."

# Step 4: create bucket and grant access.
curl -sf -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"globalAlias\":\"$BUCKET\"}" \
  "$GARAGE_ENDPOINT/v1/bucket" > /dev/null

BUCKET_ID=$(curl -sf -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$GARAGE_ENDPOINT/v1/bucket?alias=$BUCKET" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

curl -sf -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"bucketId\":\"$BUCKET_ID\",\"accessKeyId\":\"$KEY_ID\",\"permissions\":{\"read\":true,\"write\":true,\"owner\":true}}" \
  "$GARAGE_ENDPOINT/v1/bucket/allow" > /dev/null

echo "Garage initialization complete. Bucket: $BUCKET"
