# secure-backup: Agent Continuation Guide

> **⚠️ CRITICAL**: Keep this file up-to-date with every significant change.
> This is the definitive blueprint for continuing work on this project.

---

## Project Identity

**Name**: secure-backup  
**Module**: `github.com/icemarkom/secure-backup`  
**Binary**: `secure-backup`  
**License**: Apache 2.0  
**Author**: Marko Milivojevic (markom@gmail.com)  
**Language**: Go 1.21+

**Mission**: Production-ready tool for secure, encrypted backups of any directory.

---

## Current Status

### ✅ Phase 1: Core Functionality (COMPLETE)

**Commands:**
- `backup` - TAR→COMPRESS→ENCRYPT pipeline
- `restore` - DECRYPT→DECOMPRESS→EXTRACT pipeline
- `verify` - Integrity checking (quick & full modes)
- `list` - View available backups
- `version` - Show version info

**Features:**
- GPG encryption (RSA 4096-bit)
- Gzip compression (level 6, ~60-80% reduction)
- Streaming architecture (constant 10-50MB memory)
- Retention management (auto-cleanup old backups)
- Path traversal protection

### ✅ Phase 2: Build & Release (COMPLETE)

**Build System:**
- Makefile with dev targets (build, test, coverage, lint, etc.)
- Version embedding from git tags
- Self-documenting help

**Release Automation:**
- GoReleaser configuration
- Multi-platform builds (linux/darwin/windows × amd64/arm64)
- .deb package generation
- GitHub Actions CI/CD
- apt repository generation

**Documentation:**
- README.md with 3 installation methods
- USAGE.md detailed guide
- CONTRIBUTING.md developer guide

### ✅ Phase 5: User Experience (COMPLETE)

**Phase 5.1: Silent by Default + Progress** ✅ (2026-02-08)
- `internal/progress` package (ProgressReader/Writer)
- Unix philosophy: silent success, errors to stderr
- `--verbose` flag for progress and details
- Test coverage: 90% for progress package

**Phase 5.1b: Test Coverage Improvements** ✅ (2026-02-07)
- Comprehensive unit + integration tests
- Coverage: 58.9% → 83.2% for backup package
- Real GPG integration tests (encrypt: 29% → 68%)
- Standardized want/got test nomenclature

**Phase 5.2: Better Error Messages** ✅ (2026-02-08)
- `internal/errors` package with `UserError` type
- User-friendly messages with actionable hints
- 100% coverage for errors package
- Example: "File not found: /path\nHint: Check that the path exists..."

**Phase 5.3: Dry-Run Mode** ✅ (2026-02-08)
- `--dry-run` flag for backup, restore, verify
- Preview operations without file modifications
- **Dry-run automatically implies verbose**
- Retention policy dry-run support
- Output prefixed with `[DRY RUN]`

---

---

## Productionization Effort

> **Goal**: Harden the tool for production use with mission-critical data  
> **Philosophy**: Simplicity over features. No config files. Unattended operation with security.  
> **Current Trust Score**: 7.5/10 - P1 complete, significant improvement in data integrity  
> **Target**: 9/10 - Production-ready for mission-critical backups

### Critical Issues (Must Fix)

#### P1: Backup Metadata & Verification ✅ COMPLETE (2026-02-14)

**Problem**: Backups lack metadata and verification. Corruption goes undetected until restore fails.

**Status**: ✅ **IMPLEMENTED** (2026-02-14)

**What Was Delivered**:
- ✅ `internal/manifest` package with 93.2% test coverage
- ✅ SHA256 checksum computation using streaming I/O
- ✅ JSON manifest files created by default alongside backups
- ✅ Manifest validation in restore and verify commands
- ✅ Metadata display in list and verify commands
- ✅ `--skip-manifest` flag for backward compatibility

