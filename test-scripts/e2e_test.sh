#!/bin/sh
# Copyright 2026 Marko Milivojevic
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0

# End-to-end pipeline test for secure-backup
# Tests: backup → list → verify (quick + full) → restore → diff
#
# Usage: sh e2e/e2e_test.sh
# Requirements: Go toolchain, gpg
#
# POSIX-compliant. Runs in CI and locally.

set -e

# --- Configuration ---
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TMPDIR_E2E="$(mktemp -d)"
BINARY="$TMPDIR_E2E/secure-backup"

# Colors (disable if not a terminal)
if [ -t 1 ]; then
  GREEN='\033[0;32m'
  RED='\033[0;31m'
  BOLD='\033[1m'
  NC='\033[0m'
else
  GREEN=''
  RED=''
  BOLD=''
  NC=''
fi

pass() {
  printf "${GREEN}[PASS]${NC} %s\n" "$1"
}

fail() {
  printf "${RED}[FAIL]${NC} %s\n" "$1"
  exit 1
}

step() {
  printf "${BOLD}[STEP]${NC} %s\n" "$1"
}

# --- Cleanup on exit ---
cleanup() {
  rm -rf "$TMPDIR_E2E"
}
trap cleanup EXIT

# --- Step 0: Build binary ---
step "Building binary"
cd "$PROJECT_ROOT"
go build -o "$BINARY" .
test -x "$BINARY" || fail "Binary not found at $BINARY"
pass "Binary built: $BINARY"

# --- Step 1: Generate test GPG keys ---
step "Generating test GPG keys"
sh "$PROJECT_ROOT/test-scripts/generate_test_keys.sh" > /dev/null 2>&1
PUBLIC_KEY="$PROJECT_ROOT/test_data/test-public.asc"
PRIVATE_KEY="$PROJECT_ROOT/test_data/test-private.asc"
test -f "$PUBLIC_KEY" || fail "Public key not found"
test -f "$PRIVATE_KEY" || fail "Private key not found"
pass "GPG test keys ready"

# --- Step 2: Create test data (~12KB total) ---
step "Creating test data"
SOURCE_DIR="$TMPDIR_E2E/source"
BACKUP_DIR="$TMPDIR_E2E/backups"
RESTORE_DIR="$TMPDIR_E2E/restore"
mkdir -p "$SOURCE_DIR/subdir/nested" "$BACKUP_DIR" "$RESTORE_DIR"

# Small text file (~50 bytes)
echo "Hello, secure-backup! This is a small test file." > "$SOURCE_DIR/small.txt"

# Medium text file (~1KB)
i=0
while [ "$i" -lt 20 ]; do
  echo "Line $i: The quick brown fox jumps over the lazy dog."
  i=$((i + 1))
done > "$SOURCE_DIR/medium.txt"

# Binary-like content (~10KB)
dd if=/dev/urandom of="$SOURCE_DIR/binary.dat" bs=1024 count=10 2>/dev/null

# Empty file
touch "$SOURCE_DIR/empty.txt"

# Nested directory files
echo "Level 1 file" > "$SOURCE_DIR/subdir/level1.txt"
echo "Level 2 file" > "$SOURCE_DIR/subdir/nested/level2.txt"

# Internal symlink
ln -s small.txt "$SOURCE_DIR/link_to_small.txt"

FILE_COUNT=$(find "$SOURCE_DIR" -type f | wc -l | tr -d ' ')
pass "Test data created: $FILE_COUNT files + 1 symlink"

# --- Step 3: Backup dry-run (regression for #20) ---
step "Running backup dry-run"
"$BINARY" backup \
  --source "$SOURCE_DIR" \
  --dest "$BACKUP_DIR" \
  --public-key "$PUBLIC_KEY" \
  --dry-run --verbose

# Verify no side effects
test ! -f "$BACKUP_DIR/.backup.lock" || fail "Dry-run created .backup.lock"
DRY_BACKUP_COUNT=$(find "$BACKUP_DIR" -name "backup_*.tar.gz.gpg" | wc -l | tr -d ' ')
test "$DRY_BACKUP_COUNT" -eq 0 || fail "Dry-run created a backup file"
DRY_MANIFEST_COUNT=$(find "$BACKUP_DIR" -name "*_manifest.json" | wc -l | tr -d ' ')
test "$DRY_MANIFEST_COUNT" -eq 0 || fail "Dry-run created a manifest file"

pass "Backup dry-run produced no side effects"

# --- Step 4: Backup ---
step "Running backup"
"$BINARY" backup \
  --source "$SOURCE_DIR" \
  --dest "$BACKUP_DIR" \
  --public-key "$PUBLIC_KEY" \
  --verbose

# Verify backup file exists
BACKUP_FILE=$(find "$BACKUP_DIR" -name "backup_*.tar.gz.gpg" -not -name "*.tmp" | head -1)
test -n "$BACKUP_FILE" || fail "No backup file found"
test -s "$BACKUP_FILE" || fail "Backup file is empty"

