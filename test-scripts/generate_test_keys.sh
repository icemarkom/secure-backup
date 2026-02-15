#!/bin/sh
# Generate GPG test keys for integration testing
# Called automatically by tests if keys don't exist
# POSIX-compliant for CI/CD compatibility
#
# Scripts live in test-scripts/, generated data goes to test_data/

set -e

# Get project root (one level up from script directory)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DATA_DIR="$PROJECT_ROOT/test_data"
GNUPGHOME="$DATA_DIR/gnupg"
export GNUPGHOME

echo "Generating test GPG keys in $DATA_DIR..."

# Ensure data directory exists
mkdir -p "$DATA_DIR"

# Clean and create GPG home
rm -rf "$GNUPGHOME"
mkdir -p "$GNUPGHOME"
chmod 700 "$GNUPGHOME"

# Generate key non-interactively
cat > "$GNUPGHOME/gen-key-script" << 'GPGEOF'
%no-protection
Key-Type: RSA
Key-Length: 2048
Name-Real: Test User
Name-Email: test@secure-backup.local
Expire-Date: 0
%commit
GPGEOF

gpg --batch --gen-key "$GNUPGHOME/gen-key-script" 2>&1 | grep -v "^gpg:" || true

# Export keys
gpg --armor --export test@secure-backup.local > "$DATA_DIR/test-public.asc"
gpg --armor --export-secret-keys test@secure-backup.local > "$DATA_DIR/test-private.asc"

# Create sample test file
echo "This is a test file for secure-backup integration tests." > "$DATA_DIR/sample.txt"

echo "✓ Test keys generated successfully"
echo "  Public key: $DATA_DIR/test-public.asc"
echo "  Private key: $DATA_DIR/test-private.asc"
echo "  No passphrase required"
