# Man Page Discussion for secure-backup

## Overview

Man pages are traditional Unix/Linux documentation accessed via `man secure-backup`. This document discusses whether we should create man pages and the best approach.

## Pros of Man Pages

✅ **Standard Unix Convention**: Users expect `man` for CLI tools
✅ **Offline Access**: Available without internet connection
✅ **Quick Reference**: Faster than opening markdown files
✅ **Shell Integration**: Tab completion can reference man pages
✅ **Professional Polish**: Signals a mature, production-ready tool

## Cons of Man Pages

❌ **Maintenance Overhead**: Must keep in sync with markdown docs
❌ **Format Complexity**: Troff/groff syntax is arcane
❌ **Limited Formatting**: Not as rich as markdown
❌ **Installation Required**: Must install to `/usr/share/man/` or similar
❌ **Not Version Controlled Well**: Binary-like format

## Recommended Approach: Hybrid Solution

### Phase 1: Use Markdown as Source of Truth (Current)

**Keep USAGE.md as the primary documentation** because:
- Easier to maintain (markdown is readable)
- Version controlled naturally with Git
- Can be viewed on GitHub, locally, or in editors
- Modern users are comfortable with markdown

### Phase 2: Auto-Generate Man Pages from Markdown

Use tools to convert USAGE.md → man pages automatically:

#### Option A: go-md2man (Recommended)

```bash
# Install
go install github.com/cpuguy83/go-md2man/v2@latest

# Generate man page from markdown
go-md2man -in USAGE.md -out secure-backup.1

# Test
man ./secure-backup.1

# Install system-wide
sudo cp secure-backup.1 /usr/share/man/man1/
sudo mandb  # Update man database
```

**Pros**:
- Written in Go (matches our stack)
- Actively maintained
- Good markdown → troff conversion

#### Option B: pandoc

```bash
# Install pandoc
apt-get install pandoc

# Generate
pandoc USAGE.md -s -t man -o secure-backup.1
```

**Pros**:
- Very powerful, handles complex markdown
- Widely available

**Cons**:
- Adds installation dependency

### Phase 3: Integrate into Build Process

Add to Makefile or build script:

```makefile
# Makefile
.PHONY: all build man install

all: build man

build:
	go build -o secure-backup .

man:
	go-md2man -in USAGE.md -out secure-backup.1

install: build man
	sudo install -m 755 secure-backup /usr/local/bin/
	sudo install -m 644 secure-backup.1 /usr/share/man/man1/
	sudo mandb

clean:
	rm -f secure-backup secure-backup.1
```

### Phase 4: Package Manager Integration

When creating packages (deb, rpm, etc.), include man pages:

**Debian package** (`debian/secure-backup.manpages`):
```
secure-backup.1
```

**RPM spec**:
```spec
%files
/usr/bin/secure-backup
/usr/share/man/man1/secure-backup.1.gz
```

## Alternative: Cobra Auto-Generated Man Pages

Cobra can generate man pages directly from command definitions:

```go
// In cmd/root.go or separate doc generation command
import "github.com/spf13/cobra/doc"

func generateManPages() {
	header := &doc.GenManHeader{
		Title:   "BACKUP-DOCKER",
		Section: "1",
	}
	
	err := doc.GenManTree(rootCmd, header, "./man")
	if err != nil {
		log.Fatal(err)
	}
}
```

**Pros**:
- Auto-generated from actual command definitions
- Always in sync with code
- Cobra handles formatting

**Cons**:
- Less detailed than hand-written docs
- Doesn't include examples and use cases

## My Recommendation

### For Now (Phase 1 Complete)
✅ **Keep USAGE.md as primary documentation**
- It's comprehensive and user-friendly
- Easy to maintain alongside code
- Works great for GitHub and local viewing

### Next Steps (When Packaging for Distribution)
📦 **Add man page generation when creating packages**
1. Use `go-md2man` to generate from USAGE.md
2. Add to build/install process
3. Include in deb/rpm packages

### Hybrid Approach
Create **separate, focused man pages** for each command:

```
man secure-backup          # Overview and quick reference
man secure-backup-backup   # backup command details
man secure-backup-restore  # restore command details
man secure-backup-verify   # verify command details
man secure-backup-list     # list command details
```

This mirrors Docker's approach (`man docker`, `man docker-run`, etc.).

## Implementation Priority

**Priority**: 🟡 Medium (Phase 2-3 activity)

**Reasoning**:
- Phase 1 is feature-complete with good markdown docs
- Man pages are important for distribution but not critical for development
- Should be added when preparing for package managers (apt, yum, etc.)
- Can be auto-generated, so low maintenance burden if done right

## Example Man Page Structure

If we create man pages, they should follow this standard structure:

```
NAME
    secure-backup - secure, encrypted backups for directories and Docker volumes

SYNOPSIS
    secure-backup COMMAND [OPTIONS]

DESCRIPTION
    A high-performance backup tool that creates encrypted, compressed archives.
    
    Uses a TAR → COMPRESS → ENCRYPT pipeline for optimal efficiency.

COMMANDS
    backup      Create an encrypted backup
    restore     Restore from a backup
    verify      Verify backup integrity
    list        List available backups

OPTIONS
    -h, --help
        Display help information

EXAMPLES
    Create a backup:
        secure-backup backup --source /data --dest /backups --public-key key.asc
    
    Restore a backup:
        secure-backup restore --file backup.tar.gz.gpg --dest /restore --private-key key.asc

SEE ALSO
    tar(1), gzip(1), gpg(1)

AUTHOR
    Marko M. <markom@gmail.com>

REPORTING BUGS
    https://github.com/icemarkom/secure-backup/issues
```

## Decision

**Recommended**: Implement man pages in **Phase 2 or 3**, using `go-md2man` to auto-generate from USAGE.md.

**For now**: USAGE.md is sufficient and excellent documentation.

Would you like me to implement man page generation now, or defer it to a later phase?
