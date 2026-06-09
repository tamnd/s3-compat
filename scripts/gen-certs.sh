#!/usr/bin/env bash
# Generate a self-signed CA and a server certificate for localhost/127.0.0.1.
# Certs are written to certs/ at the repo root. Run once before starting
# any target in TLS mode.
set -euo pipefail

DIR="$(cd "$(dirname "$0")/.." && pwd)/certs"
mkdir -p "$DIR"

# CA key + self-signed cert
openssl genrsa -out "$DIR/ca.key" 2048 2>/dev/null
openssl req -new -x509 -key "$DIR/ca.key" -out "$DIR/ca.crt" -days 365 \
  -subj "/CN=s3compat-ca" 2>/dev/null

# Server key
openssl genrsa -out "$DIR/server.key" 2048 2>/dev/null

# CSR
openssl req -new -key "$DIR/server.key" -out "$DIR/server.csr" \
  -subj "/CN=localhost" 2>/dev/null

# SAN extension file
cat > "$DIR/san.ext" <<'EXT'
[v3_req]
subjectAltName = DNS:localhost, IP:127.0.0.1
EXT

# Sign
openssl x509 -req -in "$DIR/server.csr" \
  -CA "$DIR/ca.crt" -CAkey "$DIR/ca.key" -CAcreateserial \
  -out "$DIR/server.crt" -days 365 \
  -extensions v3_req -extfile "$DIR/san.ext" 2>/dev/null

rm "$DIR/server.csr" "$DIR/san.ext" "$DIR/ca.srl" 2>/dev/null || true
chmod 600 "$DIR"/*.key
echo "certs/ written: ca.crt, server.crt, server.key"