# Verify manifest exists
MANIFEST_FILE=$(find "$BACKUP_DIR" -name "*_manifest.json" | head -1)
test -n "$MANIFEST_FILE" || fail "No manifest file found"
test -s "$MANIFEST_FILE" || fail "Manifest file is empty"

# Verify manifest contains both size fields with positive values
grep -q '"compressed_size_bytes":' "$MANIFEST_FILE" || fail "Manifest missing compressed_size_bytes"
grep -q '"uncompressed_size_bytes":' "$MANIFEST_FILE" || fail "Manifest missing uncompressed_size_bytes"
# Verify sizes are positive integers (not zero)
COMPRESSED=$(grep -o '"compressed_size_bytes": [0-9]*' "$MANIFEST_FILE" | grep -o '[0-9]*$')
UNCOMPRESSED=$(grep -o '"uncompressed_size_bytes": [0-9]*' "$MANIFEST_FILE" | grep -o '[0-9]*$')
test "$COMPRESSED" -gt 0 || fail "compressed_size_bytes should be > 0, got $COMPRESSED"
test "$UNCOMPRESSED" -gt 0 || fail "uncompressed_size_bytes should be > 0, got $UNCOMPRESSED"

# Verify no leftover temp or lock files
LEFTOVER_TMP=$(find "$BACKUP_DIR" -name "*.tmp" | wc -l | tr -d ' ')
LEFTOVER_LOCK=$(find "$BACKUP_DIR" -name ".backup.lock" | wc -l | tr -d ' ')
test "$LEFTOVER_TMP" -eq 0 || fail "Found leftover .tmp files"
test "$LEFTOVER_LOCK" -eq 0 || fail "Found leftover .backup.lock"

pass "Backup created: $(basename "$BACKUP_FILE")"

# --- Step 4: List ---
step "Running list"
LIST_OUTPUT=$("$BINARY" list --dest "$BACKUP_DIR")

echo "$LIST_OUTPUT" | grep -q "Source:" || fail "List output missing Source:"
echo "$LIST_OUTPUT" | grep -q "Checksum:" || fail "List output missing Checksum:"
echo "$LIST_OUTPUT" | grep -q "$(basename "$BACKUP_FILE")" || fail "List output missing backup filename"

pass "List shows backup with manifest metadata"

# --- Step 5.1: Verify dry-run (quick) ---
step "Running quick verify dry-run"
"$BINARY" verify --file "$BACKUP_FILE" --quick --dry-run
pass "Quick verify dry-run exited successfully"

# --- Step 5.2: Verify (quick) ---
step "Running quick verify"
"$BINARY" verify --file "$BACKUP_FILE" --quick
pass "Quick verification passed"

# --- Step 6.1: Verify dry-run (full) ---
step "Running full verify dry-run"
"$BINARY" verify --file "$BACKUP_FILE" --private-key "$PRIVATE_KEY" --dry-run
pass "Full verify dry-run exited successfully"

# --- Step 6.2: Verify (full) ---
step "Running full verify"
"$BINARY" verify --file "$BACKUP_FILE" --private-key "$PRIVATE_KEY" --verbose
pass "Full verification passed"

# --- Step 7.1: Restore dry-run ---
step "Running restore dry-run"
DRY_RESTORE_DIR="$TMPDIR_E2E/dry-restore"
"$BINARY" restore \
  --file "$BACKUP_FILE" \
  --dest "$DRY_RESTORE_DIR" \
  --private-key "$PRIVATE_KEY" \
  --dry-run --verbose

# Verify no files were extracted
test ! -d "$DRY_RESTORE_DIR" || fail "Dry-run created restore directory"
pass "Restore dry-run produced no side effects"

# --- Step 7.2: Restore ---
step "Running restore"
"$BINARY" restore \
  --file "$BACKUP_FILE" \
  --dest "$RESTORE_DIR" \
  --private-key "$PRIVATE_KEY" \
  --verbose

# Restored tree is under RESTORE_DIR/source/ (tar preserves directory name)
RESTORED_SOURCE="$RESTORE_DIR/source"
test -d "$RESTORED_SOURCE" || fail "Restored source directory not found"

pass "Restore completed"

# --- Step 8: Diff ---
step "Comparing source and restored data"

# Compare regular files byte-for-byte
# Note: diff -r follows symlinks by default, which is what we want for content comparison
diff -r "$SOURCE_DIR" "$RESTORED_SOURCE" || fail "Restored files differ from source"

# Verify symlink is preserved as a symlink
if [ -L "$RESTORED_SOURCE/link_to_small.txt" ]; then
  TARGET=$(readlink "$RESTORED_SOURCE/link_to_small.txt")
  test "$TARGET" = "small.txt" || fail "Symlink target mismatch: got '$TARGET', want 'small.txt'"
  pass "Symlink preserved correctly"
else
  fail "Symlink not preserved as symlink"
fi

pass "All files match source"

# ═══════════════════════════════════════════
# AGE ENCRYPTION PIPELINE
# ═══════════════════════════════════════════

