#!/usr/bin/env bash
# init-garage.sh — bootstraps Garage v2.x via the built-in CLI.
# Requires the garage container to be running (docker compose up).
set -euo pipefail

COMPOSE_FILE="${GARAGE_COMPOSE_FILE:-docker/garage/docker-compose.yml}"
KEY_NAME="${GARAGE_KEY_NAME:-s3compat}"

exec_garage() {
  docker compose -f "$COMPOSE_FILE" exec -T garage /garage "$@"
}

echo "Initializing Garage..."

# Step 1: get node ID, assign layout zone + capacity, then apply.
NODE_ID=$(exec_garage node id -q 2>/dev/null | cut -d'@' -f1)
if [ -z "$NODE_ID" ]; then
  echo "Could not get Garage node ID" >&2
  exit 1
fi
echo "Node ID: $NODE_ID"

exec_garage layout assign -z dc1 -c 1G "${NODE_ID:0:12}"

CURRENT_VER=$(exec_garage layout show 2>/dev/null | grep "^Current cluster layout version:" | awk '{print $NF}' || echo "0")
NEXT_VER=$((CURRENT_VER + 1))
exec_garage layout apply --version "$NEXT_VER"
echo "Layout applied (version $NEXT_VER)."

# Step 2: create access key with bucket-create permission.
KEY_OUTPUT=$(exec_garage key create "$KEY_NAME" 2>/dev/null)
KEY_ID=$(echo "$KEY_OUTPUT" | grep "^Key ID:" | awk '{print $NF}')
KEY_SECRET=$(echo "$KEY_OUTPUT" | grep "^Secret key:" | awk '{print $NF}')
exec_garage key allow --create-bucket "$KEY_ID"
echo "Key ID: $KEY_ID"

echo "$KEY_ID" > /tmp/garage-key-id
echo "$KEY_SECRET" > /tmp/garage-key-secret
echo "Credentials written to /tmp/garage-key-id and /tmp/garage-key-secret."

# Step 3: create init bucket and grant access.
exec_garage bucket create s3compat-init
exec_garage bucket allow --read --write --owner s3compat-init --key "$KEY_ID"
echo "Garage initialization complete."
