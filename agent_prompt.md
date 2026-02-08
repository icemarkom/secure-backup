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

## Phase 6: Productionization (PLANNED)

> **Goal**: Harden the tool for production use with mission-critical data
> 
> **Current Trust Score**: 6.5/10 - Good for personal use, needs hardening for business-critical data
> 
> **Target**: 9/10 - Production-ready for mission-critical backups

### Priority 0: Must Fix Before Production (CRITICAL)

#### 6.1: Backup Verification & Integrity ⚠️ CRITICAL

**Problem**: Backups are created but never verified. Corruption goes undetected until restore fails.

**Current State**:
- `verify` command exists but not automatic
- No checksums stored alongside backups
- No detection of bit rot or corruption over time
- Quick verify only checks GPG signature, not data

**Proposed Solutions**:

**Option A: Automatic Post-Backup Verification** (Recommended)
- Compute SHA256 of encrypted backup file after creation
- Store checksum in `.sha256` sidecar file
- Optionally: quick decrypt test of first 1KB
- Add `--skip-verify` flag to disable if needed
- Estimated effort: 1-2 days

**Option B: Separate Verification Step**
- Add `--verify` flag to backup command
- User must explicitly request verification
- Lighter implementation, less safe
- Estimated effort: 0.5 days

**Option C: Full Restore Test** (Most thorough)
- After backup, restore to temp directory
- Compare checksums of all files
- Delete temp directory
- Slow but guarantees backup integrity
- Estimated effort: 2-3 days

**Decision needed**: Which option to implement?

---

#### 6.2: Atomic Backup Writes ⚠️ CRITICAL

**Problem**: Partial backup files created on failure. Race condition in error cleanup.

**Current State**:
```go
defer func() {
    outFile.Close()
    if err != nil {  // Race: err may not be set yet
        os.Remove(outputPath)
    }
}()
```

**Proposed Solutions**:

**Option A: Temp File + Rename** (Recommended)
- Write to `backup_*.tar.gz.gpg.tmp`
- Rename to final name only on success
- Atomic on same filesystem
- Clean up `.tmp` files on startup
- Estimated effort: 0.5 days

**Option B: Write-Ahead Log Pattern**
- Create intent log before backup
- Mark as complete after success
- Recovery process on startup
- More complex, overkill for this use case
- Estimated effort: 2 days

**Option C: Two-Phase Commit**
- Write backup to staging area
- Verify integrity
- Move to final destination
- Estimated effort: 1 day

**Decision needed**: Option A recommended unless specific requirements differ.

---

#### 6.3: Comprehensive Error Propagation ⚠️ CRITICAL

**Problem**: Pipeline goroutines have weak error handling. Only tar errors are captured.

**Current State**:
```go
errChan := make(chan error, 3)
// Only tar goroutine sends to errChan
// Compression/encryption errors lost
if err := <-errChan; err != nil {
    return err
}
```

**Proposed Solutions**:

**Option A: Error Group Pattern** (Recommended)
- Use `golang.org/x/sync/errgroup`
- Captures first error from any goroutine
- Cancels remaining goroutines on error
- Standard Go pattern
- Estimated effort: 1 day

**Option B: Multiple Error Channels**
- Separate channel per pipeline stage
- Check all channels with select
- More manual, more control
- Estimated effort: 1 day

**Option C: Context Cancellation**
- Pass context through pipeline
- Cancel on first error
- Requires refactoring all stages
- Estimated effort: 2-3 days

**Decision needed**: Option A recommended (standard library solution).

---

#### 6.4: Secure Passphrase Handling ⚠️ CRITICAL

**Problem**: Passphrase visible in process list and shell history.

**Current State**:
```bash
# Insecure: visible in `ps aux` and `.bash_history`
secure-backup restore --passphrase "my-secret-pass" ...
```

**Proposed Solutions**:

**Option A: Environment Variable** (Simplest)
- Read from `SECURE_BACKUP_PASSPHRASE` env var
- Document in usage guide
- Still visible in process env, but better
- Estimated effort: 0.25 days

