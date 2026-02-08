# Contributing to secure-backup

Thank you for your interest in contributing to secure-backup!

## Development Setup

### Prerequisites

- Go 1.21 or later
- GPG for testing encryption features
- make (optional, but recommended)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/icemarkom/secure-backup.git
cd secure-backup

# Install dependencies
go mod download

# Build
make build

# Run tests
make test
```

## Development Workflow

### Building

```bash
# Quick build
make build

# Development build (fmt + vet + test + build)
make dev
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run specific package tests
go test ./internal/compress -v
```

### Code Quality

Before submitting a PR, ensure:

```bash
# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Run vet
make vet

# Run all tests
make test
```

## Project Structure

```
secure-backup/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command + version
│   ├── backup.go          # backup subcommand
│   ├── restore.go         # restore subcommand
│   ├── verify.go          # verify subcommand
│   └── list.go            # list subcommand
├── internal/
│   ├── archive/           # TAR operations
│   ├── compress/          # Compression (gzip, future: zstd)
│   ├── encrypt/           # Encryption (GPG, future: age)
│   ├── backup/            # Pipeline orchestration
│   └── retention/         # Retention management
├── main.go                # Entry point
└── Makefile              # Build automation
```

## Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `go vet` before committing
- Aim for 60%+ test coverage for new code

### Interfaces

The project uses interface-based design for extensibility:

```go
// Example: Adding a new compressor
type Compressor interface {
    Compress(src io.Reader, dest io.Writer) error
    Decompress(src io.Reader, dest io.Writer) error
}
```

When adding new encryption/compression methods, implement the appropriate interface.

### Pipeline Architecture

**CRITICAL**: The backup pipeline order is:

```
BACKUP:  TAR → COMPRESS → ENCRYPT
RESTORE: DECRYPT → DECOMPRESS → EXTRACT
```

This order is **not negotiable** - encrypted data cannot be compressed.

## Testing Philosophy

- **Unit tests**: Test individual functions and methods
- **Integration tests**: Test pipeline flows (backup → restore → verify)
- **Mock interfaces**: Use mocks for interface testing (see `backup_test.go`)

### Test Coverage Goals

- Overall: 60%+
- Critical paths (pipelines): 80%+
- New features: Must include tests

## Submitting Changes

### Pull Request Process

1. **Fork** the repository
2. **Create a branch**: `git checkout -b feature/my-feature`
3. **Write code** with tests
4. **Test locally**: `make dev`
5. **Commit** with clear messages (see below)
6. **Push** to your fork
7. **Create PR** with description

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `refactor`: Code change that neither fixes nor adds feature
- `test`: Adding or updating tests
- `chore`: Maintenance (dependencies, build, etc.)

**Examples:**
```
feat(compress): Add zstd compression support

- Implement ZstdCompressor
- Add compression benchmarks
- Update documentation

Closes #42

fix(restore): Handle empty tar archives correctly

Previously would panic on empty archives. Now returns
clear error message.

docs: Update README installation instructions
```

## Feature Development Guidelines

### Adding New Encryption Methods

1. Implement `Encryptor` interface in `internal/encrypt/`
2. Add factory support in `NewEncryptor()`
3. Add CLI flag in `cmd/backup.go` and `cmd/restore.go`
4. Add tests (unit + integration)
5. Update documentation

### Adding New Compression Methods

1. Implement `Compressor` interface in `internal/compress/`
2. Add factory support in `NewCompressor()`
3. Add CLI flag in `cmd/backup.go`
4. Add tests with compression ratio benchmarks
5. Update documentation

### Adding New Storage Backends

1. Design `StorageBackend` interface
2. Implement for specific backend
3. Add configuration support
4. Add tests (can use mocks for remote services)
5. Update documentation

## Questions or Issues?

- **Bug reports**: Open an [issue](https://github.com/icemarkom/secure-backup/issues)
- **Feature requests**: Open an [issue](https://github.com/icemarkom/secure-backup/issues) with `[Feature]` prefix
- **Questions**: Open a [discussion](https://github.com/icemarkom/secure-backup/discussions)

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