# --- Step 10: Generate AGE keys ---
step "Generating AGE keys"
AGE_KEY_FILE="$TMPDIR_E2E/age-key.txt"
age-keygen -o "$AGE_KEY_FILE" 2>/dev/null
AGE_RECIPIENT=$(grep "^# public key:" "$AGE_KEY_FILE" | sed 's/^# public key: //')
test -n "$AGE_RECIPIENT" || fail "Could not extract AGE recipient from key file"
pass "AGE key pair generated"

# --- Step 11: AGE Backup ---
step "Running AGE backup"
AGE_BACKUP_DIR="$TMPDIR_E2E/age-backups"
AGE_RESTORE_DIR="$TMPDIR_E2E/age-restore"
mkdir -p "$AGE_BACKUP_DIR" "$AGE_RESTORE_DIR"

"$BINARY" backup \
  --source "$SOURCE_DIR" \
  --dest "$AGE_BACKUP_DIR" \
  --public-key "$AGE_RECIPIENT" \
  --encryption age \
  --verbose

AGE_BACKUP_FILE=$(find "$AGE_BACKUP_DIR" -name "backup_*.tar.gz.age" -not -name "*.tmp" | head -1)
test -n "$AGE_BACKUP_FILE" || fail "No AGE backup file found"
test -s "$AGE_BACKUP_FILE" || fail "AGE backup file is empty"
pass "AGE backup created: $(basename "$AGE_BACKUP_FILE")"

# --- Step 12: AGE Quick verify ---
step "Running AGE quick verify"
"$BINARY" verify --file "$AGE_BACKUP_FILE" --quick
pass "AGE quick verification passed"

# --- Step 13: AGE Full verify ---
step "Running AGE full verify"
"$BINARY" verify --file "$AGE_BACKUP_FILE" --private-key "$AGE_KEY_FILE" --verbose
pass "AGE full verification passed"

# --- Step 14: AGE Restore ---
step "Running AGE restore"
"$BINARY" restore \
  --file "$AGE_BACKUP_FILE" \
  --dest "$AGE_RESTORE_DIR" \
  --private-key "$AGE_KEY_FILE" \
  --verbose

AGE_RESTORED_SOURCE="$AGE_RESTORE_DIR/source"
test -d "$AGE_RESTORED_SOURCE" || fail "AGE restored source directory not found"
pass "AGE restore completed"

# --- Step 15: AGE Diff ---
step "Comparing AGE-restored data with source"
diff -r "$SOURCE_DIR" "$AGE_RESTORED_SOURCE" || fail "AGE restored files differ from source"

if [ -L "$AGE_RESTORED_SOURCE/link_to_small.txt" ]; then
  TARGET=$(readlink "$AGE_RESTORED_SOURCE/link_to_small.txt")
  test "$TARGET" = "small.txt" || fail "Symlink target mismatch: got '$TARGET', want 'small.txt'"
  pass "AGE: Symlink preserved correctly"
else
  fail "AGE: Symlink not preserved"
fi

pass "AGE: All files match source"

# ═══════════════════════════════════════════
# COMPRESSION NONE PIPELINE
# ═══════════════════════════════════════════

# --- Step 20: Compression None Backup ---
step "Running backup with --compression none"
NONE_BACKUP_DIR="$TMPDIR_E2E/none-backups"
NONE_RESTORE_DIR="$TMPDIR_E2E/none-restore"
mkdir -p "$NONE_BACKUP_DIR" "$NONE_RESTORE_DIR"

"$BINARY" backup \
  --source "$SOURCE_DIR" \
  --dest "$NONE_BACKUP_DIR" \
  --public-key "$PUBLIC_KEY" \
  --compression none \
  --verbose

NONE_BACKUP_FILE=$(find "$NONE_BACKUP_DIR" -name "backup_*.tar.gpg" -not -name "*.tmp" | head -1)
test -n "$NONE_BACKUP_FILE" || fail "No none-compression backup file found (.tar.gpg)"
test -s "$NONE_BACKUP_FILE" || fail "None-compression backup file is empty"

# Verify manifest exists and records compression=none
NONE_MANIFEST=$(find "$NONE_BACKUP_DIR" -name "*_manifest.json" | head -1)
test -n "$NONE_MANIFEST" || fail "No manifest for none-compression backup"
grep -q '"compression": "none"' "$NONE_MANIFEST" || fail "Manifest should record compression as 'none'"

pass "None-compression backup created: $(basename "$NONE_BACKUP_FILE")"

# --- Step 21: Compression None Quick Verify ---
step "Running quick verify on none-compression backup"
"$BINARY" verify --file "$NONE_BACKUP_FILE" --quick
pass "None-compression quick verify passed"

# --- Step 22: Compression None Full Verify ---
step "Running full verify on none-compression backup"
"$BINARY" verify --file "$NONE_BACKUP_FILE" --private-key "$PRIVATE_KEY" --verbose
pass "None-compression full verify passed"