**Decision: Unified Manifest with Checksum (Default Enabled)**
- Create `backup_*.manifest.json` alongside backup **by default**
- **Add `--skip-manifest` flag to disable** for simple/fast backups
- Manifest includes SHA256 checksum of encrypted backup
- Contains: source, timestamp, version, settings, checksum
- Optionally: quick decrypt test of first 1KB
- Estimated effort: 1-2 days

**Manifest Format**:
```json
{
  "version": "1.0.0",
  "created": "2026-02-08T22:30:00Z",
  "source": "/path/to/source",
  "backup_file": "backup_source_20260208_223000.tar.gz.gpg",
  "checksum": {
    "algorithm": "sha256",
    "value": "abc123..."
  },
  "compression": "gzip",
  "encryption": "gpg",
  "tool_version": "v0.1.0"
}
```

**Implementation Details**:
- Package: `internal/manifest` with `Manifest` and `CreatedBy` types
- Functions: `New()`, `Write()`, `Read()`, `Validate()`, `ComputeChecksum()`, `ValidateChecksum()`
- Commands updated: `backup`, `restore`, `verify`, `list`
- All commands support `--skip-manifest` flag
- Graceful error handling: manifest generation warns but doesn't fail backup
- Test coverage: 93.2% for manifest package, overall coverage maintained at 59.7%

**Files Modified**:
- New: `internal/manifest/manifest.go`, `internal/manifest/manifest_test.go`
- Modified: `cmd/backup.go`, `cmd/restore.go`, `cmd/verify.go`, `cmd/list.go`, `cmd/root.go`

**Impact**:
- Trust score: 6.5/10 → 7.5/10
- Corruption detection now possible before restore
- Metadata tracking for audit and troubleshooting
- Production-ready integrity verification

---

#### P2: Atomic Backup Writes ✅ COMPLETE (2026-02-14)

**Problem**: Partial backup files created on failure. Race condition in error cleanup.

**Status**: ✅ **IMPLEMENTED** (2026-02-14)

**What Was Delivered**:
- ✅ Atomic write pattern using temp file + rename
- ✅ Fixed race condition in error cleanup
- ✅ Applied to both backup and manifest files
- ✅ 3 new tests added (backup: 2, manifest: 1)
- ✅ Coverage maintained (backup: 83.7%, manifest: 89.6%)

**Implementation**:
- Write to `backup_*.tar.gz.gpg.tmp` during backup
- Write to `manifest.json.tmp` during manifest creation
- `os.Rename(tmpPath, finalPath)` on success (atomic)
- Delete `.tmp` file only on error (current session)
- User manually cleans up stale `.tmp` files from interrupted backups

**Files Modified**:
- Modified: `internal/backup/backup.go` - Atomic write for backup files
- Modified: `internal/manifest/manifest.go` - Atomic write for manifest files
- Modified: `internal/backup/backup_test.go` - Added atomic write tests
- Modified: `internal/manifest/manifest_test.go` - Added atomic write tests

**Tests Added**:
- `TestPerformBackup_NoTempFilesOnSuccess` - Verifies no `.tmp` files after success
- `TestPerformBackup_TempFileCleanupOnError` - Verifies cleanup on error
- `TestWrite_NoTempFilesOnSuccess` - Verifies manifest atomic write

**Impact**:
- No partial files on failure (reliability)
- Atomic operations guarantee all-or-nothing writes
- Production-ready error handling

---

#### P3: Comprehensive Error Propagation ✅ COMPLETE (2026-02-14)

**Problem**: Pipeline goroutines have weak error handling. Only tar errors are captured.

**Status**: ✅ **IMPLEMENTED** (2026-02-14)

**What Was Delivered**:
- ✅ Integrated `golang.org/x/sync/errgroup` for comprehensive error propagation
- ✅ Refactored `executePipeline` to use errgroup with context support
- ✅ All pipeline stages (tar, compress, encrypt) now report errors
- ✅ Fixed deadlock issues by properly understanding internal goroutine architecture
- ✅ Coverage improved: 83.2% → 84.2% for backup package
- ✅ All tests pass with no regressions

