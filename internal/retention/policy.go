package retention

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/icemarkom/secure-backup/internal/format"
	"github.com/icemarkom/secure-backup/internal/manifest"
)

// DefaultKeepLast is the default retention count (0 = keep all, retention disabled).
const DefaultKeepLast = 0

// Policy defines retention policy configuration
type Policy struct {
	KeepLast  int
	BackupDir string
	Pattern   string // File pattern to match (e.g., "backup_*.tar.gz.gpg")
	Verbose   bool
	DryRun    bool
}

// ApplyPolicy removes backups beyond the retention count, keeping the newest N.
func ApplyPolicy(policy Policy) (int, error) {
	if policy.KeepLast <= 0 {
		return 0, fmt.Errorf("keep count must be positive")
	}

	if policy.Pattern == "" {
		policy.Pattern = "backup_*.tar.gz.gpg"
	}

	if policy.Verbose {
		fmt.Printf("Applying retention policy: keep last %d backup(s)\n", policy.KeepLast)
		fmt.Printf("Backup directory: %s\n", policy.BackupDir)
	}

	// Find matching backup files
	pattern := filepath.Join(policy.BackupDir, policy.Pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("failed to find backup files: %w", err)
	}

	// Collect file info for sorting
	type fileEntry struct {
		path    string
		modTime time.Time
	}
	var files []fileEntry
	for _, file := range matches {
		fileInfo, err := os.Stat(file)
		if err != nil {
			if policy.Verbose {
				fmt.Printf("Warning: failed to stat %s: %v\n", file, err)
			}
			continue
		}
		if fileInfo.IsDir() {
			continue
		}
		files = append(files, fileEntry{path: file, modTime: fileInfo.ModTime()})
	}

	// Sort by modification time, newest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	// Keep first N, delete the rest
	deletedCount := 0
	for i := policy.KeepLast; i < len(files); i++ {
		file := files[i].path
		age := time.Since(files[i].modTime)

		if policy.DryRun {
			fmt.Printf("[DRY RUN] Would delete: %s (age: %s)\n",
				filepath.Base(file), format.Age(age))
			// Check for associated manifest
			manifestPath := manifest.ManifestPath(file)
			if _, err := os.Stat(manifestPath); err == nil {
				fmt.Printf("[DRY RUN] Would delete manifest: %s\n",
					filepath.Base(manifestPath))
			}
			deletedCount++
			continue
		}

		if policy.Verbose {
			fmt.Printf("Deleting old backup: %s (age: %s)\n",
				filepath.Base(file),
				format.Age(age))
		}

		if err := os.Remove(file); err != nil {
			if policy.Verbose {
				fmt.Printf("Warning: failed to delete %s: %v\n", file, err)
			}
			continue
		}

		// Also delete associated manifest file
		manifestPath := manifest.ManifestPath(file)
		if err := os.Remove(manifestPath); err == nil {
			if policy.Verbose {
				fmt.Printf("Deleted manifest: %s\n", filepath.Base(manifestPath))
			}
		}

		deletedCount++
	}

	if policy.DryRun {
		if deletedCount == 0 {
			fmt.Printf("[DRY RUN] No backups to delete (have %d, keeping %d)\n", len(files), policy.KeepLast)
		} else {
			fmt.Printf("[DRY RUN] Would delete %d backup(s)\n", deletedCount)
		}
	} else if policy.Verbose {
		if deletedCount == 0 {
			fmt.Printf("No backups to delete (have %d, keeping %d)\n", len(files), policy.KeepLast)
		} else {
			fmt.Printf("Deleted %d old backup(s)\n", deletedCount)
		}
	}

	return deletedCount, nil
}

// ListBackups lists all backups in the directory with their age
func ListBackups(backupDir string, pattern string) ([]BackupInfo, error) {
	if pattern == "" {
		pattern = "backup_*.tar.gz.gpg"
	}

	fullPattern := filepath.Join(backupDir, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find backup files: %w", err)
	}

	var backups []BackupInfo

	for _, file := range matches {
		fileInfo, err := os.Stat(file)
		if err != nil {
			continue
		}

		if fileInfo.IsDir() {
			continue
		}

		backups = append(backups, BackupInfo{
			Path:    file,
			Name:    filepath.Base(file),
			Size:    fileInfo.Size(),
			ModTime: fileInfo.ModTime(),
			Age:     time.Since(fileInfo.ModTime()),
		})
	}

	return backups, nil
}

// BackupInfo contains information about a backup file
type BackupInfo struct {
	Path    string
	Name    string
	Size    int64
	ModTime time.Time
	Age     time.Duration
}

// IsBackupFile checks if a filename matches the backup pattern
func IsBackupFile(filename string) bool {
	return strings.HasPrefix(filename, "backup_") &&
		(strings.HasSuffix(filename, ".tar.gz.gpg") ||
			strings.HasSuffix(filename, ".tar.zst.gpg") ||
			strings.HasSuffix(filename, ".tar.gz.age"))
}