# --- Step 23: Compression None Restore ---
step "Running restore on none-compression backup"
"$BINARY" restore \
  --file "$NONE_BACKUP_FILE" \
  --dest "$NONE_RESTORE_DIR" \
  --private-key "$PRIVATE_KEY" \
  --verbose

NONE_RESTORED_SOURCE="$NONE_RESTORE_DIR/source"
test -d "$NONE_RESTORED_SOURCE" || fail "None-compression restored source directory not found"
pass "None-compression restore completed"

# --- Step 24: Compression None Diff ---
step "Comparing none-compression restored data with source"
diff -r "$SOURCE_DIR" "$NONE_RESTORED_SOURCE" || fail "None-compression restored files differ from source"

if [ -L "$NONE_RESTORED_SOURCE/link_to_small.txt" ]; then
  TARGET=$(readlink "$NONE_RESTORED_SOURCE/link_to_small.txt")
  test "$TARGET" = "small.txt" || fail "Symlink target mismatch: got '$TARGET', want 'small.txt'"
  pass "None-compression: Symlink preserved correctly"
else
  fail "None-compression: Symlink not preserved"
fi

pass "None-compression: All files match source"

# ═══════════════════════════════════════════
# COMPRESSION ZSTD PIPELINE
# ═══════════════════════════════════════════

# --- Step 30: Compression Zstd Backup ---
step "Running backup with --compression zstd"
ZSTD_BACKUP_DIR="$TMPDIR_E2E/zstd-backups"
ZSTD_RESTORE_DIR="$TMPDIR_E2E/zstd-restore"
mkdir -p "$ZSTD_BACKUP_DIR" "$ZSTD_RESTORE_DIR"

"$BINARY" backup \
  --source "$SOURCE_DIR" \
  --dest "$ZSTD_BACKUP_DIR" \
  --public-key "$PUBLIC_KEY" \
  --compression zstd \
  --verbose

ZSTD_BACKUP_FILE=$(find "$ZSTD_BACKUP_DIR" -name "backup_*.tar.zst.gpg" -not -name "*.tmp" | head -1)
test -n "$ZSTD_BACKUP_FILE" || fail "No zstd backup file found (.tar.zst.gpg)"
test -s "$ZSTD_BACKUP_FILE" || fail "Zstd backup file is empty"

# Verify manifest exists and records compression=zstd
ZSTD_MANIFEST=$(find "$ZSTD_BACKUP_DIR" -name "*_manifest.json" | head -1)
test -n "$ZSTD_MANIFEST" || fail "No manifest for zstd backup"
grep -q '"compression": "zstd"' "$ZSTD_MANIFEST" || fail "Manifest should record compression as 'zstd'"

pass "Zstd backup created: $(basename "$ZSTD_BACKUP_FILE")"

# --- Step 31: Compression Zstd Quick Verify ---
step "Running quick verify on zstd backup"
"$BINARY" verify --file "$ZSTD_BACKUP_FILE" --quick
pass "Zstd quick verify passed"

# --- Step 32: Compression Zstd Full Verify ---
step "Running full verify on zstd backup"
"$BINARY" verify --file "$ZSTD_BACKUP_FILE" --private-key "$PRIVATE_KEY" --verbose
pass "Zstd full verify passed"

# --- Step 33: Compression Zstd Restore ---
step "Running restore on zstd backup"
"$BINARY" restore \
  --file "$ZSTD_BACKUP_FILE" \
  --dest "$ZSTD_RESTORE_DIR" \
  --private-key "$PRIVATE_KEY" \
  --verbose

ZSTD_RESTORED_SOURCE="$ZSTD_RESTORE_DIR/source"
test -d "$ZSTD_RESTORED_SOURCE" || fail "Zstd restored source directory not found"
pass "Zstd restore completed"

# --- Step 34: Compression Zstd Diff ---
step "Comparing zstd restored data with source"
diff -r "$SOURCE_DIR" "$ZSTD_RESTORED_SOURCE" || fail "Zstd restored files differ from source"

if [ -L "$ZSTD_RESTORED_SOURCE/link_to_small.txt" ]; then
  TARGET=$(readlink "$ZSTD_RESTORED_SOURCE/link_to_small.txt")
  test "$TARGET" = "small.txt" || fail "Symlink target mismatch: got '$TARGET', want 'small.txt'"
  pass "Zstd: Symlink preserved correctly"
else
  fail "Zstd: Symlink not preserved"
fi

pass "Zstd: All files match source"

# ═══════════════════════════════════════════
# COMPRESSION ZSTD + AGE PIPELINE
# ═══════════════════════════════════════════

# --- Step 40: Zstd + AGE Backup ---
step "Running backup with --compression zstd --encryption age"
ZSTD_AGE_BACKUP_DIR="$TMPDIR_E2E/zstd-age-backups"
ZSTD_AGE_RESTORE_DIR="$TMPDIR_E2E/zstd-age-restore"
mkdir -p "$ZSTD_AGE_BACKUP_DIR" "$ZSTD_AGE_RESTORE_DIR"