**Decision: Error Group Pattern**
- Use `golang.org/x/sync/errgroup`
- Captures first error from any goroutine
- Supports context-based cancellation
- Standard Go pattern
- Estimated effort: 1 day ✅ ACTUAL: 1 day

**Implementation Details**:
- Package: Added `golang.org/x/sync v0.19.0` dependency
- Refactored: `internal/backup/backup.go` - `executePipeline()` function
- Context: `context.Background()` with `errgroup.WithContext()`  
- Tar runs in errgroup goroutine, returns errors via `g.Wait()`
- Compress/Encrypt called sequentially (they spawn internal goroutines)
- Errors propagate via `pipe.CloseWithError()` → `io.Copy()`

**Key Insight**:
- Compress/Encrypt methods already spawn their own internal goroutines
- Wrapping them in additional goroutines caused circular pipe deadlocks
- Solution: Call sequentially, rely on internal goroutines + pipe error propagation

**Impact**:
- Trust score: 7.5/10 → 8.0/10
- All pipeline errors now captured (tar, compress, encrypt, output)
- Context support enables future cancellation features
- More robust error handling for production use


---

#### P4: Secure Passphrase Handling ⚠️ CRITICAL

**Problem**: Passphrase visible in process list and shell history.

**Current State**:
```bash
# Insecure: visible in `ps aux` and `.bash_history`
secure-backup restore --passphrase "my-secret-pass" ...
```

**Goal**: Unattended use with security.

**Decision: Modified Hybrid with Security Warning**

**Priority Order (Mutually Exclusive)**:
1. **`--passphrase` flag** (current) - Check first for backward compatibility
   - **Print security warning** if used
   - Example: "Warning: Passphrase on command line is insecure. Use --passphrase-file or SECURE_BACKUP_PASSPHRASE env var."
2. **`SECURE_BACKUP_PASSPHRASE` env var** - Check second
3. **`--passphrase-file` flag** - Check third
   - User manages file permissions (chmod 600)

**Mutually Exclusive**: Error if multiple methods provided.

**Estimated effort**: 1 day

**Implementation Notes**:
- Check in order: flag → env var → file
- Warn on insecure flag usage
- Error if multiple sources provided
- No interactive prompt (breaks unattended use)

---

### High Priority Issues

#### P5: Backup Locking

**Problem**: Concurrent backups to same destination can conflict.

**Current State**:
- No protection against simultaneous backups
- Race conditions in retention cleanup

**Decision: File-Based Locking (Fail Loudly, No Cleanup)**
- Use lock file: `<dest>/.backup.lock` with PID and timestamp
- **No cleanup - fail loudly if lock exists**
- **Print error with PID and helpful message**
- User must manually remove stale locks
- Estimated effort: 0.5 days

**Implementation Notes**:
- Lock file contains: PID, timestamp, hostname
- Before backup: check if lock exists
- If exists: read PID from lock file and print error with details
- **Error message**: `"Backup already in progress (PID 12345, lock file: .backup.lock). If stale, remove manually."`
- On success: remove lock file
- On error: remove lock file (we created it)

---

#### P6: Restore Safety Checks

**Problem**: Restore silently overwrites existing files.

**Current State**:
```go
outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, ...)
```

**Decision: Require --force Flag**
- Error if destination directory is not empty
- `--force` to override and allow overwrite
- Prevents accidental data loss
- Estimated effort: 0.5 days

**Implementation Notes**:
- Check if destination directory exists and is non-empty
- Return error with helpful message
- `--force` flag bypasses check

---

### Implementation Strategy

