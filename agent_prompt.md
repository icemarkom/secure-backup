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
**Philosophy**: Simplicity over features. No config files. Unattended operation with security.

---

## ⚠️ Agent Rules of Engagement (MANDATORY)

> **First step on any task**: Read this file, check `task.md` for in-progress items, review recent commits, and ask clarifying questions — don't assume.

### Git Workflow

- **MUST** use feature branches and Pull Requests for all changes. **NEVER push directly to `main`.**
- **NEVER** auto-commit without explicit user instruction. Stage with `git add`, then ask: "Ready to commit?"
- **NEVER** merge PRs without explicit user instruction — the user decides when to merge.
- **ALWAYS** wait for local review before push — present changes for user review, no exceptions.
- **Branch naming**: `fix/issue-description`, `feat/feature-name`, `release/vX.Y.Z`
- **Auto-close**: Use `Fixes #N` or `Closes #N` in PR descriptions to auto-close issues on merge.
- **Issue tracking**: When a new idea or improvement is discussed and agreed upon, **ask the user** whether to open a GitHub issue for it — unless they've already explicitly requested one.

**Required workflow**:
```
BRANCH → MAKE → STAGE → REVIEW → WAIT → COMMIT → PUSH → PR → ASK → WAIT → MERGE
```

### Versioning

- **Format**: `vMAJOR.MINOR.PATCH` (Semantic Versioning)
- On release request: bump **PATCH** by default
- **MAJOR/MINOR** bumps: only on explicit user instruction

### Documentation Updates

When commands, flags, or behavior change, **ALWAYS update ALL of these**:
- `agent_prompt.md` — this file (source of truth for agents)
- `docs/secure-backup.1` — man page (must stay in sync with CLI)
- `USAGE.md` — detailed usage guide
- `README.md` — overview and quick-start

Document current state, not plans. Update this file and `task.md` before finishing any task. Add to Decision Log if architectural decisions were made.

### Testing Standards

- Use `want` for expected values and `got` for actual values (standard Go convention)
- Unit tests first, then integration tests. Test thoroughly, maintain coverage.
- Test keys: **ALWAYS** generated via `test-scripts/generate_test_keys.sh`, **NEVER** checked into git
- Prefer native Go over `os.Exec` calls
- Interface-based design for extensibility

---

## Capabilities

**Commands:**

| Command | Description |
|---------|-------------|
| `backup` | TAR → COMPRESS → ENCRYPT pipeline |
| `restore` | DECRYPT → DECOMPRESS → EXTRACT pipeline |
| `verify` | Integrity checking (quick & full modes) |
| `list` | View available backups |
| `version` | Show version info |

**Encryption**: GPG (`--encryption gpg`) and AGE (`--encryption age`) via `Encryptor` interface
**Compression**: Gzip, zstd, lz4 (`--compression gzip|zstd|lz4|none`) via `Compressor` interface
**Architecture**: Streaming I/O (constant 10-50MB memory, 1 MB buffered pipes)

**Key features:**
- Manifest-based integrity verification (SHA256 checksum, enabled by default, `--skip-manifest` to disable)
- Atomic backup writes (temp file + rename)
- Comprehensive error propagation via `errgroup`
- Secure passphrase handling: `--passphrase` (with security warning) | `SECURE_BACKUP_PASSPHRASE` env var | `--passphrase-file` (mutually exclusive)
- Per-destination backup locking (`.backup.lock`, fail loudly, manual cleanup)
- Restore safety checks (`--force` required for non-empty destinations)
- Count-based retention (`--retention N` keeps last N backups)
- Dry-run mode (`--dry-run` on backup, restore, verify — implies verbose, no side effects)
- Silent by default, `--verbose` for progress bars and details
- Path traversal protection, symlink preservation in tar
- Signal handling (SIGTERM/SIGINT) with context propagation
- Configurable file permissions (`--file-mode`, default 0600)
- License headers enforced via CI (`make license-check`)

**Build & Release:**
- Makefile with dev targets (build, test, coverage, lint)
- GoReleaser: multi-platform builds (linux/darwin/windows × amd64/arm64), .deb packages
- GitHub Actions CI/CD with `paths-ignore` for docs-only changes
- E2E tests: `e2e/e2e_test.sh` (POSIX shell, full backup→list→verify→restore→diff cycle)