**Option B: Interactive Prompt** (Recommended)
- Use `golang.org/x/term` for secure input
- Prompt user if passphrase not provided
- No echo to terminal
- Best UX for interactive use
- Estimated effort: 0.5 days

**Option C: File-Based Passphrase**
- Read from `--passphrase-file` flag
- User manages file permissions
- Good for automation
- Estimated effort: 0.25 days

**Option D: Hybrid Approach** (Most Flexible)
- Try env var first
- Then passphrase file
- Finally interactive prompt
- Covers all use cases
- Estimated effort: 1 day

**Decision needed**: Option D recommended for maximum flexibility.

---

### Priority 1: Should Fix Soon (HIGH)

#### 6.5: Resource Limits & DoS Protection

**Problem**: No protection against malicious or corrupted archives.

**Current State**:
- No limit on number of files
- No limit on individual file sizes
- No limit on total archive size
- No symlink depth checking
- No filename length validation

**Proposed Solutions**:

**Option A: Configurable Limits** (Recommended)
- Add config for max files, max size, max depth
- Sane defaults (e.g., 1M files, 100GB total, 10 symlink depth)
- `--no-limits` flag to disable
- Estimated effort: 1-2 days

**Option B: Hard-Coded Limits**
- Fixed limits in code
- Simpler, less flexible
- Estimated effort: 0.5 days

**Option C: Progressive Limits**
- Warn at threshold, error at hard limit
- Better UX for edge cases
- Estimated effort: 2 days

**Decision needed**: Which limits are most important?

---

#### 6.6: Backup Locking

**Problem**: Concurrent backups to same destination can conflict.

**Current State**:
- No protection against simultaneous backups
- Race conditions in retention cleanup
- Possible file corruption

**Proposed Solutions**:

**Option A: File-Based Locking** (Recommended)
- Use `flock` or lock file
- Lock destination directory during backup
- Clean up stale locks on startup
- Estimated effort: 0.5 days

**Option B: PID File**
- Write PID to `.backup.lock`
- Check if process still running
- Simpler but less robust
- Estimated effort: 0.25 days

**Option C: No Locking**
- Document that concurrent backups not supported
- User responsibility
- Estimated effort: 0 days

**Decision needed**: Option A recommended for safety.

---

#### 6.7: Restore Safety Checks

**Problem**: Restore silently overwrites existing files.

**Current State**:
```go
outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, ...)
```

**Proposed Solutions**:

**Option A: Require --force Flag** (Recommended)
- Error if destination not empty
- `--force` to override
- Prevents accidental overwrites
- Estimated effort: 0.5 days

**Option B: Interactive Confirmation**
- Prompt user if files exist
- Not suitable for automation
- Estimated effort: 0.5 days

**Option C: Backup Existing Files**
- Move existing files to `.bak` before restore
- Safer but more complex
- Estimated effort: 1 day

**Decision needed**: Option A recommended (standard pattern).

---

#### 6.8: Migrate from Deprecated OpenPGP

**Problem**: `golang.org/x/crypto/openpgp` is deprecated and unmaintained.

**Current State**:
- Using frozen, unmaintained library
- Security vulnerabilities won't be patched
- Works but risky long-term

**Proposed Solutions**:

**Option A: Implement age Encryption** (Recommended)
- Already planned in Phase 3
- Modern, maintained alternative
- Simpler key management
- Support both GPG and age
- Estimated effort: 3-5 days

**Option B: Use ProtonMail's gopenpgp**
- Maintained fork of openpgp
- Drop-in replacement
- Still GPG-based
- Estimated effort: 1-2 days

**Option C: Use External GPG Binary**
- Shell out to `gpg` command
- Most compatible
- Slower, more dependencies
- Estimated effort: 2-3 days

**Decision needed**: Option A recommended (aligns with roadmap).

---

### Priority 2: Nice to Have (MEDIUM)

#### 6.9: Structured Logging & Observability