**Week 1: Critical Fixes (P1-P4)**
- Day 1-2: P1 - Backup Metadata & Verification (unified `--manifest` flag)
- Day 2-3: P2 - Atomic Writes (temp file + rename, no cleanup)
- Day 3-4: P3 - Error Propagation (errgroup)
- Day 4-5: P4 - Passphrase Handling (hybrid with warning)

**Week 2: High Priority (P5-P6)**
- Day 1: P5 - Backup Locking (fail loudly with PID, no cleanup)
- Day 2: P6 - Restore Safety (--force flag)

**Total Estimated Effort**: 1.5-2 weeks

---

## Future Phases

**Phase 3**: Enhanced Encryption (MERGED INTO PHASE 6.8)
- age encryption support (`filippo.io/age`)
- Modern alternative to GPG
- Simpler key management

**Phase 4**: Advanced Compression
- zstd compression (better ratio, faster)
- lz4 compression (fastest)
- Compression benchmarking

**Phase 7**: Docker Integration (Optional)
- Docker SDK client
- Volume discovery and listing
- `--volume` flag for volume backups
- Container pause/restart support

---

## Architecture

### Critical Pipeline Order

```
BACKUP:  Source → TAR → COMPRESS → ENCRYPT → File
RESTORE: File → DECRYPT → DECOMPRESS → EXTRACT → Destination
```

**Why**: Encrypted data is cryptographically random and incompressible. Wrong order = 0% compression!

### Project Structure

```
secure-backup/
├── cmd/                    # Cobra CLI commands
│   ├── backup.go          # backup command
│   ├── restore.go         # restore command
│   ├── verify.go          # verify command
│   └── list.go            # list command
├── internal/
│   ├── archive/           # TAR operations
│   ├── compress/          # Compression (gzip, future: zstd)
│   ├── encrypt/           # Encryption (GPG, future: age)
│   ├── errors/            # User-friendly error handling
│   ├── progress/          # Progress tracking
│   ├── backup/            # Pipeline orchestration
│   └── retention/         # Retention management
├── test_data/             # Test infrastructure (keys gitignored)
├── main.go                # Entry point
├── README.md              # User documentation
├── USAGE.md               # Detailed usage guide
└── agent_prompt.md        # This file
```

### Key Design Patterns

1. **Interface-Based Extensibility** - Easy to add age/zstd later
2. **Streaming Everything** - Constant memory usage via `io.Pipe()`
3. **Security First** - Path traversal protection, symlink validation
4. **Minimal Dependencies** - stdlib + cobra + testify

---

## Testing

### Philosophy

**Goal**: Pragmatic testing - unit tests for coverage, integration tests for confidence

**Current Coverage: 83.2%** (backup package)

**Package Breakdown:**
```
internal/errors/     100.0%  ✅ User error handling
internal/progress/    90.0%  ✅ Progress tracking
internal/backup/      83.2%  ✅ Pipeline orchestration
internal/compress/    76.9%  ✅ Real gzip operations
internal/retention/   75.3%  ✅ Policy management
internal/archive/     72.5%  ✅ Real tar operations
internal/encrypt/     67.9%  ✅ Real GPG encryption
```

### Test Strategy

**Two-Phase Approach:**

1. **Unit Tests** - Fast, focused, test individual functions and error paths
2. **Integration Tests** - Full backup → restore → verify cycles with real GPG

**Test with REAL operations:**
- ✅ GPG encryption/decryption (auto-generated test keys)
- ✅ Full pipelines (tar, gzip, GPG)
- ✅ Retention policy (file creation, deletion, time-based filtering)

**Test Key Management:**
- Script: `test_data/generate_test_keys.sh` (POSIX-compliant)
- Keys are **ALWAYS** generated, **NEVER** checked into git
- CI runs script before tests
- Keys gitignored for security best practices

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

# Skip integration tests (faster)
go test ./... -short
```

---

## Development Workflow

### Building
```bash
go build -o secure-backup .
```

### Testing
```bash
# All tests
go test ./...

# Specific package
go test ./internal/backup -v

