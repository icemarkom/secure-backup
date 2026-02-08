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

## Future Phases

**Phase 3**: Enhanced Encryption
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
**Last Updated By**: Agent (conversation d81fd923-e470-4ee8-b379-ef77b291ed87)  
**Project Phase**: Phase 5 Complete (User Experience)  
**Next Milestone**: Phase 3 (Enhanced Encryption) or Phase 4 (Advanced Compression)
