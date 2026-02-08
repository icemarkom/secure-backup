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
- `--verbose, -v`: Show progress and detailed output

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
```

**Output File Format:**
```
backup_{dirname}_{timestamp}.tar.gz.gpg
Example: backup_documents_20260207_165324.tar.gz.gpg
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

## Getting Help

- **Bug Reports**: [GitHub Issues](https://github.com/icemarkom/secure-backup/issues)
- **View Command Help**: `secure-backup [command] --help`
- **Verbose Mode**: Add `--verbose` to any command for detailed output
