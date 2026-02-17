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
