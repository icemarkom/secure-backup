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
**Language**: Go 1.26+

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
- GPG and AGE encryption
- Gzip compression (default) or `--compression none` passthrough
- Streaming architecture (constant 10-50MB memory, 1 MB buffered pipes)
- Retention management (keep last N backups)
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

**Phase 5.1: Silent by Default + Progress** ✅ (2026-02-08, wired 2026-02-15)
- `internal/progress` package (ProgressReader/Writer)
- Unix philosophy: silent success, errors to stderr
- `--verbose` flag for progress and details
- Progress bars wired into: backup, restore, verify pipelines + checksum compute/validate
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

## Productionization Effort

> **Goal**: Harden the tool for production use with mission-critical data  
> **Philosophy**: Simplicity over features. No config files. Unattended operation with security.  
> **Status**: ✅ P1-P7, P10-P13, P16-P19 COMPLETE | ⛔ P8-P9, P14-P15 WON'T FIX | All items resolved  
> **Trust Score**: 7.5/10 — All productionization items resolved

### Critical Issues (All Complete) ✅

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

#### P4: Secure Passphrase Handling ✅ COMPLETE (2026-02-15)

**Problem**: Passphrase visible in process list and shell history.

**Status**: ✅ **IMPLEMENTED** (2026-02-15)

**What Was Delivered**:
- ✅ `internal/passphrase` package with 94.9% test coverage
- ✅ `Get()` function supporting three methods with priority order
- ✅ Security warning printed to stderr when `--passphrase` flag is used
- ✅ Mutual exclusivity validation - error if multiple sources provided
- ✅ Updated `cmd/restore.go` and `cmd/verify.go` with `--passphrase-file` flag
- ✅ Environment variable support: `SECURE_BACKUP_PASSPHRASE`
- ✅ Comprehensive tests (17 test cases covering all scenarios)
- ✅ Documentation updated (USAGE.md and README.md)

**Decision: Modified Hybrid with Security Warning**

**Priority Order (Mutually Exclusive)**:
1. **`--passphrase` flag** (backward compatibility)
   - **Prints security warning to stderr** if used
   - Warning: "WARNING: Passphrase on command line is insecure and visible in process lists. Use SECURE_BACKUP_PASSPHRASE environment variable or --passphrase-file instead."
2. **`SECURE_BACKUP_PASSPHRASE` env var** - Recommended for automation
3. **`--passphrase-file` flag** - Recommended for interactive use
   - User manages file permissions (chmod 600)
   - Warns if file is world-readable

**Implementation Details**:
- Package: `internal/passphrase` with `Get()` and `readFromFile()` functions
- Commands updated: `cmd/restore.go`, `cmd/verify.go`
- Both commands support all three passphrase methods
- File reading automatically trims whitespace/newlines
- Empty passphrase allowed (for keys without passphrase)
- Errors are user-friendly with hints

**Files Modified**:
- New: `internal/passphrase/passphrase.go`, `internal/passphrase/passphrase_test.go`
- Modified: `cmd/restore.go`, `cmd/verify.go`, `USAGE.md`, `README.md`

**Impact**:
- Trust score: 8.0/10 → 8.5/10
- Secure passphrase handling for unattended operation
- Backward compatible with existing workflows
- Production-ready security for cron jobs and automation

**Estimated effort**: 1 day ✅ ACTUAL: 1 day

---

### High Priority Issues

#### P5: Backup Locking ✅ COMPLETE (2026-02-15)

**Problem**: Concurrent backups to same destination can conflict.

**Status**: ✅ **IMPLEMENTED** (2026-02-15)

**What Was Delivered**:
- ✅ `internal/lock` package with 81.8% test coverage
- ✅ Per-destination file-based locking (`<dest>/.backup.lock`)
- ✅ Atomic write pattern (temp file + rename)
- ✅ Helpful error messages with PID, hostname, timestamp
- ✅ Manual cleanup only (fail loudly philosophy)
- ✅ 12 comprehensive unit tests, all passing

**Decision: Per-Destination File-Based Locking**
- Lock file: `<dest>/.backup.lock` with PID, timestamp, hostname
- **Fail loudly** - no automatic cleanup of stale locks
- User must manually remove stale locks
- Chosen for **simplicity** over maximum concurrency
- Estimated effort: 0.5 days ✅ ACTUAL: 0.5 days

**Implementation Details**:
- Package: `internal/lock` with `LockInfo` type, `Acquire()`, `Release()`, `Read()` functions
- Commands updated: `cmd/backup.go` - lock acquired before backup, released via defer
- Error message: "Backup already in progress (PID 12345 on hostname, started 2026-01-01 00:00:00)"
- Lock cleanup: `defer lock.Release(lockPath)` handles both success and error cases
- Test coverage: 81.8% for lock package

**Files Modified**:
- New: `internal/lock/lock.go`, `internal/lock/lock_test.go`
- Modified: `cmd/backup.go`

