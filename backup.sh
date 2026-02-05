#!/bin/bash
set -o pipefail

# ==============================================================================
# Encrypted Backup Script
# ==============================================================================
#
# Usage:
#   ./backup.sh <SOURCE_DIR> <DEST_DIR> <GPG_RECIPIENT>
#
# Arguments:
#   SOURCE_DIR      The directory to back up.
#   DEST_DIR        The directory where the backup file will be saved.
#   GPG_RECIPIENT   The GPG email or key ID to encrypt the backup for.
#
# Restore:
#   gpg --decrypt <backup_file> | tar -xz -C <restore_dir>
#
# Description:
#   Creates a compressed, encrypted archive of the source directory.
#   - Silent on success (exit code 0).
#   - Prints errors to stderr on failure (non-zero exit code).
#   - Retains backups for 7 days.
#
# ==============================================================================

# Configuration
SOURCE_DIR="${1}"
DEST_DIR="${2}"
RECIPIENT="${3}"
RETENTION_DAYS=7

# Validation
if [[ -z "${SOURCE_DIR}" || -z "${DEST_DIR}" || -z "${RECIPIENT}" ]]; then
    echo "Usage: ${0} <source_dir> <dest_dir> <gpg_recipient_email_or_id>" >&2
    exit 1
fi

if [[ ! -d "${SOURCE_DIR}" ]]; then
    echo "Error: Source directory '${SOURCE_DIR}' does not exist." >&2
    exit 1
fi

if [[ ! -d "${DEST_DIR}" ]]; then
    mkdir -p "${DEST_DIR}"
fi

# Generate Filename
FOLDER_NAME=$(basename "$(realpath "${SOURCE_DIR}")")
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
FILENAME="backup_${FOLDER_NAME}_${TIMESTAMP}.tar.gz.gpg"
OUTPUT_PATH="${DEST_DIR}/${FILENAME}"

# Create Backup
# tar: c (create), z (gzip), f - (stdout to pipe)
# gpg: encrypts from stdin
if ! tar -czf - -C "$(dirname "$(realpath "${SOURCE_DIR}")")" "${FOLDER_NAME}" 2>/dev/null | \
    gpg --encrypt --recipient "${RECIPIENT}" --batch --yes --trust-model always --output "${OUTPUT_PATH}" 2>/dev/null; then
    echo "Error: Backup failed." >&2
    # Clean up partial file if it exists
    [[ -f "${OUTPUT_PATH}" ]] && rm -f "${OUTPUT_PATH}"
    exit 1
fi

# Cleanup Old Backups (Make before break: only runs if backup succeeded)
if [[ -d "${DEST_DIR}" ]]; then
    find "${DEST_DIR}" -name "backup_*.tar.gz.gpg" -type f -mtime "+${RETENTION_DAYS}" -delete
fi

exit 0
