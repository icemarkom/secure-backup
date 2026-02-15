# User Guide - secure-backup

Comprehensive documentation for using secure-backup to create secure, encrypted backups.

## Table of Contents

- [Output Behavior](#output-behavior)
- [Prerequisites](#prerequisites)
- [Commands Reference](#commands-reference)
- [Common Use Cases](#common-use-cases)
- [GPG Key Management](#gpg-key-management)
- [Automated Backups](#automated-backups)
- [Troubleshooting](#troubleshooting)
- [Advanced Topics](#advanced-topics)

## Output Behavior

**secure-backup follows Unix philosophy: silent success, errors to stderr.**

### Default Behavior (Silent)

By default, all commands are **silent on success**. Only the exit code indicates success or failure:

```bash
$ secure-backup backup --source ~/docs --dest /backups --public-key key.asc
$ echo $?
0  # Success

$ secure-backup backup --source /missing --dest /backups --public-key key.asc
Error: invalid source path: stat /missing: no such file or directory
$ echo $?
1  # Failure
```

**Why silent by default?**
- Script-friendly: `secure-backup backup ... && next-command` works cleanly
- Cron-safe: No noise in cron logs on success
- Errors are always visible (written to stderr)
- Add `--verbose` when you want details

### Verbose Mode

Add `--verbose` (or `-v`) to any command to see progress bars and status messages:

```bash
$ secure-backup backup --source ~/docs --dest /backups --public-key key.asc --verbose
Starting backup of /home/user/docs (1.2 GiB)
Destination: /backups/backup_docs_20260207_180500.tar.gz.gpg
Backup completed successfully: /backups/backup_docs_20260207_180500.tar.gz.gpg
Backup size: 450 MiB
```

**Note**: Progress bars (if enabled in future) will write to stderr, status messages to stdout.

### List Command

The `list` command always shows output since it's a query operation (not a mutation):

```bash
$ secure-backup list --dest /backups
Found 3 backup(s) in /backups:
...
```

---

## Prerequisites

### GPG Keys

You need GPG keys for encryption. If you don't have them:

```bash
# Generate a new key pair
gpg --gen-key

# Follow the prompts:
# - Choose RSA and RSA
# - Key size: 4096
# - Expiration: 0 (doesn't expire) or your preference
# - Enter your name and email
# - Set a strong passphrase
```

### Export Your Keys

```bash
# Export public key (for backups)
gpg --export your@email.com > ~/.gnupg/backup-pub.asc

# Export private key (keep this SECRET!)
gpg --export-secret-keys your@email.com > ~/.gnupg/backup-priv.asc

# Secure the private key
chmod 600 ~/.gnupg/backup-priv.asc
```

## Commands Reference

### backup - Create Encrypted Backups

**Syntax:**
```bash
secure-backup backup [flags]
```

**Flags:**
- `--source` (required): Directory to backup
- `--dest` (required): Where to save backup files
- `--public-key` (required): Path to GPG public key
- `--encryption`: Encryption method (default: "gpg")
- `--retention`: Delete backups older than N days (default: 0 = keep all)
- `--skip-manifest`: Disable manifest generation (not recommended)
- `--verbose, -v`: Show progress and detailed output
- `--dry-run`: Preview operation without creating files

**Examples:**

```bash
# Basic backup (silent)
secure-backup backup \
  --source /home/user/documents \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc

# Backup with verbose output and 30-day retention
secure-backup backup \
  --source /var/www/html \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --retention 30 \
  --verbose

# Backup system configuration
sudo secure-backup backup \
  --source /etc \
  --dest /root/backups \
  --public-key /root/.gnupg/backup-pub.asc

# Preview backup before executing (dry-run)
secure-backup backup \
  --source /home/user/documents \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --dry-run \
  --verbose
```

**Output File Format:**
```
backup_{dirname}_{timestamp}.tar.gz.gpg  # Encrypted backup
backup_{dirname}_{timestamp}.json        # Manifest file
Example: backup_documents_20260207_165324.tar.gz.gpg
         backup_documents_20260207_165324.json
```

#### Manifest Files

By default, backups create manifest files for integrity verification:

```bash
# Default behavior - creates both files
secure-backup backup \
  --source /data \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc

# Result:
# /backups/backup_data_20260207_165000.tar.gz.gpg
# /backups/backup_data_20260207_165000.json  ← Manifest
```

**Manifest contents:**
- SHA256 checksum of backup file
- Source path and timestamp
- Tool version and settings
- File size and hostname

**Skip manifests** (not recommended for production):
```bash
secure-backup backup \
  --source /data \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --skip-manifest  # No manifest file created
```

### restore - Restore from Backups

**Syntax:**
```bash
secure-backup restore [flags]
```

**Flags:**
- `--file` (required): Backup file to restore
- `--dest` (required): Where to extract files
- `--private-key` (required): Path to GPG private key
- `--passphrase`: GPG key passphrase (or enter when prompted)
- `--verbose, -v`: Show progress and detailed output
- `--dry-run`: Preview operation without extracting files

**Examples:**

```bash
# Basic restore (silent)
secure-backup restore \
  --file /backups/backup_documents_20260207.tar.gz.gpg \
  --dest /restore/location \
  --private-key ~/.gnupg/backup-priv.asc

# Restore with verbose output
secure-backup restore \
  --file /backups/backup_etc_20260207.tar.gz.gpg \
  --dest /tmp/etc-restore \
  --private-key /root/.gnupg/backup-priv.asc \
  --verbose
```

**Important Notes:**
- Files are restored into a subdirectory named after the original source
- Original permissions and timestamps are preserved
- Symlinks are preserved as symlinks (not followed)

### verify - Check Backup Integrity

**Syntax:**
```bash
secure-backup verify [flags]
```

**Flags:**
- `--file` (required): Backup file to verify
- `--quick`: Fast check (header validation only)
- `--private-key`: GPG private key (required for full verify)
- `--passphrase`: GPG key passphrase
- `--verbose, -v`: Show detailed output
- `--dry-run`: Preview operation without performing verification

**Examples:**

```bash
# Quick verification (silent on success)
secure-backup verify \
  --file /backups/backup_documents_20260207.tar.gz.gpg \
  --quick

# Full verification with verbose output
secure-backup verify \
  --file /backups/backup_documents_20260207.tar.gz.gpg \
  --private-key ~/.gnupg/backup-priv.asc \
  --verbose

# Preview what verification would check (dry-run)
secure-backup verify \
  --file /backups/backup_documents_20260207.tar.gz.gpg \
  --private-key ~/.gnupg/backup-priv.asc \
  --dry-run \
  --verbose
```

#### Manifest-Based Verification

If a manifest file exists, verification automatically uses it:

```bash
# With manifest - verifies checksum first
secure-backup verify \
  --file /backups/backup_data_20260207.tar.gz.gpg \
  --private-key ~/.gnupg/backup-priv.asc

# Steps:
# 1. Check for manifest file (backup_data_20260207.json)
# 2. Verify SHA256 checksum against manifest
# 3. Perform full decryption and integrity check
```

**Without manifest** (--skip-manifest was used during backup):
```bash
# No checksum pre-validation, goes straight to decryption
secure-backup verify \
  --file /backups/backup_data_20260207.tar.gz.gpg \
  --private-key ~/.gnupg/backup-priv.asc
```

### list - View Available Backups

**Syntax:**
```bash
secure-backup list [flags]
```

**Flags:**
- `--dest` (required): Backup directory to list
- `--pattern`: Filename pattern (default: `backup_*.tar.gz.gpg`)

**Examples:**

```bash
# List all backups (always shows output)
secure-backup list --dest /backups

# List backups matching custom pattern
secure-backup list \
  --dest /backups \
  --pattern "backup_etc_*.tar.gz.gpg"
```

**Output includes manifest information when available:**

```bash
$ secure-backup list --dest /backups

Found 3 backup(s) in /backups:

backup_data_20260207_120000.tar.gz.gpg
  Age: 2 days
  Size: 1.2 GiB
  Manifest: ✓ (checksum: abc123...)

backup_docs_20260206_120000.tar.gz.gpg
  Age: 3 days
  Size: 450 MiB
  Manifest: ✓ (checksum: def456...)

backup_old_20260201_120000.tar.gz.gpg
  Age: 8 days
  Size: 800 MiB
  Manifest: ✗ (missing)
```

## Dry-Run Mode

The `--dry-run` flag allows you to preview what an operation would do without actually executing it. This is useful for:

- **Testing configurations** before running automated backups
- **Verifying paths and settings** are correct
- **Estimating backup sizes** and understanding what will be processed
- **Checking retention policies** to see which files would be deleted
- **Debugging issues** without making changes to the filesystem

### How It Works

When `--dry-run` is enabled:

1. **No files are created, modified, or deleted**
2. **Minimal validation** is performed (e.g., checking if source exists)
3. **Output is prefixed** with `[DRY RUN]` for clarity
4. **Expected paths and sizes** are shown
5. **Pipeline stages are always listed** (dry-run automatically implies verbose output)

> **Note:** The `--dry-run` flag automatically enables verbose output to provide a useful preview. You don't need to specify `--verbose` separately.

### Examples

**Preview a backup:**
```bash
secure-backup backup \
  --source /home/user/documents \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --dry-run

# Output:
# [DRY RUN] Backup preview:
# [DRY RUN]   Source: /home/user/documents (1.2 GiB)
# [DRY RUN]   Destination: /backups/backup_documents_20260207_180500.tar.gz.gpg
# [DRY RUN]   Compression: gzip
# [DRY RUN]   Encryption: GPG
# [DRY RUN]
# [DRY RUN] Pipeline stages that would execute:
# [DRY RUN]   1. TAR - Archive source directory
# [DRY RUN]   2. COMPRESS - Compress with gzip
# [DRY RUN]   3. ENCRYPT - Encrypt with GPG
# [DRY RUN]   4. WRITE - Write to destination file
```

**Preview a restore:**
```bash
secure-backup restore \
  --file /backups/backup_documents_20260207.tar.gz.gpg \
  --dest /restore/location \
  --private-key ~/.gnupg/backup-priv.asc \
  --dry-run

# Output:
# [DRY RUN] Restore preview:
# [DRY RUN]   Backup file: /backups/backup_documents_20260207.tar.gz.gpg (523.4 MiB)
# [DRY RUN]   Destination: /restore/location
# [DRY RUN]
# [DRY RUN] Pipeline stages that would execute:
# [DRY RUN]   1. DECRYPT - Decrypt backup file with GPG
# [DRY RUN]   2. DECOMPRESS - Decompress with gzip
# [DRY RUN]   3. EXTRACT - Extract tar archive to destination
```

**Preview retention policy:**
```bash
secure-backup backup \
  --source /data \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --retention 30 \
  --dry-run

# Output includes:
# [DRY RUN] Would delete: backup_data_20240101_120000.tar.gz.gpg (37 days old)
# [DRY RUN] Would delete: backup_data_20240105_120000.tar.gz.gpg (33 days old)
# [DRY RUN] Would delete 2 old backup(s)
```

**Preview verification:**
```bash
secure-backup verify \
  --file /backups/backup_documents_20260207.tar.gz.gpg \
  --private-key ~/.gnupg/backup-priv.asc \
  --dry-run

# Output:
# [DRY RUN] Verify preview:
# [DRY RUN]   Backup file: /backups/backup_documents_20260207.tar.gz.gpg (523.4 MiB)
# [DRY RUN]   Mode: Full verification (decrypt + decompress)
# [DRY RUN]
# [DRY RUN] Full verification would:
# [DRY RUN]   1. DECRYPT - Decrypt with GPG
# [DRY RUN]   2. DECOMPRESS - Decompress with gzip
# [DRY RUN]   3. VERIFY - Read entire archive to verify integrity
```

### Best Practices

1. **Always test new configurations** with `--dry-run` first
2. **Use dry-run to preview operations** - detailed output is automatic
3. **Verify retention policies** before enabling them in production
4. **Check paths and sizes** match your expectations
5. **Test restore operations** before you need them in an emergency

---

## Common Use Cases

### 1. Daily Document Backups

```bash
#!/bin/bash
# daily-backup.sh

secure-backup backup \
  --source "$HOME/Documents" \
  --dest /mnt/backup-drive/daily \
  --public-key "$HOME/.gnupg/backup-pub.asc" \
  --retention 90

# Silent on success - only errors will be reported
# Add to cron: 0 2 * * * /home/user/daily-backup.sh
```

### 2. Automated Backups with Notifications

```bash
#!/bin/bash
# backup-with-notification.sh

if secure-backup backup \
  --source /data \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --retention 30; then
  echo "Backup completed successfully" | mail -s "Backup OK" admin@example.com
else
  echo "Backup FAILED - check logs" | mail -s "Backup FAILED" admin@example.com
fi
```

### 3. Verify All Backups (Scheduled Check)

```bash
#!/bin/bash
# verify-backups.sh - Run weekly to ensure backup integrity

BACKUP_DIR="/backups"

for backup in "$BACKUP_DIR"/backup_*.tar.gz.gpg; do
  if secure-backup verify --file "$backup" --quick; then
    echo "✓ OK: $backup"
  else
    echo "✗ FAILED: $backup" | mail -s "Backup verification failed" admin@example.com
  fi
done
```

## File Management

### Backup Files and Manifests

Each backup creates two files:

| File | Purpose | Required |
|------|---------|----------|
| `backup_*.tar.gz.gpg` | Encrypted backup data | Yes |
| `backup_*.json` | Manifest with checksum | Recommended |

**Both files should be kept together** for optimal reliability.

### Temporary Files

During backup creation, you may briefly see `.tmp` files:

```bash
/backups/backup_data_20260207_120000.tar.gz.gpg.tmp  # During write
/backups/backup_data_20260207_120000.json.tmp        # During write
```

**Normal behavior:**
- ✅ `.tmp` files are automatically removed on success
- ✅ `.tmp` files are removed on error (same session)
- ⚠️  Interrupted backups (Ctrl+C, power loss) may leave stale `.tmp` files

**Cleaning up stale files:**

```bash
# Safe to remove old .tmp files
find /backups -name "*.tmp" -mtime +1 -delete

# Or manually
rm /backups/*.tmp
```

### Retention and Cleanup

The retention policy (--retention flag) automatically deletes old backups:

```bash
secure-backup backup \
  --source /data \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --retention 30  # Delete backups older than 30 days
```

**What gets deleted:**
- ✅ Old `.tar.gz.gpg` backup files
- ✅ Corresponding `.json` manifest files
- ❌ `.tmp` files (you must clean these manually)

## Automated Backups

### Crontab Examples

```bash
# Edit crontab
crontab -e

# Daily backup at 2 AM (silent - only reports errors)
0 2 * * * /usr/local/bin/secure-backup backup --source /data --dest /backups --public-key ~/.gnupg/backup-pub.asc --retention 30

# Weekly backup with verbose logging
0 3 * * 0 /usr/local/bin/secure-backup backup --source /home --dest /backups --public-key ~/.gnupg/backup-pub.asc --retention 365 --verbose >> /var/log/backups.log 2>&1
```

## Troubleshooting

### Exit Codes

- `0` = Success
- `1` = Failure (check stderr for error message)

### Common Issues

#### Silent Mode - How to Debug

If a backup fails silently in cron:

1. **Check exit code**: The exit code will be non-zero on failure
2. **Check stderr**: Errors are always written to stderr
3. **Add verbose mode temporarily**: Use `--verbose` to see what's happening
4. **Check cron logs**: `grep CRON /var/log/syslog`

#### Command Appears to Do Nothing

If you run a command and see no output:

- **This is normal!** Silent on success is the default
- Check exit code: `echo $?` should show `0` for success
- Use `--verbose` flag to see progress and status messages

For all other troubleshooting, see the full documentation.

#### Stale .tmp Files After Interruption

**Problem:** Backup was interrupted and `.tmp` files remain.

**Cause:** The backup process was killed (Ctrl+C, system crash, power loss) before it could clean up.

**Solution:**
1. Verify the interruption: Check if a final backup file exists
2. If no final file exists, the backup failed
3. Manually remove `.tmp` files:
   ```bash
   rm /backups/*.tmp
   
   # Or use find to remove old temp files
   find /backups -name "*.tmp" -mtime +1 -delete
   ```

**Prevention:** Use proper shutdown procedures and ensure backups complete.

#### Manifest File Missing

**Problem:** Verify or restore warns about missing manifest.

**Cause:** Backup was created with `--skip-manifest` flag or from an older version.

**Solution:** This is expected if manifests were disabled. Operations will still work, just without checksum pre-validation. To generate manifests for future backups, omit the `--skip-manifest` flag.

## Getting Help

- **Bug Reports**: [GitHub Issues](https://github.com/icemarkom/secure-backup/issues)
- **View Command Help**: `secure-backup [command] --help`
- **Verbose Mode**: Add `--verbose` to any command for detailed output