"$BINARY" backup \
  --source "$SOURCE_DIR" \
  --dest "$ZSTD_AGE_BACKUP_DIR" \
  --public-key "$AGE_RECIPIENT" \
  --encryption age \
  --compression zstd \
  --verbose

ZSTD_AGE_BACKUP_FILE=$(find "$ZSTD_AGE_BACKUP_DIR" -name "backup_*.tar.zst.age" -not -name "*.tmp" | head -1)
test -n "$ZSTD_AGE_BACKUP_FILE" || fail "No zstd+AGE backup file found (.tar.zst.age)"
test -s "$ZSTD_AGE_BACKUP_FILE" || fail "Zstd+AGE backup file is empty"

# Verify manifest records both compression and encryption
ZSTD_AGE_MANIFEST=$(find "$ZSTD_AGE_BACKUP_DIR" -name "*_manifest.json" | head -1)
test -n "$ZSTD_AGE_MANIFEST" || fail "No manifest for zstd+AGE backup"
grep -q '"compression": "zstd"' "$ZSTD_AGE_MANIFEST" || fail "Manifest should record compression as 'zstd'"
grep -q '"encryption": "age"' "$ZSTD_AGE_MANIFEST" || fail "Manifest should record encryption as 'age'"

pass "Zstd+AGE backup created: $(basename "$ZSTD_AGE_BACKUP_FILE")"

# --- Step 41: Zstd + AGE Quick Verify ---
step "Running quick verify on zstd+AGE backup"
"$BINARY" verify --file "$ZSTD_AGE_BACKUP_FILE" --quick
pass "Zstd+AGE quick verify passed"

# --- Step 42: Zstd + AGE Full Verify ---
step "Running full verify on zstd+AGE backup"
"$BINARY" verify --file "$ZSTD_AGE_BACKUP_FILE" --private-key "$AGE_KEY_FILE" --verbose
pass "Zstd+AGE full verify passed"

# --- Step 43: Zstd + AGE Restore ---
step "Running restore on zstd+AGE backup"
"$BINARY" restore \
  --file "$ZSTD_AGE_BACKUP_FILE" \
  --dest "$ZSTD_AGE_RESTORE_DIR" \
  --private-key "$AGE_KEY_FILE" \
  --verbose

ZSTD_AGE_RESTORED_SOURCE="$ZSTD_AGE_RESTORE_DIR/source"
test -d "$ZSTD_AGE_RESTORED_SOURCE" || fail "Zstd+AGE restored source directory not found"
pass "Zstd+AGE restore completed"

# --- Step 44: Zstd + AGE Diff ---
step "Comparing zstd+AGE restored data with source"
diff -r "$SOURCE_DIR" "$ZSTD_AGE_RESTORED_SOURCE" || fail "Zstd+AGE restored files differ from source"

if [ -L "$ZSTD_AGE_RESTORED_SOURCE/link_to_small.txt" ]; then
  TARGET=$(readlink "$ZSTD_AGE_RESTORED_SOURCE/link_to_small.txt")
  test "$TARGET" = "small.txt" || fail "Symlink target mismatch: got '$TARGET', want 'small.txt'"
  pass "Zstd+AGE: Symlink preserved correctly"
else
  fail "Zstd+AGE: Symlink not preserved"
fi

pass "Zstd+AGE: All files match source"

# ═══════════════════════════════════════════
# COMPRESSION LZ4 PIPELINE
# ═══════════════════════════════════════════

# --- Step 50: Compression Lz4 Backup ---
step "Running backup with --compression lz4"
LZ4_BACKUP_DIR="$TMPDIR_E2E/lz4-backups"
LZ4_RESTORE_DIR="$TMPDIR_E2E/lz4-restore"
mkdir -p "$LZ4_BACKUP_DIR" "$LZ4_RESTORE_DIR"

"$BINARY" backup \
  --source "$SOURCE_DIR" \
  --dest "$LZ4_BACKUP_DIR" \
  --public-key "$PUBLIC_KEY" \
  --compression lz4 \
  --verbose

LZ4_BACKUP_FILE=$(find "$LZ4_BACKUP_DIR" -name "backup_*.tar.lz4.gpg" -not -name "*.tmp" | head -1)
test -n "$LZ4_BACKUP_FILE" || fail "No lz4 backup file found (.tar.lz4.gpg)"
test -s "$LZ4_BACKUP_FILE" || fail "Lz4 backup file is empty"

# Verify manifest exists and records compression=lz4
LZ4_MANIFEST=$(find "$LZ4_BACKUP_DIR" -name "*_manifest.json" | head -1)
test -n "$LZ4_MANIFEST" || fail "No manifest for lz4 backup"
grep -q '"compression": "lz4"' "$LZ4_MANIFEST" || fail "Manifest should record compression as 'lz4'"

pass "Lz4 backup created: $(basename "$LZ4_BACKUP_FILE")"

# --- Step 51: Compression Lz4 Quick Verify ---
step "Running quick verify on lz4 backup"
"$BINARY" verify --file "$LZ4_BACKUP_FILE" --quick
pass "Lz4 quick verify passed"