**Manual Verification**:
- ✅ Stale lock detection works with helpful error message
- ✅ Lock properly cleaned up after successful backup
- ✅ No `.backup.lock` remains after completion

**Impact**:
- Trust score: 8.5/10 → 8.8/10
- Prevents concurrent backup conflicts
- Safe retention cleanup operations
- Production-ready locking mechanism

---

#### P6: Restore Safety Checks ✅ COMPLETE (2026-02-15)

**Problem**: Restore silently overwrites existing files.

**Status**: ✅ **IMPLEMENTED** (2026-02-15)

**What Was Delivered**:
- ✅ `isDirectoryNonEmpty()` helper function with full edge case handling
- ✅ `--force` flag to allow restore to non-empty directories
- ✅ Safety check before restore execution (fails if non-empty without --force)
- ✅ User-friendly error with hint to use --force
- ✅ Warning message in verbose mode when using --force
- ✅ 5 comprehensive tests covering all scenarios (100% coverage for new code)
- ✅ All tests pass, no regressions

**Decision: Require --force Flag**
- Error if destination directory is not empty
- `--force` to override and allow overwrite
- Prevents accidental data loss
- Estimated effort: 0.5 days ✅ ACTUAL: 0.5 days

**Implementation Details**:
- Helper: `isDirectoryNonEmpty()` - Checks if directory exists and contains files
  - Returns `false` for non-existent directories (safe to proceed)
  - Returns `false` for empty directories (safe to proceed)
  - Returns `true` for directories with files or subdirectories (requires --force)
  - Returns error if path exists but is not a directory
- Safety check runs before `os.MkdirAll()` and pipeline execution
- Added `Force` field to `RestoreConfig` struct
- Added `--force` flag to `cmd/restore.go`
- Error message: "Destination directory is not empty: /path\nHint: Use --force to overwrite..."
- Verbose warning: "WARNING: Restoring to non-empty directory - existing files may be overwritten"

**Files Modified**:
- Modified: `internal/backup/restore.go` - Added `isDirectoryNonEmpty()`, updated `PerformRestore()`
- Modified: `internal/backup/restore_test.go` - Added 5 new tests (helper + 4 scenarios)
- Modified: `cmd/restore.go` - Added `--force` flag

**Tests Added**:
- `TestIsDirectoryNonEmpty` - Unit test for helper (5 sub-tests)
- `TestPerformRestore_NonEmptyDestination_WithoutForce` - Verifies error without --force
- `TestPerformRestore_NonEmptyDestination_WithForce` - Verifies success with --force
- `TestPerformRestore_EmptyDestination_WithoutForce` - Verifies no change for empty dir
- `TestPerformRestore_NonexistentDestination_WithoutForce` - Verifies no change for non-existent

**Impact**:
- Trust score: 8.8/10 → 9.0/10 ⭐ **PRODUCTION READY**
- Prevents accidental data loss during restore
- User-friendly error messages guide users to safe operations
- Production-ready safety mechanisms complete


---

### Production Hardening Round 2 (P7-P19)

> **Source**: Code review (2026-02-15, conversation fd1d5446)  
> **Status**: ✅ All items resolved — GitHub issues #1-#13 closed

#### Critical Priority