**Problem**: No structured logging, hard to monitor in production.

**Current State**:
- Printf-style logging
- No log levels
- No JSON output
- No metrics

**Proposed Solutions**:

**Option A: Add slog (Go 1.21+)** (Recommended)
- Standard library structured logging
- Multiple output formats (text, JSON)
- Log levels (debug, info, warn, error)
- Estimated effort: 1-2 days

**Option B: Use logrus or zap**
- More features than slog
- Additional dependency
- Estimated effort: 1-2 days

**Option C: Minimal Structured Output**
- JSON output flag for key events
- No full logging framework
- Estimated effort: 0.5 days

**Decision needed**: Option A recommended (stdlib, future-proof).

---

#### 6.10: Backup Metadata & Manifests

**Problem**: No metadata about backup contents or creation.

**Current State**:
- Only filename contains timestamp
- No source path recorded
- No tool version
- No compression/encryption settings

**Proposed Solutions**:

**Option A: JSON Manifest File** (Recommended)
- Create `backup_*.manifest.json` alongside backup
- Contains: source, timestamp, version, settings, checksum
- Easy to parse and audit
- Estimated effort: 1 day

**Option B: Embedded Metadata**
- Store metadata inside encrypted archive
- More complex to extract
- Estimated effort: 2 days

**Option C: Extended Attributes**
- Use filesystem xattrs
- Not portable across filesystems
- Estimated effort: 1 day

**Decision needed**: Option A recommended (portable, auditable).

---

#### 6.11: Metrics & Performance Tracking

**Problem**: No visibility into backup performance.

**Proposed Solutions**:

**Option A: Basic Metrics**
- Track: duration, size, compression ratio
- Output to stderr or log file
- Estimated effort: 0.5 days

**Option B: Prometheus Metrics**
- Export metrics in Prometheus format
- Requires metrics endpoint
- Overkill for CLI tool
- Estimated effort: 2 days

**Option C: JSON Performance Report**
- Write performance data to `.metrics.json`
- Easy to parse and analyze
- Estimated effort: 1 day

**Decision needed**: Option A or C depending on use case.

---

#### 6.12: Enhanced Testing

**Problem**: Good coverage (83%) but limited edge case testing.

**Proposed Improvements**:
- Chaos testing (random failures in pipeline)
- Large file testing (multi-GB backups)
- Corruption testing (bit flips in backups)
- Performance benchmarks
- Fuzz testing for tar/compression
- Estimated effort: 3-5 days

---

### Implementation Strategy

**Recommended Phases**:

1. **Week 1**: P0 Critical Fixes
   - 6.1: Backup Verification (Option A)
   - 6.2: Atomic Writes (Option A)
   - 6.3: Error Propagation (Option A)
   - 6.4: Secure Passphrase (Option D)

2. **Week 2**: P1 High Priority
   - 6.5: Resource Limits (Option A)
   - 6.6: Backup Locking (Option A)
   - 6.7: Restore Safety (Option A)

3. **Week 3**: P1 Crypto Migration
   - 6.8: Implement age encryption (Option A)

4. **Week 4**: P2 Observability
   - 6.9: Structured Logging (Option A)
   - 6.10: Metadata (Option A)
   - 6.11: Metrics (Option C)

**Total Estimated Effort**: 3-4 weeks for full productionization

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
| 2026-02-08 | Add Phase 6: Productionization | Production readiness analysis revealed critical gaps |
| 2026-02-08 | Merge Phase 3 into 6.8 | age encryption is part of crypto migration |

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

**Last Updated**: 2026-02-08  
**Last Updated By**: Agent (conversation 03ed4df2-03cd-4bb1-9dae-181686ba8fca)  
**Project Phase**: Phase 5 Complete (User Experience), Phase 6 Planned (Productionization)  
**Production Trust Score**: 6.5/10 - Good for personal use, needs Phase 6 for mission-critical data  
**Next Milestone**: Phase 6 Productionization (3-4 weeks estimated)
