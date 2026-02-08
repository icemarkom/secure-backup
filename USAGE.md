# User Guide - secure-backup

Comprehensive documentation for using secure-backup to create secure, encrypted backups.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Commands Reference](#commands-reference)
- [Common Use Cases](#common-use-cases)
- [GPG Key Management](#gpg-key-management)
- [Automated Backups](#automated-backups)
- [Troubleshooting](#troubleshooting)
- [Advanced Topics](#advanced-topics)

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
- `--verbose, -v`: Detailed output

**Examples:**

```bash
# Basic backup
secure-backup backup \
  --source /home/user/documents \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc

# Backup with 30-day retention
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
- `--verbose, -v`: Detailed output

**Examples:**

```bash
# Basic restore
secure-backup restore \
  --file /backups/backup_documents_20260207.tar.gz.gpg \
  --dest /restore/location \
  --private-key ~/.gnupg/backup-priv.asc

# Restore with passphrase from environment
export GPG_PASSPHRASE="your-passphrase"
secure-backup restore \
  --file /backups/backup_etc_20260207.tar.gz.gpg \
  --dest /tmp/etc-restore \
  --private-key /root/.gnupg/backup-priv.asc \
  --passphrase "$GPG_PASSPHRASE"
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
- `--verbose, -v`: Detailed output

**Examples:**

```bash
# Quick verification (no decryption)
secure-backup verify \
  --file /backups/backup_documents_20260207.tar.gz.gpg \
  --quick

# Full verification (decrypt + decompress everything)
secure-backup verify \
  --file /backups/backup_documents_20260207.tar.gz.gpg \
  --private-key ~/.gnupg/backup-priv.asc \
  --verbose
```

**Verification Modes:**

| Mode | Speed | What it checks | Private key needed? |
|------|-------|----------------|---------------------|
| Quick | <1s | File exists, has content | No |
| Full | Slower | Complete decrypt + decompress + tar validation | Yes |

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
# List all backups
secure-backup list --dest /backups

# List backups matching custom pattern
secure-backup list \
  --dest /backups \
  --pattern "backup_etc_*.tar.gz.gpg"
```

**Sample Output:**
```
Found 3 backup(s) in /backups:

Filename                                           Size       Age      Modified
──────────────────────────────────────────────────────────────────────────────
backup_documents_20260207_143000.tar.gz.gpg      125.3 MiB  2h       2026-02-07 14:30
backup_photos_20260206_140000.tar.gz.gpg         2.1 GiB    1d2h     2026-02-06 14:00
backup_etc_20260205_140000.tar.gz.gpg            45.2 MiB   2d2h     2026-02-05 14:00
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
  --retention 90 \
  --verbose

# Add to cron: 0 2 * * * /home/user/daily-backup.sh
```

### 2. Pre-Update System Snapshot

```bash
#!/bin/bash
# Before system updates, backup critical config

sudo secure-backup backup \
  --source /etc \
  --dest /root/backup-snapshots \
  --public-key /root/.gnupg/backup-pub.asc
```

### 3. Weekly Full Backups

```bash
#!/bin/bash
# weekly-full-backup.sh

BACKUP_PATHS=(
  "/home/user/Documents"
  "/home/user/Pictures"
  "/var/www"
)

for path in "${BACKUP_PATHS[@]}"; do
  secure-backup backup \
    --source "$path" \
    --dest /mnt/nas/weekly-backups \
    --public-key ~/.gnupg/backup-pub.asc \
    --retention 365
done

# Add to cron: 0 3 * * 0 /home/user/weekly-full-backup.sh
```

### 4. Database Backup Workflow

```bash
#!/bin/bash
# Backup a database dump

# 1. Create database dump
pg_dump mydb > /tmp/mydb-$(date +%Y%m%d).sql

# 2. Backup and encrypt the dump
secure-backup backup \
  --source /tmp/mydb-$(date +%Y%m%d).sql \
  --dest /backups/database \
  --public-key ~/.gnupg/backup-pub.asc \
  --retention 30

# 3. Clean up plaintext dump
rm /tmp/mydb-$(date +%Y%m%d).sql
```

### 5. Verify All Backups (Scheduled Check)

```bash
#!/bin/bash
# verify-backups.sh - Run weekly to ensure backup integrity

BACKUP_DIR="/backups"

for backup in "$BACKUP_DIR"/backup_*.tar.gz.gpg; do
  echo "Verifying: $backup"
  secure-backup verify --file "$backup" --quick
  
  if [ $? -eq 0 ]; then
    echo "✓ OK: $backup"
  else
    echo "✗ FAILED: $backup" | mail -s "Backup verification failed" admin@example.com
  fi
done

# Add to cron: 0 4 * * 1 /root/verify-backups.sh
```

## GPG Key Management

### Key Security Best Practices

1. **Protect Your Private Key**
   ```bash
   # Secure permissions
   chmod 600 ~/.gnupg/backup-priv.asc
   
   # Store backup on encrypted USB drive
   cp ~/.gnupg/backup-priv.asc /mnt/usb-backup/
   ```

2. **Use Strong Passphrases**
   - Minimum 20 characters
   - Use a password manager
   - Never store in plain text scripts

3. **Key Backup Strategy**
   - Keep private key in 3 locations (computer, USB, cloud backup)
   - Print paper backup of key fingerprint
   - Document recovery procedure

### Working with Multiple Keys

```bash
# List available keys
gpg --list-keys

# Use specific key for backup
gpg --export "Work Backup Key" > work-backup-pub.asc

secure-backup backup \
  --source /work-files \
  --dest /backups/work \
  --public-key work-backup-pub.asc
```

### Key Expiration

```bash
# Check key expiration
gpg --list-keys your@email.com

# Extend expiration if needed
gpg --edit-key your@email.com
# Then type: expire
```

## Automated Backups

### Crontab Examples

```bash
# Edit crontab
crontab -e

# Daily backup at 2 AM with 30-day retention
0 2 * * * /usr/local/bin/secure-backup backup --source /data --dest /backups --public-key ~/.gnupg/backup-pub.asc --retention 30

# Weekly backup on Sunday at 3 AM
0 3 * * 0 /usr/local/bin/secure-backup backup --source /home --dest /backups --public-key ~/.gnupg/backup-pub.asc --retention 365

# Hourly quick verification during business hours
0 9-17 * * 1-5 /usr/local/bin/secure-backup verify --file /backups/latest.tar.gz.gpg --quick
```

### Systemd Timer (Alternative to Cron)

Create `/etc/systemd/system/daily-backup.service`:
```ini
[Unit]
Description=Daily Encrypted Backup
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/secure-backup backup \
  --source /data \
  --dest /backups \
  --public-key /root/.gnupg/backup-pub.asc \
  --retention 30
User=root
```

Create `/etc/systemd/system/daily-backup.timer`:
```ini
[Unit]
Description=Daily Backup Timer

[Timer]
OnCalendar=daily
OnCalendar=02:00
Persistent=true

[Install]
WantedBy=timers.target
```

Enable:
```bash
sudo systemctl enable --now daily-backup.timer
sudo systemctl list-timers  # Verify
```

## Troubleshooting

### Common Issues

#### "Failed to load public keys"

**Cause**: Invalid or inaccessible public key file

**Solutions**:
```bash
# Verify key file exists
ls -l ~/.gnupg/backup-pub.asc

# Re-export key
gpg --export your@email.com > ~/.gnupg/backup-pub.asc

# Check file is readable
chmod 644 ~/.gnupg/backup-pub.asc
```

#### "Failed to decrypt private key"

**Cause**: Wrong passphrase or encrypted private key

**Solutions**:
```bash
# Test key manually
gpg --decrypt-files test-encrypted-file

# Use --passphrase flag
secure-backup restore \
  --file backup.tar.gz.gpg \
  --dest /restore \
  --private-key ~/.gnupg/backup-priv.asc \
  --passphrase "your-passphrase"
```

#### "Permission denied" during backup

**Cause**: Insufficient permissions to read source files

**Solutions**:
```bash
# Use sudo for system files
sudo secure-backup backup --source /etc --dest /backups --public-key ~/.gnupg/backup-pub.asc

# Fix source permissions
chmod -R +r /path/to/source
```

#### Backup file corruption

**Solutions**:
```bash
# Run full verification
secure-backup verify \
  --file suspicious-backup.tar.gz.gpg \
  --private-key ~/.gnupg/backup-priv.asc

# Check filesystem
fsck /dev/backup-partition

# Restore from older backup
secure-backup list --dest /backups
```

## Advanced Topics

### Backup Size Estimation

```bash
# Estimate uncompressed size
du -sh /path/to/source

# Estimate compressed size (rough)
tar -czf - /path/to/source | wc -c
```

### Custom Backup Naming

Backups are automatically named: `backup_{sourcename}_{timestamp}.tar.gz.gpg`

The timestamp format is: `YYYYMMDD_HHMMSS`

### Compression Efficiency

Different file types compress differently:

| File Type | Typical Compression |
|-----------|---------------------|
| Text files, logs | 85-95% |
| JSON, XML, YAML | 80-90% |
| Database files | 60-70% |
| Images (JPG, PNG) | 0-5% (already compressed) |
| Videos (MP4, etc) | 0-5% (already compressed) |

### Security Considerations

1. **Backup Storage Security**
   - Store backups on encrypted filesystems
   - Use separate physical drives
   - Consider off-site backups

2. **Key Management**
   - Rotate keys annually
   - Use hardware security modules for critical data
   - Implement key escrow for business continuity

3. **Passphrase Protection**
   - Never store passphrases in scripts
   - Use environment variables or prompt
   - Consider using GPG agent for automation

### Performance Tuning

The tool uses sensible defaults, but you can optimize for your use case:

- **Large files**: The streaming architecture handles any size efficiently
- **Many small files**: tar archiving is optimal for this
- **Network storage**: Consider local staging before remote backup
- **Resource limits**: Tool uses minimal RAM regardless of backup size

### Integration with Other Tools

```bash
# Backup, then sync to S3
secure-backup backup --source /data --dest /tmp/backups --public-key ~/.gnupg/backup-pub.asc
aws s3 sync /tmp/backups s3://mybucket/backups/

# Backup, then rsync to remote server
secure-backup backup --source /data --dest /tmp/backups --public-key ~/.gnupg/backup-pub.asc
rsync -av /tmp/backups/ remote-server:/backups/

# Combine with monitoring
secure-backup backup ... && echo "SUCCESS" | mail -s "Backup OK" admin@example.com
```

## Getting Help

- **Bug Reports**: [GitHub Issues](https://github.com/icemarkom/secure-backup/issues)
- **View Command Help**: `secure-backup [command] --help`
- **Test Coverage**: `go test ./... -v` in the source directory

## Future Features (Roadmap)

- Docker volume backup integration
- Alternative encryption methods (age)
- Additional compression algorithms (zstd, lz4)
- Progress indicators for large backups
- Parallel compression for multi-core systems
- Incremental backup support