| ID | GitHub | Issue | File(s) | Status |
|----|--------|-------|---------|--------|
| **P7** | [#1](https://github.com/icemarkom/secure-backup/issues/1) | ~~TOCTOU race in lock: `Stat()` + `WriteFile()` is not atomic~~ | `internal/lock/lock.go` | ✅ COMPLETE |
| **P8** | [#2](https://github.com/icemarkom/secure-backup/issues/2) | ~~Symlink traversal on extract: link targets not validated~~ | `internal/archive/tar.go` | ⛔ WON'T FIX |
| **P9** | [#3](https://github.com/icemarkom/secure-backup/issues/3) | ~~No decompression bomb protection: unbounded `io.Copy`~~ | `internal/archive/tar.go` | ⛔ WON'T FIX |

#### P7: TOCTOU Race in Lock ✅ COMPLETE (2026-02-15)

**Problem**: `Acquire()` used `os.Stat()` + temp file + `os.Rename()` — a TOCTOU race allowed concurrent processes to both pass the check and overwrite each other's lock.

**Fix**: Replaced with `os.OpenFile(lockPath, O_WRONLY|O_CREATE|O_EXCL, 0644)` — kernel-guaranteed atomic create-or-fail. No race window.

**What Was Delivered**:
- ✅ Atomic lock creation using `O_CREATE|O_EXCL`
- ✅ Eliminated temp file + rename pattern entirely
- ✅ Extracted `lockExistsError()` helper for cleaner code
- ✅ New `TestAcquire_ConcurrentRace` test (10 goroutines, exactly 1 wins)
- ✅ Updated `TestAcquire_AtomicCreation` (validates JSON content immediately)
- ✅ All 15 lock tests pass, full test suite passes with no regressions

**Files Modified**:
- Modified: `internal/lock/lock.go` — Atomic `O_CREATE|O_EXCL` lock creation
- Modified: `internal/lock/lock_test.go` — Updated atomic test + new concurrent race test


#### High Priority

| ID | GitHub | Issue | File(s) | Effort |
|----|--------|-------|---------|--------|
| **P10** | [#4](https://github.com/icemarkom/secure-backup/issues/4) | ~~Backup file permissions 0666 (umask-dependent) instead of 0600~~ | `internal/backup/backup.go`, `internal/manifest/manifest.go`, `cmd/backup.go` | ✅ COMPLETE |
| **P11** | [#5](https://github.com/icemarkom/secure-backup/issues/5) | ~~Retention deletes backups but orphans manifest files~~ | `internal/retention/policy.go` | ✅ COMPLETE |
| **P12** | [#6](https://github.com/icemarkom/secure-backup/issues/6) | ~~Context not propagated; no SIGTERM/signal handling~~ | `main.go`, `cmd/root.go`, `cmd/*.go`, `internal/backup/*.go` | ✅ COMPLETE |
| **P13** | [#7](https://github.com/icemarkom/secure-backup/issues/7) | ~~Manifest path derived via brittle `TrimSuffix` on extension~~ | `internal/manifest/manifest.go` | ✅ COMPLETE |
| **P14** | [#8](https://github.com/icemarkom/secure-backup/issues/8) | ~~Passphrase stored as `string`, never zeroed after use~~ | `internal/passphrase/passphrase.go`, `internal/encrypt/gpg.go` | ⛔ WON'T FIX |
| **P15** | [#9](https://github.com/icemarkom/secure-backup/issues/9) | ~~`err` variable shadowing in backup defer/cleanup logic~~ | `internal/backup/backup.go` | ⛔ WON'T FIX |
| **P16** | [#10](https://github.com/icemarkom/secure-backup/issues/10) | ~~Zero `cmd/` test coverage~~ → CLI error output testing | `cmd/*.go`, `test-scripts/e2e_test.sh` | ✅ COMPLETE |

#### Medium Priority

| ID | GitHub | Issue | File(s) | Effort |
|----|--------|-------|---------|--------|
| **P17** | [#11](https://github.com/icemarkom/secure-backup/issues/11) | ~~`filepath.Walk` follows symlinks during backup creation~~ | `internal/archive/tar.go` | ✅ COMPLETE |
| **P18** | [#12](https://github.com/icemarkom/secure-backup/issues/12) | ~~Armor decode fallback corrupts stream (partial read)~~ | `internal/encrypt/gpg.go` | ✅ COMPLETE |
| **P19** | [#13](https://github.com/icemarkom/secure-backup/issues/13) | ~~`formatSize()` duplicated in 3 files~~ | `internal/format/` | ✅ COMPLETE |

---

### Implementation Strategy

**Week 1: Critical Fixes (P1-P4)** ✅ COMPLETE
- Day 1-2: P1 - Backup Metadata & Verification (unified `--manifest` flag)
- Day 2-3: P2 - Atomic Writes (temp file + rename, no cleanup)
- Day 3-4: P3 - Error Propagation (errgroup)
- Day 4-5: P4 - Passphrase Handling (hybrid with warning)

**Week 2: High Priority (P5-P6)** ✅ COMPLETE
- Day 1: P5 - Backup Locking (fail loudly with PID, no cleanup)
- Day 2: P6 - Restore Safety (--force flag)

**Week 3: Security Hardening (P7-P10)**
- Day 1: P7 - Fix lock TOCTOU + P10 - File permissions (quick wins) ✅ COMPLETE
- ~~Day 2: P8 - Symlink validation on extract~~ ⛔ WON'T FIX + P17 - Symlink handling on create ✅ COMPLETE
- ~~Day 3: P9 - Decompression bomb protection~~ ⛔ WON'T FIX

**Week 4: Reliability & Quality (P11-P16)**
- Day 1: P11 - Retention manifest cleanup + P13 - Manifest path fix
- Day 2: P12 - Context/signal propagation
- ~~Day 3: P14 - Passphrase zeroing~~ ⛔ WON'T FIX + ~~P15 - Error shadowing fix~~ ⛔ WON'T FIX
- ~~Day 4-5: P16 - cmd/ integration tests~~ ⛔ WON'T FIX

**Week 5: Polish (P17-P19)**
- P18 - Armor decode fix ✅ COMPLETE + P19 - Deduplicate formatSize ✅ COMPLETE

**Total Estimated Effort**: P1-P6 done (1.5 weeks) + P7-P19 (~2.5 weeks)

## Future Phases

**Phase 3**: Enhanced Encryption ✅ COMPLETE — [#14](https://github.com/icemarkom/secure-backup/issues/14)
- ✅ AGE encryption via `filippo.io/age` v1.3.1
- ✅ `encrypt.Method` iota type (`GPG`, `AGE`) with `ParseMethod()`, `String()`, `Extension()`
- ✅ `MethodGPG`/`MethodAGE` string constants, `ValidMethods()`/`ValidMethodNames()` helpers
- ✅ Unified flags: `--public-key` = GPG file path or AGE string (keyed off `--encryption`)
- ✅ `--private-key` = file path for both GPG (.asc) and AGE identity
- ✅ `ResolveMethod()` centralizes detection from flags/file extensions (`.gpg` / `.age`)
- ✅ 11 AGE-specific tests, all pass

**Phase 4**: Advanced Compression — [#15](https://github.com/icemarkom/secure-backup/issues/15)
- ✅ `compress.Method` iota type (`Gzip`) with `ParseMethod()`, `String()`, `ValidMethods()`/`ValidMethodNames()`
- ✅ `MethodGzip` string constant, `Type() Method` on `Compressor` interface
- ✅ CLI help text consolidated: `Long` descriptions moved to `init()` using `fmt.Sprintf` with constants
- ✅ Help text uses `strings.ToUpper()` for display labels, lowercase for CLI parameter values
- ✅ Comprehensive unit tests: `encrypt_test.go` (8 tests), `compress_test.go` (6 tests)
- ✅ AGE integration tests: backup→restore and backup→verify cycles
- ✅ AGE e2e pipeline: keygen → backup → verify → restore → diff in `e2e_test.sh`
- ✅ `--compression none` passthrough via `NoneCompressor` ([#40](https://github.com/icemarkom/secure-backup/issues/40))
- ✅ `--compression` flag wired in `cmd/backup.go` with `ParseMethod()`; default `gzip`
- ✅ `compress.ResolveMethod()` auto-detects compression from filename (`.tar.gz.*` → Gzip, `.tar.*` → None)
- ✅ Restore/verify auto-detect compression — no `--compression` flag needed
- ✅ Dynamic retention patterns via `compressor.Extension()` + `encMethod.Extension()`
- ✅ `manifest.New()` accepts dynamic compression/encryption names
- ✅ `IsBackupFile()` recognizes `.tar.gpg` and `.tar.age` extensions
- ✅ Dry-run output dynamic: skips DECOMPRESS step when compression=none
- ✅ E2E test: full `--compression none` pipeline (backup → verify → restore → diff)
- Future: zstd compression, lz4 compression, benchmarking

**Phase 7**: Docker Integration (Optional) — [#16](https://github.com/icemarkom/secure-backup/issues/16)
- Docker SDK client
- Volume discovery and listing
- `--volume` flag for volume backups
- Container pause/restart support

**Performance**: Backup Pipeline Performance ✅ COMPLETE — [#24](https://github.com/icemarkom/secure-backup/issues/24)
- ✅ Removed ASCII armor — binary GPG output (breaking change, auto-detects armored on decrypt)
- ✅ Switched to `klauspost/pgzip` — parallel gzip compression
- ✅ Added 1 MB `bufio` buffer on tar→compress pipe

**Testing**: End-to-End Pipeline Test ✅ COMPLETE — [#17](https://github.com/icemarkom/secure-backup/issues/17)
- Full `backup → list → verify → restore → diff` cycle in CI
- POSIX shell script (`e2e/e2e_test.sh`) testing the compiled binary
- Dry-run regression tests for all subcommands (backup, verify quick, verify full, restore)
- Separate CI job gated on unit test success

**Bug Fix**: Dry-Run Lock Side Effect ✅ FIXED — [#20](https://github.com/icemarkom/secure-backup/issues/20)
- `backup --dry-run` was acquiring a real `.backup.lock` file, blocking concurrent backups
- Fix: Guard `lock.Acquire()` with `!backupDryRun` in `cmd/backup.go`
- E2E regression tests added for dry-run in all subcommands

**Bug Fix**: Verify Partial Output Before Error ✅ FIXED — [#10](https://github.com/icemarkom/secure-backup/issues/10)
- `verify --file=...` (without `--private-key`) displayed manifest/checksum success before the error
- Fix: Validate all required flags before any output in `cmd/verify.go`
- Full SHA256 checksums now displayed (no truncation) in verify and list output
- Cobra `completion` subcommand disabled
- E2E regression test updated (step 9c asserts no partial output)

**Bug Fix**: Silent-by-Default Violations ✅ FIXED — [#29](https://github.com/icemarkom/secure-backup/issues/29)
- `backup` retention message printed with inverted `!backupVerbose` condition (output only when NOT verbose)
- `verify` manifest/checksum display always printed regardless of `--verbose`
- Fix: Removed inverted condition in `cmd/backup.go`, gated manifest display in `cmd/verify.go`

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
│   ├── backup/            # Pipeline orchestration
│   ├── common/            # Shared utilities (formatting, IO buffers, user errors)
│   ├── compress/          # Compression (gzip, future: zstd)
│   ├── encrypt/           # Encryption (GPG, future: age)
│   ├── lock/              # Backup locking (per-destination)
│   ├── manifest/          # Backup metadata & integrity verification
│   ├── passphrase/        # Secure passphrase handling (flag/env/file)
│   ├── progress/          # Progress tracking
│   └── retention/         # Retention management
├── examples/              # Usage examples (cron.daily script)
├── test-scripts/          # Test scripts (key generation, E2E)
├── test_data/             # Generated test data (keys, gitignored)
├── main.go                # Entry point
├── README.md              # User documentation
├── USAGE.md               # Detailed usage guide
└── agent_prompt.md        # This file
```

### Key Design Patterns

1. **Interface-Based Extensibility** - Easy to add age/zstd later
2. **Streaming Everything** - Constant memory usage via `io.Pipe()`
3. **Security First** - Path traversal protection, symlink validation
4. **Minimal Dependencies** - stdlib + cobra + testify + pgzip

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
- Script: `test-scripts/generate_test_keys.sh` (POSIX-compliant)
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

### 1. Git Workflow — Branch + PR (MANDATORY)

**All changes MUST go through feature branches and Pull Requests. Never push directly to `main`.**

**NEVER auto-commit without explicit user instruction**

**NEVER merge PRs without explicit user instruction — the user decides when to merge**

✅ **CORRECT:**
```
1. Create feature branch: git checkout -b <branch-name>
2. Make changes
3. git add <files>
4. Ask user: "Ready to commit?"
5. WAIT for user to say "commit" or "commit and push"
6. ONLY THEN: git commit + git push -u origin <branch-name>
7. Create PR: gh pr create --title "..." --body "..."
8. WAIT for user to say "merge"
9. ONLY THEN: gh pr merge
10. After merge, issue #N auto-closes
```

❌ **WRONG:**
```
1. Make changes on main
2. git commit + git push directly to main  ← VIOLATION
```

**Pattern: BRANCH → MAKE → STAGE → ASK → WAIT → COMMIT → PUSH → PR → ASK → WAIT → MERGE**

**Branch naming**: Use descriptive names like `release/v1.0.0`, `fix/issue-description`, `feat/feature-name`.

**GitHub Issues: Use auto-close keywords in PR descriptions**

> **✅ Auto-close keywords** (`Fixes #N`, `Closes #N`) in PR descriptions/commit messages
> now work correctly since we use merged PRs. Include them in commit messages or PR body
> to automatically close issues when the PR is merged.

✅ **CORRECT:**
```
1. Create branch and make changes
2. Ask user: "Ready to commit?"
3. WAIT for user approval
4. git commit -m "fix: description (Fixes #N)"
5. git push -u origin <branch-name>
6. gh pr create --title "fix: description" --body "Fixes #N"
7. After PR is merged, issue #N auto-closes
```

❌ **WRONG:**
```
1. Make changes
2. gh issue close #N  ← VIOLATION: PR hasn't been merged yet!
3. git commit + push later
```

❌ **ALSO WRONG:**
```
1. Make changes, commit, push, create PR
2. gh pr merge  ← VIOLATION: User hasn't approved the merge!
```

**Pattern: Branch → Push → PR → ASK → WAIT → Merge → Issue auto-closes**
### 2. Versioning — Semantic Versioning (MANDATORY)

**Format: `vMAJOR.MINOR.PATCH`** (e.g., `v1.0.0`)

- **When user requests a new release**: Bump the **PATCH** version (the Z in vX.Y.Z)
- **MAJOR/MINOR bumps**: Only on explicit user instruction
- Follow [Semantic Versioning](https://semver.org/) conventions

### 3. Documentation

- **ALWAYS update agent_prompt.md** after significant work
- Document current state, not just plans
- This file is the source of truth for next agent

### 4. Testing Standards

- Use `want` for expected values (not `expected`)
- Use `got` for actual values (not `result`)
- Follows standard Go testing conventions

### 5. Development Approach

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

### Why GPG? (Design Choice)

**GPG is an intentional design choice**, not a deficiency. The use of `golang.org/x/crypto/openpgp` is deliberate:
- User already has GPG keys
- Standard, widely supported
- Proven security
- Meets all functional requirements for this tool

This library was current when the project started and remains functional. **Do not flag it as a risk or failure mode.** If a maintained alternative is desired, the planned path is adding `age` encryption (Phase 3) as a *new option*, not replacing the GPG library.

**Phase 3 (Future)**: age
- Simpler, modern alternative via `filippo.io/age`
- Better UX for key management
- Both will be supported via the existing `Encryptor` interface

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
| 2026-02-15 | P4 Implementation Complete | Secure passphrase handling with env var/file support, 94.9% coverage, trust score 8.0→8.5 |
| 2026-02-15 | P5 Implementation Complete | Per-destination backup locking, 81.8% lock coverage, fail loudly with manual cleanup, trust score 8.5→8.8 |
| 2026-02-15 | Per-destination locking chosen | Prioritized simplicity over maximum concurrency, aligns with project philosophy |
| 2026-02-15 | P7 Implementation Complete | Fixed TOCTOU race with O_CREATE|O_EXCL atomic lock creation, 15 tests pass, concurrent race test added |
| 2026-02-15 | P13 Implementation Complete | Centralized `ManifestPath()` in `internal/manifest`, replaced 4 brittle `TrimSuffix` calls, `_manifest.json` suffix (no double extension) |
| 2026-02-15 | P11 Implementation Complete | Retention `ApplyPolicy()` now deletes manifest files alongside backups, dry-run reports manifests, 2 new tests |
| 2026-02-15 | P19 Implementation Complete | Consolidated `formatSize()` (3 copies) and `formatAge()` (2 copies) into new `internal/format` package with `Size()` and `Age()` functions |
| 2026-02-15 | P15 Closed as Won't Fix | `err` shadowing is inherent to Go; no scoping trick prevents all future mistakes. Current code is correct. Code review and linting are the right mitigation. |
| 2026-02-15 | P14 Closed as Won't Fix | Go strings are immutable and cannot be zeroed. Passphrase also flows through openpgp which makes internal copies. Zeroing the `[]byte` copy gives false security. Practical attack vectors (process lists, shell history, file permissions) were addressed in P4. |
| 2026-02-15 | P16 Reopened with narrowed scope | Original full cmd/ integration testing remains won't-fix for ROI. Reopened as "CLI Error Output Testing" after discovering duplicate error display and unwanted help output on runtime errors. Added `SilenceErrors` on root command, per-RunE `SilenceUsage` pattern, and 3 e2e tests. |
| 2026-02-15 | P8 Closed as Won't Fix | Threat model doesn't apply: backups are GPG-encrypted so attackers can't inject malicious tar entries. Fix risks breaking legitimate restores containing symlinks with absolute or out-of-tree targets. Tool is not a general-purpose tar extractor. |
| 2026-02-15 | P9 Closed as Won't Fix | Threat model doesn't apply: backups are GPG-encrypted, so attackers cannot inject crafted tar entries to create decompression bombs. The tool only extracts archives it created from real files. Adding size limits risks breaking legitimate restores of large backups. |
| 2026-02-15 | P10 Implementation Complete | Added `--file-mode` flag with `default` (0600), `system` (umask), or explicit octal modes. Secure by default, user-overridable. World-readable warning on stderr. Applied to both backup and manifest files. 3 new tests, all pass. |
| 2026-02-15 | Go version bumped to 1.26.0 | Updated `go.mod`, `test.yml`, `release.yaml` from Go 1.25 to 1.26. |
| 2026-02-15 | P17 Implementation Complete | Switched `filepath.Walk` → `filepath.WalkDir` in `CreateTar` to preserve symlinks as `tar.TypeSymlink` entries instead of dereferencing them. Used `os.Lstat` for source. 3 symlink tests (internal, external, round-trip), all pass. |
| 2026-02-15 | P18 Implementation Complete | Removed armor decode fallback in `Decrypt()`. Now always requires armored input (which is all the tool produces). Non-armored input returns explicit error instead of silently corrupting stream. 1 new test added. |
| 2026-02-15 | E2E Pipeline Test (#17) Complete | POSIX shell script in `e2e/e2e_test.sh` exercising full `backup → list → verify → restore → diff` through compiled binary. Chose shell over Go tests for production-realistic testing. Separate CI job, `make e2e` target. |
| 2026-02-15 | Dry-Run Lock Bug (#20) Fixed | `backup --dry-run` was creating `.backup.lock` on disk, blocking concurrent operations. Guarded `lock.Acquire()` with `!backupDryRun`. Added dry-run e2e regression tests for all subcommands. Reading files (GPG keys, manifests) is intentional; only writes are suppressed. |
| 2026-02-15 | GoReleaser config updated for v2 | Fixed deprecated `snapshot.name_template` → `version_template` and `format_overrides.format` → `formats`. Config passes `goreleaser check` cleanly. |
| 2026-02-15 | v1.0.0 Release | All productionization complete. Validated GoReleaser snapshot build: 5 platform archives + 2 .deb packages (amd64/arm64). README cleaned up for 1.0.0. Apt repo signing via Release/InRelease is standard Debian practice; individual .deb signing not needed. |
| 2026-02-15 | CLI error output fix (#10) | Fixed duplicate error display and unwanted help output on runtime errors. `SilenceErrors: true` on root command, per-RunE `cmd.SilenceUsage = true` pattern preserves usage for missing required flags. 3 e2e tests added. |
| 2026-02-15 | Branch+PR workflow | Switched from direct-to-main pushes to mandatory branch+PR workflow. All changes must go through feature branches and Pull Requests. Auto-close keywords (`Fixes #N`) now work via merged PRs. |
| 2026-02-15 | Performance issue filed (#24) | Identified 3 bottlenecks in backup pipeline: ASCII armor (+33% output bloat), single-threaded stdlib gzip, zero-buffered io.Pipe(). Combined fix expected 3-5× speedup for multi-GB backups. |
| 2026-02-15 | Performance fixes implemented (#24) | Removed ASCII armor (binary GPG only, breaking change), switched to pgzip (parallel gzip), added 1 MB buffered I/O on tar→compress pipe. All tests pass. |
| 2026-02-15 | Verify partial output fix (#10) | Moved `--private-key` validation before manifest output in `verify.go`. No partial success output before errors. |
| 2026-02-15 | Full SHA256 display | Removed `[:16]` truncation from verify and list checksum output. Full hash is useful for scripting and manual comparison. |
| 2026-02-15 | Cobra completion disabled | `CompletionOptions.DisableDefaultCmd = true` on root command. Can be re-enabled later if needed. |
| 2026-02-15 | Retention changed from days to count ([#27](https://github.com/icemarkom/secure-backup/issues/27)) | `--retention N` now means "keep last N backups" instead of "delete backups older than N days." Count-based retention works correctly regardless of backup frequency (hourly, daily, weekly). `DefaultKeepLast = 0` constant in `internal/retention`. |
| 2026-02-15 | Silent-by-default fix ([#29](https://github.com/icemarkom/secure-backup/issues/29)) | Fixed 2 violations: (1) `cmd/backup.go` retention message used inverted `!backupVerbose` condition, (2) `cmd/verify.go` manifest/checksum display was ungated. All output now requires `--verbose` except `list`, `version`, dry-run, and stderr warnings. |
| 2026-02-15 | Progress bars wired ([#31](https://github.com/icemarkom/secure-backup/issues/31)) | Connected existing `internal/progress` package to 5 operations: backup pipeline (tar reader), restore pipeline (file reader), full verify (file reader), checksum compute, checksum validate. All gated on `--verbose`. Added `ComputeChecksumProgress` / `ValidateChecksumProgress` to manifest package. |
| 2026-02-16 | License headers added ([#34](https://github.com/icemarkom/secure-backup/issues/34)) | Apache 2.0 boilerplate + `SPDX-License-Identifier: Apache-2.0` added to all 36 Go files and 2 shell scripts. CI enforcement via `google/addlicense -check` in `test.yml` unit job. `make license-check` target for local use. Makefile/YAML excluded from headers. |
| 2026-02-16 | Checked-in test keys removed ([#36](https://github.com/icemarkom/secure-backup/issues/36)) | 6 vestigial files under `test_data/test_data/` (including GPG private key and keyring) were tracked despite `.gitignore`. Removed via `git rm --cached`. Test infra already generates keys on the fly. |
| 2026-02-16 | Phase 3: age encryption ([#14](https://github.com/icemarkom/secure-backup/issues/14)) | Added AGE encryption via `filippo.io/age` v1.3.1. `--public-key` meaning is encryption-dependent: file path for GPG, direct string for AGE — keyed off `--encryption`, not prefix detection. `--public-key` remains `MarkFlagRequired` (Cobra validation). `--private-key` is always a file path. Introduced `encrypt.Method` iota type (`GPG`, `AGE`) with `ParseMethod()`, `String()`, `Extension()` to replace string matching. String constants `MethodGPG`/`MethodAGE` eliminate hardcoded strings. `ValidMethods()`/`ValidMethodNames()` provide dynamic method enumeration for CLI help text. `encrypt.ResolveMethod()` centralizes method detection from flags/file extensions. All switch blocks use explicit `case encrypt.GPG:`/`case encrypt.AGE:` with `default:` reserved for error handling. Restore/verify auto-detect from `.age`/`.gpg` extension. Passphrase logic skipped for AGE. 11 new tests. |
| 2026-02-16 | Compression Method scaffolding | Applied same iota-based `Method` type pattern to `internal/compress`: `Gzip` constant only — ZSTD/LZ4 not added (scaffolding only, no phantom features). `MethodGzip` string constant, `ParseMethod()`, `ValidMethods()`/`ValidMethodNames()`. Added `Type() Method` to `Compressor` interface. Replaced all hardcoded `"gzip"` strings across cmd/ and test files with `compress.Gzip`. Dry-run output now uses `cfg.Compressor.Type()` dynamically. |
| 2026-02-16 | CLI help text consolidated | Moved `Long` descriptions in `backup.go`, `restore.go`, `verify.go` from struct literals to `init()` functions. Uses `fmt.Sprintf` with `compress.ValidMethodNames()`, `encrypt.ValidMethodNames()`, `strings.ToUpper(encrypt.MethodGPG)`, `strings.ToUpper(encrypt.MethodAGE)` — zero hardcoded method names in help text. Lowercase constants for CLI parameter values, uppercase for display labels. |
| 2026-02-16 | Test coverage expanded | Added `encrypt_test.go` (8 tests for Method helpers), `compress_test.go` (6 tests for Method helpers), AGE integration tests (backup→restore, backup→verify cycles), AGE e2e pipeline in `e2e_test.sh`. |
| 2026-02-16 | None compression ([#40](https://github.com/icemarkom/secure-backup/issues/40)) | Added `--compression none` passthrough via `NoneCompressor`. Identity `Compress()`/`Decompress()`, empty `Extension()` so filenames become `.tar.gpg`/`.tar.age`. Added `.tar.gpg`/`.tar.age` to `knownBackupExtensions`. Named `NoneCompressor` (not Noop) for consistency with CLI-facing `none` method name. |
| 2026-02-16 | None compression CLI wiring ([#40](https://github.com/icemarkom/secure-backup/issues/40)) | Wired `--compression` flag in `cmd/backup.go` (default: `gzip`). Added `compress.ResolveMethod(filename)` for auto-detection in restore/verify (mirrors `encrypt.ResolveMethod()`). Retention pattern now dynamic via `compressor.Extension()` + `encMethod.Extension()`. `manifest.New()` changed from 3-arg to 5-arg (compression, encryption now caller-supplied). `IsBackupFile()` updated for `.tar.gpg`/`.tar.age`. Dry-run output skips DECOMPRESS step when none. E2E test added for full none-compression pipeline. |
| 2026-02-16 | Retention pattern scope fix ([#43](https://github.com/icemarkom/secure-backup/issues/43)) | Retention `ApplyPolicy()` used a narrow extension-specific glob (e.g. `backup_*.tar.gz.gpg`), so switching compression/encryption orphaned old backups. Fix: broad `backup_*` glob + `IsBackupFile()` post-filtering. Removed `Pattern` from `Policy` struct. `ListBackups()` simplified to single-param (no pattern). 2 new tests: `TestApplyPolicy_MixedExtensions`, `TestListBackups_MixedExtensions`. |
| 2026-02-16 | Filed embedded manifest issue ([#44](https://github.com/icemarkom/secure-backup/issues/44)) | Embed manifest JSON as first tar entry for durability. Sidecar remains for fast access. Future enhancement. |
| 2026-02-16 | Filed manifest-first management issue ([#45](https://github.com/icemarkom/secure-backup/issues/45)) | Manifested backups as first-class citizens, non-manifested as orphans. Affects list, verify, retention. Future enhancement. |
| 2026-02-16 | Standardized IO buffer size ([#47](https://github.com/icemarkom/secure-backup/issues/47)) | Created shared `IOBufferSize = 1 MiB` const. Replaced all pipeline `io.Copy` calls with `io.CopyBuffer(... common.NewBuffer())` across 8 files (11 call sites). Benchmarked 32KB–4MB; 1MiB chosen for syscall reduction on disk IO. |
| 2026-02-16 | Consolidated helper packages ([#48](https://github.com/icemarkom/secure-backup/issues/48)) | Merged `internal/format`, `internal/ioutil`, `internal/errors` into `internal/common`. All shared utility functions (formatting, IO buffers, user-friendly errors) live in one package. `internal/progress` kept separate (external dep). **Ongoing: all new shared helpers go in `internal/common`.** |
| 2026-02-17 | Cron.daily example script ([#50](https://github.com/icemarkom/secure-backup/pull/50)) | Added `examples/cron.daily/secure-backup` — drop-in script for `/etc/cron.daily/` on Ubuntu. Supports multiple source directories via bash array, configurable AGE/GPG encryption, retention, logging, and per-source failure tracking. `.gitignore` scoped `secure-backup` → `/secure-backup` to avoid ignoring the example. |
| 2026-02-17 | Manifest size fields ([#51](https://github.com/icemarkom/secure-backup/issues/51)) | Renamed `size_bytes` → `compressed_size_bytes`, added `uncompressed_size_bytes`. Uncompressed size counted inside `CreateTar` as raw file data bytes (no tar headers, no TOCTOU). `CreateTar` returns `(int64, error)`, plumbed through `executePipeline` → `PerformBackup` → manifest. `getDirectorySize()` remains for progress bar estimate only. |

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

- ⚠️ **NEVER auto-commit** - Stage and ASK, don't assume
- ⚠️ **NEVER auto-merge PRs** - Create the PR and ASK, the user decides when to merge
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

# Check license headers
make license-check
```

---

**Last Updated**: 2026-02-17  
**Last Updated By**: Agent (conversation 597017f6-d83e-4dca-a2e8-c34f4d358dcf)  
**Current Release**: v1.1.3  
**Project Phase**: Post-1.0 improvements ✅  
**Production Trust Score**: 7.5/10 — All productionization items resolved  
**Productionization**: P1-P7, P10-P13, P16-P19 ✅ | P8-P9, P14-P15 ⛔ | **ALL ITEMS RESOLVED** 🎉  
**Next Milestone**: [#44](https://github.com/icemarkom/secure-backup/issues/44) embedded manifest, [#45](https://github.com/icemarkom/secure-backup/issues/45) manifest-first management, [#15](https://github.com/icemarkom/secure-backup/issues/15) zstd, [#16](https://github.com/icemarkom/secure-backup/issues/16) Docker