# --- Step 52: Compression Lz4 Full Verify ---
step "Running full verify on lz4 backup"
"$BINARY" verify --file "$LZ4_BACKUP_FILE" --private-key "$PRIVATE_KEY" --verbose
pass "Lz4 full verify passed"

# --- Step 53: Compression Lz4 Restore ---
step "Running restore on lz4 backup"
"$BINARY" restore \
  --file "$LZ4_BACKUP_FILE" \
  --dest "$LZ4_RESTORE_DIR" \
  --private-key "$PRIVATE_KEY" \
  --verbose

LZ4_RESTORED_SOURCE="$LZ4_RESTORE_DIR/source"
test -d "$LZ4_RESTORED_SOURCE" || fail "Lz4 restored source directory not found"
pass "Lz4 restore completed"

# --- Step 54: Compression Lz4 Diff ---
step "Comparing lz4 restored data with source"
diff -r "$SOURCE_DIR" "$LZ4_RESTORED_SOURCE" || fail "Lz4 restored files differ from source"

if [ -L "$LZ4_RESTORED_SOURCE/link_to_small.txt" ]; then
  TARGET=$(readlink "$LZ4_RESTORED_SOURCE/link_to_small.txt")
  test "$TARGET" = "small.txt" || fail "Symlink target mismatch: got '$TARGET', want 'small.txt'"
  pass "Lz4: Symlink preserved correctly"
else
  fail "Lz4: Symlink not preserved"
fi

pass "Lz4: All files match source"

# ═══════════════════════════════════════════
# COMPRESSION LZ4 + AGE PIPELINE
# ═══════════════════════════════════════════

# --- Step 60: Lz4 + AGE Backup ---
step "Running backup with --compression lz4 --encryption age"
LZ4_AGE_BACKUP_DIR="$TMPDIR_E2E/lz4-age-backups"
LZ4_AGE_RESTORE_DIR="$TMPDIR_E2E/lz4-age-restore"
mkdir -p "$LZ4_AGE_BACKUP_DIR" "$LZ4_AGE_RESTORE_DIR"

"$BINARY" backup \
  --source "$SOURCE_DIR" \
  --dest "$LZ4_AGE_BACKUP_DIR" \
  --public-key "$AGE_RECIPIENT" \
  --encryption age \
  --compression lz4 \
  --verbose

LZ4_AGE_BACKUP_FILE=$(find "$LZ4_AGE_BACKUP_DIR" -name "backup_*.tar.lz4.age" -not -name "*.tmp" | head -1)
test -n "$LZ4_AGE_BACKUP_FILE" || fail "No lz4+AGE backup file found (.tar.lz4.age)"
test -s "$LZ4_AGE_BACKUP_FILE" || fail "Lz4+AGE backup file is empty"

# Verify manifest records both compression and encryption
LZ4_AGE_MANIFEST=$(find "$LZ4_AGE_BACKUP_DIR" -name "*_manifest.json" | head -1)
test -n "$LZ4_AGE_MANIFEST" || fail "No manifest for lz4+AGE backup"
grep -q '"compression": "lz4"' "$LZ4_AGE_MANIFEST" || fail "Manifest should record compression as 'lz4'"
grep -q '"encryption": "age"' "$LZ4_AGE_MANIFEST" || fail "Manifest should record encryption as 'age'"

pass "Lz4+AGE backup created: $(basename "$LZ4_AGE_BACKUP_FILE")"

# --- Step 61: Lz4 + AGE Quick Verify ---
step "Running quick verify on lz4+AGE backup"
"$BINARY" verify --file "$LZ4_AGE_BACKUP_FILE" --quick
pass "Lz4+AGE quick verify passed"

# --- Step 62: Lz4 + AGE Full Verify ---
step "Running full verify on lz4+AGE backup"
"$BINARY" verify --file "$LZ4_AGE_BACKUP_FILE" --private-key "$AGE_KEY_FILE" --verbose
pass "Lz4+AGE full verify passed"

# --- Step 63: Lz4 + AGE Restore ---
step "Running restore on lz4+AGE backup"
"$BINARY" restore \
  --file "$LZ4_AGE_BACKUP_FILE" \
  --dest "$LZ4_AGE_RESTORE_DIR" \
  --private-key "$AGE_KEY_FILE" \
  --verbose

LZ4_AGE_RESTORED_SOURCE="$LZ4_AGE_RESTORE_DIR/source"
test -d "$LZ4_AGE_RESTORED_SOURCE" || fail "Lz4+AGE restored source directory not found"
pass "Lz4+AGE restore completed"

# --- Step 64: Lz4 + AGE Diff ---
step "Comparing lz4+AGE restored data with source"
diff -r "$SOURCE_DIR" "$LZ4_AGE_RESTORED_SOURCE" || fail "Lz4+AGE restored files differ from source"

if [ -L "$LZ4_AGE_RESTORED_SOURCE/link_to_small.txt" ]; then
  TARGET=$(readlink "$LZ4_AGE_RESTORED_SOURCE/link_to_small.txt")
  test "$TARGET" = "small.txt" || fail "Symlink target mismatch: got '$TARGET', want 'small.txt'"
  pass "Lz4+AGE: Symlink preserved correctly"
