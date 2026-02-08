# secure-backup

**Secure, encrypted backups for any directory**

A high-performance backup tool written in Go that creates encrypted, compressed archives of directories. While it can backup Docker volumes, it's designed as a general-purpose secure backup solution for any use case.

## Features

✅ **Phase 1 - Core Functionality (COMPLETE)**
- **GPG Encryption**: Secure backups using your existing GPG keys
- **Gzip Compression**: 60-80% size reduction for most data
- **Streaming Pipeline**: Efficient memory usage regardless of backup size
- **Retention Management**: Automatic cleanup of old backups
- **Verify Integrity**: Quick and full verification modes
- **List Backups**: View all backups with age and size information

🎯 **Phase 2 - Build Platform Support (NEXT)**
- Cross-platform builds (Linux, macOS, Windows)
- Makefile or Bazel build system
- Release packaging (deb, rpm, tar.gz)
- Version embedding in binary

🔮 **Phase 3+ - Future Enhancements**  
- Additional encryption methods (age)
- Advanced compression algorithms (zstd, lz4)
- User experience improvements (progress bars, config files)
- Remote storage backends (S3, SFTP, rsync)

💡 **Optional Future Feature**
- Docker volume integration (specialty use case)

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
- Go 1.21 or later
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

### GPG Keys Required

Before using secure-backup, you'll need GPG keys for encryption:

```bash
# Generate a new GPG key pair
gpg --full-generate-key

# Export your public key for backups
gpg --export yourname@example.com > ~/.gnupg/backup-pub.asc
```

## Quick Start

### Output Behavior

**Silent by default** - `secure-backup` follows Unix philosophy:
- ✅ **Success**: Silent (exit code 0)
- ❌ **Errors**: Printed to stderr (exit code 1)
- 📝 **Details**: Add `--verbose` flag

### 1. Export Your GPG Public Key

```bash
gpg --export yourname@example.com > ~/.gnupg/backup-pub.asc
```

### 2. Create Your First Backup (Silent Mode)

```bash
secure-backup backup \
  --source /path/to/important/data \
  --dest /path/to/backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --retention 30

# Check if it worked
echo $?  # 0 = success
```

### 3. Verbose Mode (See Progress)

```bash
secure-backup backup \
  --source /path/to/important/data \
  --dest /path/to/backups \
  --public-key ~/.gnupg/backup-pub.asc \
  --retention 30 \
  --verbose

# Output shows:
# Starting backup of /path/to/important/data (1.2 GiB)
# Destination: /path/to/backups/backup_data_20260207_180500.tar.gz.gpg
# Backup completed successfully
```

This creates an encrypted backup like: `backup_data_20260207_165000.tar.gz.gpg`

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
secure-backup restore \
  --file /path/to/backup.tar.gz.gpg \
  --dest /path/to/restore/location \
  --private-key ~/.gnupg/backup-priv.asc
```

## Commands

| Command | Description | Dry-Run Support |
|---------|-------------|-----------------|
| `backup` | Create an encrypted, compressed backup | ✅ Yes |
| `restore` | Restore files from a backup | ✅ Yes |
| `verify` | Verify backup integrity (quick or full) | ✅ Yes |
| `list` | List available backups with metadata | N/A |

All commands support `--dry-run` to preview operations without executing them.

See [USAGE.md](USAGE.md) for detailed documentation.

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
# Daily backup at 2 AM with 30-day retention
0 2 * * * /usr/local/bin/secure-backup backup --source /data --dest /backups --public-key ~/.gnupg/backup-pub.asc --retention 30
```

## Project Status

**Current State**: Phase 1 Complete ✅

- ✅ All core commands implemented and tested
- ✅ 60%+ unit test coverage on core modules
- ✅ Fully functional for directory backups
- ✅ Production-ready for non-Docker use cases

**Next Steps**: Phase 2 - Build Platform Support

See [implementation_plan.md](.gemini/antigravity/brain/*/implementation_plan.md) for detailed architectural decisions.

## Performance

- **Memory Usage**: 10-50 MB regardless of backup size (streaming)
- **Compression Ratio**: 60-90% for text/logs, 0-5% for pre-compressed files
- **Speed**: ~20-100 MB/s backup, ~50-200 MB/s restore (hardware dependent)

## Security

- **Encryption**: RSA 4096-bit GPG keys (industry standard)
- **Compression**: Applied before encryption (critical for efficiency)
- **Path Validation**: Protection against path traversal attacks
- **Permissions**: Preserved during backup/restore

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