---

## Architecture

### Pipeline Order (Critical)

```
BACKUP:  Source → TAR → COMPRESS → ENCRYPT → File
RESTORE: File → DECRYPT → DECOMPRESS → EXTRACT → Destination
```

**Why compress before encrypt**: Encrypted data is cryptographically random and **cannot be compressed**. Wrong order = 0% compression.

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
│   ├── compress/          # Compression (gzip, zstd, lz4, none)
│   ├── encrypt/           # Encryption (GPG, AGE)
│   ├── lock/              # Backup locking (per-destination)
│   ├── manifest/          # Backup metadata & integrity verification
│   ├── passphrase/        # Secure passphrase handling (flag/env/file)
│   ├── progress/          # Progress tracking
│   └── retention/         # Retention management
├── docs/                  # Man page (secure-backup.1)
├── examples/              # Usage examples (cron.daily script)
├── test-scripts/          # Test scripts (key generation, E2E)
├── test_data/             # Generated test data (keys, gitignored)
├── main.go                # Entry point
├── README.md              # User documentation
├── USAGE.md               # Detailed usage guide
└── agent_prompt.md        # This file
```

### Key Design Patterns

1. **Interface-Based Extensibility** — `Compressor` and `Encryptor` interfaces with iota-based `Method` types, `ParseMethod()`, `ValidMethods()`, `Extension()` helpers
2. **Streaming Everything** — Constant memory via `io.Pipe()`, 1 MB buffered I/O (`common.IOBufferSize`)
3. **Security First** — Path traversal protection, GPG symlink validation, 0600 default permissions
4. **Minimal Dependencies** — stdlib + cobra + testify + pgzip + zstd + lz4 + age

---

## Testing

**Philosophy**: Pragmatic testing — unit tests for coverage, integration tests for confidence.

**Two-phase approach:**
1. **Unit Tests** — Fast, focused, individual functions and error paths
2. **Integration Tests** — Full backup → restore → verify cycles with real GPG/AGE

**Test with REAL operations** — real GPG encryption/decryption, real compression, real tar archives. Auto-generated test keys via `test-scripts/generate_test_keys.sh`.

**E2E tests**: `e2e/e2e_test.sh` — POSIX shell script exercising the compiled binary through full pipelines for all compression × encryption combinations, plus dry-run regression tests.

**All new shared helpers go in `internal/common`** — do not create new utility packages.

---

## File Naming Convention

**Format**: `backup_{sourcename}_{timestamp}.tar.{compression}.{encryption}`

| Example | Compression | Encryption |
|---------|-------------|------------|
| `backup_docs_20260207_165324.tar.gz.gpg` | gzip | GPG |
| `backup_docs_20260207_165324.tar.zst.age` | zstd | AGE |
| `backup_docs_20260207_165324.tar.lz4.gpg` | lz4 | GPG |
| `backup_docs_20260207_165324.tar.gpg` | none | GPG |

Manifest: `backup_docs_20260207_165324_manifest.json` (sidecar, `_manifest.json` suffix)

---

## Important Context

### Why GPG?

GPG via `golang.org/x/crypto/openpgp` was the original design choice — standard, widely supported, proven. AGE (`filippo.io/age`) was added as a modern alternative. Both are fully supported. **Do not flag the openpgp library as a risk** — it is functional and intentional.

### Compression Before Encryption

Encrypted data is cryptographically random and incompressible. The pipeline **must** compress before encrypting. This is enforced by architecture — see Pipeline Order above.

---

## Decision Log

Key architectural decisions that affect future development:

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-02-07 | Apache 2.0 license | Permissive, business-friendly |
| 2026-02-08 | Interface-based compression/encryption | Easy to add new methods via `Compressor`/`Encryptor` interfaces |
| 2026-02-08 | Streaming architecture | Constant memory usage, handles any backup size |
| 2026-02-08 | No config files | Simplicity, all config via CLI flags and env vars |
| 2026-02-08 | Fail loudly on conflicts | No silent recovery; print errors with context and exit |
| 2026-02-08 | No automatic cleanup | User deals with stale `.tmp` and `.lock` files manually |
| 2026-02-08 | Manifest enabled by default | `--skip-manifest` to disable; verify-by-default philosophy |
| 2026-02-08 | Passphrase priority | `--passphrase` (with warning) → `SECURE_BACKUP_PASSPHRASE` env → `--passphrase-file` (mutually exclusive) |
| 2026-02-15 | Binary GPG output | Removed ASCII armor (breaking change). Auto-detects armored input on decrypt for backward compat |
| 2026-02-15 | pgzip for parallel compression | `klauspost/pgzip` replaces stdlib gzip for multi-core performance |
| 2026-02-15 | Count-based retention | `--retention N` = keep last N backups (not time-based) |
| 2026-02-15 | Branch+PR workflow | All changes through feature branches and PRs, never direct to main |
| 2026-02-16 | Iota-based Method types | `compress.Method` and `encrypt.Method` with `ParseMethod()`, `String()`, `Extension()` |
| 2026-02-16 | Dynamic retention patterns | Broad `backup_*` glob + `IsBackupFile()` filtering (not extension-specific) |
| 2026-02-17 | All shared helpers in `internal/common` | Consolidated from `internal/format`, `internal/ioutil`, `internal/errors` |
| 2026-02-18 | Docker integration abandoned | General-purpose directory backup tool; Docker volumes backed up via mount paths. Closed [#16](https://github.com/icemarkom/secure-backup/issues/16) |
| 2026-02-18 | agent_prompt.md streamlined | Removed phases/productionization terminology, use GH issues for tracking. Rules of engagement elevated to top. Reduced 1008→232 lines |
| 2026-02-18 | Sidecar-only manifests (no embedding) | Evaluated 4 embedding strategies (tar entry, trailing footer, outer wrapper, hybrid). All add complexity/fragility or break format compatibility with standard GPG/AGE tools. Sidecar is simple, reliable, and preserves `gpg --decrypt` fallback. Closed [#44](https://github.com/icemarkom/secure-backup/issues/44) |
| 2026-02-18 | Manifest-first backup management | Retention scoped by `(hostname, source_path)` from manifest. List partitions into managed/orphan sections. Orphans excluded from retention with stderr warning. Resolves [#45](https://github.com/icemarkom/secure-backup/issues/45) and [#43](https://github.com/icemarkom/secure-backup/issues/43) |

---

## Open Issues

Track all planned work via [GitHub Issues](https://github.com/icemarkom/secure-backup/issues).

Key open items:
- [#65](https://github.com/icemarkom/secure-backup/issues/65) — Adopt subcommand for orphan backups
- [#67](https://github.com/icemarkom/secure-backup/issues/67) — Ctrl+C (SIGINT) does not interrupt running pipelines
- [#68](https://github.com/icemarkom/secure-backup/issues/68) — Replace deprecated `golang.org/x/crypto/openpgp` with `github.com/ProtonMail/go-crypto/openpgp` (security)
- [#69](https://github.com/icemarkom/secure-backup/issues/69) — Validate symlink targets in `ExtractTar` to prevent symlink-chained path traversal (security)
- [#70](https://github.com/icemarkom/secure-backup/issues/70) — `quickVerify` does not detect AGE file format (bug)
- [#71](https://github.com/icemarkom/secure-backup/issues/71) — Retention should sort by manifest `CreatedAt`, not filesystem `ModTime` (bug)
- [#72](https://github.com/icemarkom/secure-backup/issues/72) — `common.Age()` returns `"0m"` for durations under one minute (enhancement)
- [#73](https://github.com/icemarkom/secure-backup/issues/73) — Remove empty `internal/docker` package (tech-debt)
- [#74](https://github.com/icemarkom/secure-backup/issues/74) — Log error when `lock.Release()` fails in deferred cleanup (enhancement)
- [#75](https://github.com/icemarkom/secure-backup/issues/75) — Replace `filepath.Walk` with `filepath.WalkDir` in `getDirectorySize` (tech-debt)

---

**Last Updated**: 2026-02-18
**Current Release**: v1.4.0
**Documentation**: See [USAGE.md](USAGE.md) and [README.md](README.md) for detailed usage and examples.