else
  fail "Lz4+AGE: Symlink not preserved"
fi

pass "Lz4+AGE: All files match source"

# ═══════════════════════════════════════════
# MANIFEST-FIRST MANAGEMENT TESTS (issue #45)
# ═══════════════════════════════════════════

# --- Step 70: Managed vs Orphan list output ---
step "Testing managed vs orphan list output"
MFM_BACKUP_DIR="$TMPDIR_E2E/mfm-backups"
MFM_SOURCE_A="$TMPDIR_E2E/mfm-source-a"
MFM_SOURCE_B="$TMPDIR_E2E/mfm-source-b"
mkdir -p "$MFM_BACKUP_DIR" "$MFM_SOURCE_A" "$MFM_SOURCE_B"
echo "source A data" > "$MFM_SOURCE_A/file.txt"
echo "source B data" > "$MFM_SOURCE_B/file.txt"

# Create a managed backup (with manifest)
"$BINARY" backup \
  --source "$MFM_SOURCE_A" \
  --dest "$MFM_BACKUP_DIR" \
  --public-key "$PUBLIC_KEY"

# Create an orphan backup (without manifest)
"$BINARY" backup \
  --source "$MFM_SOURCE_B" \
  --dest "$MFM_BACKUP_DIR" \
  --public-key "$PUBLIC_KEY" \
  --skip-manifest

LIST_MFM_OUTPUT=$("$BINARY" list --dest "$MFM_BACKUP_DIR")

echo "$LIST_MFM_OUTPUT" | grep -q "Managed Backups" || fail "List output missing 'Managed Backups' section"
echo "$LIST_MFM_OUTPUT" | grep -q "Orphan Backups" || fail "List output missing 'Orphan Backups' section"
echo "$LIST_MFM_OUTPUT" | grep -q "Source:" || fail "Managed section should show Source:"
echo "$LIST_MFM_OUTPUT" | grep -q "(no manifest)" || fail "Orphan section should show '(no manifest)'"

pass "List correctly shows managed and orphan sections"

# --- Step 71: Orphan excluded from retention ---
step "Testing orphan excluded from retention"
ORFRET_BACKUP_DIR="$TMPDIR_E2E/orfret-backups"
ORFRET_SOURCE="$TMPDIR_E2E/orfret-source"
mkdir -p "$ORFRET_BACKUP_DIR" "$ORFRET_SOURCE"
echo "retention test data" > "$ORFRET_SOURCE/file.txt"

# Create 3 managed backups
for i in 1 2 3; do
  sleep 1
  "$BINARY" backup \
    --source "$ORFRET_SOURCE" \
    --dest "$ORFRET_BACKUP_DIR" \
    --public-key "$PUBLIC_KEY"
done

# Create 2 orphan backups
for i in 1 2; do
  sleep 1
  "$BINARY" backup \
    --source "$ORFRET_SOURCE" \
    --dest "$ORFRET_BACKUP_DIR" \
    --public-key "$PUBLIC_KEY" \
    --skip-manifest
done

# Record orphan filenames (backups without a matching manifest)
ORPHAN_FILES=""
for f in "$ORFRET_BACKUP_DIR"/backup_*.tar.gz.gpg; do
  base=$(basename "$f" .tar.gz.gpg)
  if [ ! -f "$ORFRET_BACKUP_DIR/${base}_manifest.json" ]; then
    ORPHAN_FILES="$ORPHAN_FILES $f"
  fi
done

# Verify we have orphans
test -n "$ORPHAN_FILES" || fail "No orphan files detected before retention"

# Apply retention: keep 2 — should only affect managed, leave orphans
RETENTION_OUTPUT=$("$BINARY" backup \
  --source "$ORFRET_SOURCE" \
  --dest "$ORFRET_BACKUP_DIR" \
  --public-key "$PUBLIC_KEY" \
  --retention 2 --verbose 2>&1)

# Verify stderr warnings about orphans
echo "$RETENTION_OUTPUT" | grep -q "skipping orphan" || fail "Expected orphan skip warnings"

# Verify all orphan files still exist (the key invariant)
for f in $ORPHAN_FILES; do
  test -f "$f" || fail "Orphan file $(basename "$f") was deleted by retention"
done

pass "Orphans correctly excluded from retention"

# --- Step 72: Scoped retention with two sources ---
step "Testing scoped retention with two sources"
SCOPED_BACKUP_DIR="$TMPDIR_E2E/scoped-backups"
SCOPED_SOURCE_A="$TMPDIR_E2E/scoped-src-a"
SCOPED_SOURCE_B="$TMPDIR_E2E/scoped-src-b"
mkdir -p "$SCOPED_BACKUP_DIR" "$SCOPED_SOURCE_A" "$SCOPED_SOURCE_B"
echo "source A" > "$SCOPED_SOURCE_A/file.txt"
echo "source B" > "$SCOPED_SOURCE_B/file.txt"

