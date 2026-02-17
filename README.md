# secure-backup

**Secure, encrypted backups for any directory**

A high-performance backup tool written in Go that creates encrypted, compressed archives of directories. While it can backup Docker volumes, it's designed as a general-purpose secure backup solution for any use case.

## Features

- **Multiple Encryption**: GPG (RSA 4096-bit) and AGE (X25519) encryption
- **Flexible Compression**: Gzip (default) or none (passthrough for pre-compressed data)
- **Streaming Pipeline**: Efficient memory usage regardless of backup size
- **Backup Manifests**: Automatic checksum verification and metadata tracking
- **Retention Management**: Keep only the last N backups
- **Verify Integrity**: Quick and full verification modes
- **List Backups**: View all backups with age and size information
- **Production Hardened**: Atomic writes, backup locking, signal handling, secure defaults
- **Cross-platform**: Linux, macOS, Windows (amd64/arm64)
- **Release Packaging**: `.deb` packages, GitHub Releases, apt repository

## Installation

### Option 1: Download Pre-built Binary

Download from [GitHub Releases](https://github.com/icemarkom/secure-backup/releases/latest):

**Linux (amd64):**
```bash
wget https://github.com/icemarkom/secure-backup/releases/latest/download/secure-backup_*_linux_amd64.tar.gz
tar -xzf secure-backup_*_linux_amd64.tar.gz
sudo install -m 755 secure-backup /usr/local/bin/
```

**macOS (Universal):**
```bash
wget https://github.com/icemarkom/secure-backup/releases/latest/download/secure-backup_*_darwin_arm64.tar.gz
tar -xzf secure-backup_*_darwin_arm64.tar.gz
sudo install -m 755 secure-backup /usr/local/bin/
```

### Option 2: Install via apt (Debian/Ubuntu)

Linux users on Debian-based systems can install and keep up-to-date with releases via apt.

Releases are signed with my personal [GPG key](https://github.com/icemarkom/gpg-key/releases/latest/download/markom@gmail.com.asc).

**WARNING:** Most instructions on how to add third-party apt repositories on the Internet are inherently unsafe. What follows is my _suggested_ approach.

**Step 1:** Download GPG key to `/etc/apt/keyrings`
```bash
sudo mkdir -p /etc/apt/keyrings/
curl -Ls https://github.com/icemarkom/gpg-key/releases/latest/download/markom@gmail.com.asc \
  | sudo tee /etc/apt/keyrings/markom@gmail.com.asc
```

**Step 2:** Add repository to `/etc/apt/sources.list.d`
```bash
echo 'deb [signed-by=/etc/apt/keyrings/markom@gmail.com.asc] https://github.com/icemarkom/secure-backup/releases/latest/download/ /' \
  | sudo tee /etc/apt/sources.list.d/secure-backup.list
```

**Step 3:** Update apt and install
```bash
sudo apt update
sudo apt install secure-backup
```

### Option 3: Build from Source

**Requirements:**
- Go 1.26 or later
- make (optional, for Makefile targets)

```bash
git clone https://github.com/icemarkom/secure-backup.git
cd secure-backup

# Using Makefile (if make is installed)
make build
sudo make install

# Or using go directly
go build -o secure-backup .
sudo install -m 755 secure-backup /usr/local/bin/
```

## Building

```bash
# Build
make build

# Run tests
make test

# Coverage report
make coverage

# Clean artifacts
make clean

# Development workflow (fmt, vet, test, build)
make dev

# See all targets
make help
```

### Encryption Keys

secure-backup supports **GPG** (default) and **AGE** encryption.

#### GPG Keys

```bash
# Generate a new GPG key pair
gpg --full-generate-key

# Export your public key for backups
gpg --export yourname@example.com > ~/.gnupg/backup-pub.asc
```

#### AGE Keys

```bash
# Install age: https://github.com/FiloSottile/age
# Generate a new key pair
age-keygen -o key.txt

# The public key (recipient) is printed to stdout
# The private key (identity) is saved to key.txt
```

## Quick Start

### Output Behavior

**Silent by default** - `secure-backup` follows Unix philosophy:
- ✅ **Success**: Silent (exit code 0)
- ❌ **Errors**: Printed to stderr (exit code 1)
- 📝 **Details**: Add `--verbose` flag

### 1. Create Your First Backup with GPG

```bash
gpg --export yourname@example.com > ~/.gnupg/backup-pub.asc

secure-backup backup \
  --source /path/to/important/data \
  --dest /path/to/backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --retention 30
```

### 2. Or Use AGE Encryption

```bash
age-keygen -o key.txt
# Copy the public key (age1...) from the output

secure-backup backup \
  --source /path/to/important/data \
  --dest /path/to/backups \
  --encryption age \
  --public-key "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p" \
  --retention 30
```

### 3. Verbose Mode (See Progress)

```bash
secure-backup backup \
  --source /path/to/important/data \
  --dest /path/to/backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --retention 30 \
  --verbose
```

This creates two files:
- `backup_data_20260207_165000.tar.gz.gpg` - Encrypted backup (or `.tar.gz.age` for AGE)
- `backup_data_20260207_165000_manifest.json` - Manifest with checksum and metadata

### 4. Preview Operations (Dry-Run)

Preview what would happen without executing:

```bash
# Preview backup operation
secure-backup backup \
  --source /path/to/data \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --dry-run

# Output:
# [DRY RUN] Backup preview:
# [DRY RUN]   Source: /path/to/data (1.2 GiB)
# [DRY RUN]   Destination: /backups/backup_data_20260207_180500.tar.gz.gpg
# [DRY RUN]   Compression: gzip
# [DRY RUN]   Encryption: GPG
# [DRY RUN]
# [DRY RUN] Pipeline stages that would execute:
# [DRY RUN]   1. TAR - Archive source directory
# [DRY RUN]   2. COMPRESS - Compress with gzip
# [DRY RUN]   3. ENCRYPT - Encrypt with GPG
# [DRY RUN]   4. WRITE - Write to destination file
```

### 5. List Your Backups

```bash
secure-backup list --dest /path/to/backups
```

### 6. Verify Backup Integrity

```bash
# Quick check (fast)
secure-backup verify --file /path/to/backup.tar.gz.gpg --quick

# Full check (thorough)
secure-backup verify \
  --file /path/to/backup.tar.gz.gpg \
  --private-key ~/.gnupg/backup-priv.asc
```

### 7. Restore When Needed

```bash
# Restore to an empty or non-existent directory
secure-backup restore \
  --file /path/to/backup.tar.gz.gpg \
  --dest /path/to/restore/location \
  --private-key ~/.gnupg/backup-priv.asc

# Restore to a non-empty directory (requires --force)
secure-backup restore \
  --file /path/to/backup.tar.gz.gpg \
  --dest /path/to/existing/directory \
  --private-key ~/.gnupg/backup-priv.asc \
  --force
```

**Safety Feature**: The restore command will fail if the destination directory is not empty, unless you specify `--force`. This prevents accidental data loss.

**Note**: The `--force` flag does not delete existing files - it allows restore to proceed. Files with the same names will be overwritten, but other files remain untouched.

## Secure Passphrase Handling

Multiple secure options for providing GPG key passphrases. AGE keys do not use passphrases.

### Three Methods (in priority order)

#### 1. Environment Variable (Recommended for automation)

```bash
export SECURE_BACKUP_PASSPHRASE="your-secret-passphrase"
secure-backup restore \
  --file /backups/backup.tar.gz.gpg \
  --dest /restore \
  --private-key ~/.gnupg/backup-priv.asc
```

**Best for**: Cron jobs, scripts, CI/CD pipelines

#### 2. Passphrase File (Recommended for interactive use)

```bash
echo "your-secret-passphrase" > ~/.gpg-passphrase
chmod 600 ~/.gpg-passphrase  # CRITICAL - secure the file

secure-backup restore \
  --file /backups/backup.tar.gz.gpg \
  --dest /restore \
  --private-key ~/.gnupg/backup-priv.asc \
  --passphrase-file ~/.gpg-passphrase
```

**Best for**: Interactive use, personal backups

#### 3. Command Line Flag (NOT RECOMMENDED)

```bash
# ⚠️  WARNING: Insecure - visible in process lists and shell history
secure-backup restore \
  --file /backups/backup.tar.gz.gpg \
  --dest /restore \
  --private-key ~/.gnupg/backup-priv.asc \
  --passphrase "your-secret-passphrase"
  
# Security warning will be displayed:
# WARNING: Passphrase on command line is insecure and visible in process lists.
#          Use SECURE_BACKUP_PASSPHRASE environment variable or --passphrase-file instead.
```

**Only use when**: Rapid testing, throwaway keys

### Security Notes

- **Mutually exclusive**: Only one method can be used at a time
- **Priority order**: Flag → Environment Variable → File
- **File permissions**: Always use `chmod 600` on passphrase files
- **Keys without passphrases**: All methods work with empty passphrases (just omit all options)

## Commands

| Command | Description | Dry-Run Support |
|---------|-------------|-----------------|
| `backup` | Create an encrypted, compressed backup | ✅ Yes |
| `restore` | Restore files from a backup | ✅ Yes |
| `verify` | Verify backup integrity (quick or full) | ✅ Yes |
| `list` | List available backups with metadata | N/A |

All commands support `--dry-run` to preview operations without executing them.

See [USAGE.md](USAGE.md) for detailed documentation.

## Backup Manifests

Backups include manifest files for integrity verification.

### What Are Manifests?

Each backup creates two files:
- `backup_*.tar.gz.gpg` (or `.tar.gz.age`, `.tar.gpg`, `.tar.age`) - The encrypted backup
- `backup_*_manifest.json` - Manifest with checksum and metadata

**Manifest Contents:**
- SHA256 checksum of backup file
- Source path and backup timestamp
- Tool version and compression method
- File size and hostname

### Why Manifests Matter

✅ **Detect corruption** before restore  
✅ **Verify backups** without decryption  
✅ **Track metadata** for auditing

### Example Manifest

```json
{
  "created_at": "2026-02-14T18:30:00Z",
  "source_path": "/home/user/documents",
  "backup_file": "backup_documents_20260214_183000.tar.gz.gpg",
  "checksum_algorithm": "sha256",
  "checksum_value": "abc123def456...",
  "size_bytes": 523400000,
  "compression": "gzip",
  "encryption": "gpg",
  "created_by": {
    "tool": "secure-backup",
    "version": "v0.2.0",
    "hostname": "myserver"
  }
}
```

### Disabling Manifests

Use `--skip-manifest` to create backups without manifest files (not recommended for production):

```bash
secure-backup backup \
  --source /data \
  --dest /backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --skip-manifest
```

**When to skip:** Testing, ephemeral backups, or when every second counts.

## Architecture

### Critical Pipeline Order

The tool uses a **compress-before-encrypt** pipeline for optimal efficiency:

```
BACKUP:  Source → TAR → COMPRESS → ENCRYPT → File
RESTORE: File → DECRYPT → DECOMPRESS → EXTRACT → Destination
```

**Why this matters**: Encrypted data is cryptographically random and cannot be compressed. Reversing the order would result in 0% compression and 10x larger backups.

### Design Principles

- **Streaming Architecture**: Uses `io.Pipe` for constant memory usage
- **Interface-Based**: Easy to add new encryption/compression methods
- **Security First**: Path traversal protection, proper validation
- **Zero External Dependencies**: Core functionality uses only Go stdlib + `golang.org/x/crypto`

## Automated Backups with Cron

Add to your crontab (`crontab -e`):

```bash
# Daily backup at 2 AM, keep last 30 backups
0 2 * * * /usr/local/bin/secure-backup backup --source /data --dest /backups --public-key ~/.gnupg/backup-pub.asc --retention 30
```

## Troubleshooting

### Stale `.tmp` Files

If a backup is interrupted (Ctrl+C, power loss), you may see `.tmp` files:

```bash
$ ls /backups/
backup_data_20260214_120000.tar.gz.gpg
backup_data_20260214_120000.json
backup_data_20260214_130000.tar.gz.gpg.tmp  # ← Stale temp file
backup_data_20260214_130000.json.tmp         # ← Stale manifest temp file
```

**How to clean up:**
```bash
# Remove .tmp files manually
rm /backups/*.tmp

# Or use find to remove old temp files (older than 1 day)
find /backups -name "*.tmp" -mtime +1 -delete
```

**Why this happens:** Interrupted backups leave temp files. This is intentional - the tool only cleans up its own session files, not stale files from previous runs.

**Prevention:** Always let backups complete, or use process managers that send proper termination signals.

### Missing Manifest Files

If you see warnings about missing manifest files:

- **Cause**: Backup was created with `--skip-manifest` flag or from an older version
- **Impact**: Verify and restore will still work, just without checksum pre-validation
- **Solution**: Re-run backup without `--skip-manifest` to generate manifests

## Project Status

**Current Release**: v1.1.0 ✅

- ✅ All core commands implemented and tested
- ✅ GPG and AGE encryption support
- ✅ Gzip and none (passthrough) compression
- ✅ 60%+ unit test coverage on core modules
- ✅ Production hardened (P1-P19 resolved)
- ✅ Cross-platform builds and `.deb` packaging
- ✅ End-to-end pipeline test in CI (GPG + AGE)

## Performance

- **Memory Usage**: 10-50 MB regardless of backup size (streaming)
- **Compression Ratio**: 60-90% for text/logs, 0-5% for pre-compressed files
- **Speed**: ~20-100 MB/s backup, ~50-200 MB/s restore (hardware dependent)

## Security

- **Encryption**: GPG (RSA 4096-bit) or AGE (X25519) — both industry standard
- **Compression**: Applied before encryption (critical for efficiency)
- **Path Validation**: Protection against path traversal attacks
- **File Permissions**: Backup and manifest files default to `0600` (owner read/write only)
  - Override with `--file-mode=system` (use system umask) or `--file-mode=0640` (explicit octal)
  - Warning issued if permissions are world-readable

## Contributing

This project follows standard Go conventions. To contribute:

1. Run tests: `go test ./...`
2. Check coverage: `go test ./... -coverprofile=coverage.out`
3. Format code: `go fmt ./...`
4. Run linter: `golangci-lint run`

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Author

Marko Milivojevic (markom@gmail.com)