# With coverage
go test ./... -coverprofile=coverage.out
```

### Manual Testing
```bash
# 1. Create test data
mkdir -p /tmp/test-{source,backups,restore}
echo "test data" > /tmp/test-source/test.txt

# 2. Backup
./secure-backup backup \
  --source /tmp/test-source \
  --dest /tmp/test-backups \
  --public-key ~/.gnupg/backup-pub.asc

# 3. List
./secure-backup list --dest /tmp/test-backups

# 4. Verify
./secure-backup verify \
  --file /tmp/test-backups/backup_*.tar.gz.gpg \
  --private-key ~/.gnupg/backup-priv.asc

# 5. Restore
./secure-backup restore \
  --file /tmp/test-backups/backup_*.tar.gz.gpg \
  --dest /tmp/test-restore \
  --private-key ~/.gnupg/backup-priv.asc

# 6. Verify content
diff -r /tmp/test-source /tmp/test-restore/test-source
```

---

## User Preferences (CRITICAL - MUST FOLLOW)

### 1. Git Workflow

**NEVER auto-commit without explicit user instruction**

✅ **CORRECT:**
```
1. Make changes
2. git add <files>
3. Ask user: "Ready to commit?"
4. WAIT for user to say "commit" or "commit and push"
5. ONLY THEN: git commit + git push
```

❌ **WRONG:**
```
1. Make changes
2. git add <files>
3. git commit (WITHOUT ASKING)  ← VIOLATION
4. git push (WITHOUT ASKING)     ← VIOLATION
```

**Pattern: MAKE → STAGE → ASK → WAIT → COMMIT (only when authorized)**

### 2. Documentation

- **ALWAYS update agent_prompt.md** after significant work
- Document current state, not just plans
- This file is the source of truth for next agent

### 3. Testing Standards

- Use `want` for expected values (not `expected`)
- Use `got` for actual values (not `result`)
- Follows standard Go testing conventions

### 4. Development Approach

- **Testing**: Unit tests first, then integration tests
- **Test keys**: ALWAYS generated via script, NEVER checked into git
- **Native Go**: Avoid `os.Exec` calls where possible
- **Interface-based**: Design for extensibility
- **General-purpose first**: Docker is secondary

---

## Important Context

### Why Compress Before Encrypt?

Encrypted data is cryptographically random and **cannot be compressed**.

- ✅ Correct: TAR → COMPRESS → ENCRYPT (60-80% smaller)
- ❌ Wrong: TAR → ENCRYPT → COMPRESS (0% compression!)

### Why GPG First, Then age?

**Phase 1 (Current)**: GPG
- User already has GPG keys
- Standard, widely supported
- Proven security

**Phase 3 (Future)**: age
- Simpler, modern alternative
- Better UX for key management
- Both will be supported via interface

### Why Deprioritize Docker?

Original plan: Docker-specific backup tool.

**Pivot**: Core functionality is valuable for **any** directory backup. Docker volumes are just another use case. Wider audience appeal as general-purpose tool.

Docker support remains planned (Phase 7), just deprioritized.

---

## File Naming Convention

**Backup files**: `backup_{sourcename}_{timestamp}.tar.gz.gpg`

Example: `backup_documents_20260207_165324.tar.gz.gpg`

- `.tar` = tar archive
- `.gz` = gzip compressed
- `.gpg` = GPG encrypted

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-02-07 | Rename to secure-backup | General-purpose focus, wider appeal |
| 2026-02-07 | Deprioritize Docker to Phase 7 | Core backup valuable standalone |
| 2026-02-07 | Apache 2.0 license | Permissive, business-friendly |
| 2026-02-08 | Compress-before-encrypt | Critical for compression efficiency |
| 2026-02-08 | Interface-based design | Easy to add age/zstd later |
| 2026-02-08 | Streaming architecture | Constant memory, any size backup |
| 2026-02-08 | Gzip level 6 default | Good balance speed/ratio |
| 2026-02-08 | Remove Phase 5.4 | Disk space checks unreliable, prompts break automation |
| 2026-02-08 | Remove Phase 6 | Remote storage backends not viable for now |
| 2026-02-08 | Productionization Effort | Production readiness analysis revealed 6 critical items (P1-P6) |
| 2026-02-08 | Simplicity over features | No config files, unattended operation with security |
| 2026-02-08 | GPG stays, may add others | Original requirement, age/others are additions not replacements |
| 2026-02-08 | Combine P1+P7 into P1 | Manifest and checksum naturally coupled, simpler UX |
| 2026-02-08 | Manifest enabled by default | Production hardening means verify by default, --skip-manifest to disable |
| 2026-02-08 | No automatic cleanup | User deals with stale .tmp and .lock files manually |
| 2026-02-08 | Fail loudly on conflicts | No silent recovery, print errors with PID and exit |
| 2026-02-08 | Passphrase priority order | Flag (with warning) → env var → file (mutually exclusive) |
| 2026-02-14 | P1 Implementation Complete | Manifest package with 93.2% coverage, all commands integrated, trust score 6.5→7.5 |
| 2026-02-14 | P2 Implementation Complete | Atomic writes using temp file + rename, fixed race condition, 3 new tests added |
| 2026-02-14 | P3 Implementation Complete | Errgroup pattern for comprehensive error propagation, context support, coverage 83.2%→84.2%, trust score 7.5→8.0 |
| 2026-02-14 | Compress/Encrypt Goroutine Insight | Methods already spawn internal goroutines; wrapping causes deadlocks, call sequentially instead |

---

## Next Agent Instructions

### On Starting Work:

1. **Read this file first** - Current source of truth
2. **Check task.md** - See what's in progress
3. **Review recent commits** - Understand latest changes
4. **Ask clarifying questions** - Don't assume user intent

### During Work:

1. **Update task.md** - Mark progress
2. **Create artifacts** - For complex features
3. **Stage changes** - `git add`, **NEVER auto-commit**
4. **Test thoroughly** - Maintain coverage
5. **Follow conventions** - Use `want`/`got` nomenclature

### Before Finishing:

1. **Update this file** - Reflect current state
2. **Update task.md** - Mark completed items
3. **Document decisions** - Add to decision log if architectural
4. **Stage all changes** - `git add` but **DO NOT commit**
5. **ASK user** - "Ready to commit?" and wait for approval
6. **Leave clear handoff** - Next agent should know exact state

### Critical Reminders:

- ⚠️ **NEVER auto-commit** - This is the #1 rule
- ⚠️ **ALWAYS update this file** after significant work
- ⚠️ **ALWAYS ask before committing** - Stage and ask, don't assume
- ⚠️ **USE want/got** nomenclature in all tests

---

## Quick Reference

```bash
# Build
go build -o secure-backup .

# Test
go test ./...

# Coverage
go test ./... -coverprofile=coverage.out

# Run
./secure-backup --help

# Install locally
go build -o secure-backup . && sudo mv secure-backup /usr/local/bin/

# Format
go fmt ./...

# Vet
go vet ./...

# Lint (if golangci-lint installed)
golangci-lint run
```

---

---

**Last Updated**: 2026-02-14  
**Last Updated By**: Agent (conversation 122bad7c-d394-4718-b5ff-8fe3e44e3b2b)  
**Project Phase**: Phase 5 Complete (User Experience), Productionization Effort In Progress  
**Production Trust Score**: 8.0/10 - P1+P2+P3 complete, strong data integrity, reliability, and error handling  
**Productionization**: P1 ✅ COMPLETE, P2 ✅ COMPLETE, P3 ✅ COMPLETE, P4-P6 remaining (3 items), ~3-4 days estimated  
**Next Milestone**: P4 - Secure Passphrase Handling
