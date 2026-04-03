#!/usr/bin/env bash
# Generate RSA-4096 key pair for JWT RS256 signing.
# Run once before first deploy: ./scripts/keygen.sh
set -euo pipefail

mkdir -p keys
openssl genrsa -out keys/private.pem 4096
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
chmod 600 keys/private.pem
echo "✅ RSA key pair generated at keys/private.pem and keys/public.pem"
echo "   Add JWT_PRIVATE_KEY_PATH=./keys/private.pem to your .env"
echo "   Never commit keys/ to version control."