# Create 3 backups from source A
for i in 1 2 3; do
  sleep 1
  "$BINARY" backup \
    --source "$SCOPED_SOURCE_A" \
    --dest "$SCOPED_BACKUP_DIR" \
    --public-key "$PUBLIC_KEY"
done

# Create 3 backups from source B
for i in 1 2 3; do
  sleep 1
  "$BINARY" backup \
    --source "$SCOPED_SOURCE_B" \
    --dest "$SCOPED_BACKUP_DIR" \
    --public-key "$PUBLIC_KEY"
done

# Verify 6 total backups
SCOPED_BEFORE=$(find "$SCOPED_BACKUP_DIR" -name "backup_*.tar.gz.gpg" | wc -l | tr -d ' ')
test "$SCOPED_BEFORE" -eq 6 || fail "Expected 6 backups before scoped retention, got $SCOPED_BEFORE"

# Apply retention (keep 2) by making a new backup from source A
# This triggers retention which should keep 2 per (host, source)
sleep 1
"$BINARY" backup \
  --source "$SCOPED_SOURCE_A" \
  --dest "$SCOPED_BACKUP_DIR" \
  --public-key "$PUBLIC_KEY" \
  --retention 2 --verbose 2>&1

# After retention:
# Source A: 3 existing + 1 new = 4, keep 2 → delete 2 → 2 remain
# Source B: 3 existing, keep 2 → delete 1 → 2 remain
# Total: 4 backups, 4 manifests
SCOPED_AFTER=$(find "$SCOPED_BACKUP_DIR" -name "backup_*.tar.gz.gpg" | wc -l | tr -d ' ')
SCOPED_MANIFESTS=$(find "$SCOPED_BACKUP_DIR" -name "*_manifest.json" | wc -l | tr -d ' ')
test "$SCOPED_AFTER" -eq 4 || fail "Expected 4 backups after scoped retention (2+2), got $SCOPED_AFTER"
test "$SCOPED_MANIFESTS" -eq 4 || fail "Expected 4 manifests after scoped retention, got $SCOPED_MANIFESTS"

pass "Scoped retention correctly retains per (host, source)"

# --- Step 9: CLI error output behavior (#10) ---

# 9a: Runtime error should show single error, no usage
step "Testing runtime error output (no usage dump)"
ERR_OUTPUT=$("$BINARY" backup \
  --source "$SOURCE_DIR" \
  --dest "$BACKUP_DIR" \
  --public-key /nonexistent/key.asc 2>&1 || true)

# Error message should appear exactly once
ERR_COUNT=$(echo "$ERR_OUTPUT" | grep -c "Error:" || true)
test "$ERR_COUNT" -eq 1 || fail "Expected 1 'Error:' line, got $ERR_COUNT"

# Usage text should NOT appear
echo "$ERR_OUTPUT" | grep -q "Usage:" && fail "Runtime error should not show Usage:"
echo "$ERR_OUTPUT" | grep -q "Flags:" && fail "Runtime error should not show Flags:"

pass "Runtime error: single error, no usage dump"

# 9b: Missing required flag (Cobra-enforced) should show usage
step "Testing missing required flag output (usage shown)"
MISS_OUTPUT=$("$BINARY" backup --dest /tmp 2>&1 || true)

echo "$MISS_OUTPUT" | grep -q "required flag" || fail "Missing flag error not shown"
echo "$MISS_OUTPUT" | grep -q "Usage:" || fail "Missing flag should show Usage:"

pass "Missing required flag (Cobra): usage shown"

# 9c: Missing conditionally-required flag should show usage
step "Testing missing conditional flag output (usage shown)"
MANUAL_OUTPUT=$("$BINARY" verify --file "$BACKUP_FILE" 2>&1 || true)

echo "$MANUAL_OUTPUT" | grep -q "private-key" || fail "Missing --private-key error not shown"
echo "$MANUAL_OUTPUT" | grep -q "Usage:" || fail "Missing conditional flag should show Usage:"

# No partial success output should appear before the error
echo "$MANUAL_OUTPUT" | grep -q "Manifest:" && fail "Partial output shown before error"
echo "$MANUAL_OUTPUT" | grep -q "Checksum:" && fail "Partial output shown before error"

pass "Missing conditional flag (manual): usage shown, no partial output"

# 9d: --help should work
step "Testing --help output"
HELP_OUTPUT=$("$BINARY" backup --help 2>&1)

echo "$HELP_OUTPUT" | grep -q "Usage:" || fail "--help should show Usage:"
echo "$HELP_OUTPUT" | grep -q "Flags:" || fail "--help should show Flags:"

pass "--help works correctly"

# --- Summary ---
echo ""
printf "${GREEN}${BOLD}════════════════════════════════════════${NC}\n"
printf "${GREEN}${BOLD}  E2E PIPELINE TEST: ALL STEPS PASSED  ${NC}\n"
printf "${GREEN}${BOLD}════════════════════════════════════════${NC}\n"
