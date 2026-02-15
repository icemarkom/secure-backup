package retention

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icemarkom/secure-backup/internal/format"
	"github.com/icemarkom/secure-backup/internal/manifest"
)

// Policy defines retention policy configuration
type Policy struct {
	RetentionDays int
	BackupDir     string
	Pattern       string // File pattern to match (e.g., "backup_*.tar.gz.gpg")
	Verbose       bool
	DryRun        bool
}

// ApplyPolicy removes backups older than the retention period
func ApplyPolicy(policy Policy) (int, error) {
	if policy.RetentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive")
	}

	if policy.Pattern == "" {
		policy.Pattern = "backup_*.tar.gz.gpg"
	}

	// Calculate cutoff time
	cutoffTime := time.Now().AddDate(0, 0, -policy.RetentionDays)

	if policy.Verbose {
		fmt.Printf("Applying retention policy: %d days\n", policy.RetentionDays)
		fmt.Printf("Cutoff time: %s\n", cutoffTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("Backup directory: %s\n", policy.BackupDir)
	}

	// Find matching backup files
	pattern := filepath.Join(policy.BackupDir, policy.Pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("failed to find backup files: %w", err)
	}

	deletedCount := 0

	for _, file := range matches {
		fileInfo, err := os.Stat(file)
		if err != nil {
			if policy.Verbose {
				fmt.Printf("Warning: failed to stat %s: %v\n", file, err)
			}
			continue
		}

		// Skip directories
		if fileInfo.IsDir() {
			continue
		}

		// Check if file is older than retention period
		if fileInfo.ModTime().Before(cutoffTime) {
			age := time.Since(fileInfo.ModTime())
			days := int(age.Hours() / 24)

			if policy.DryRun {
				// Dry-run mode: show what would be deleted
				fmt.Printf("[DRY RUN] Would delete: %s (%d days old)\n",
					filepath.Base(file), days)
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
	}

	if policy.DryRun {
		if deletedCount == 0 {
			fmt.Println("[DRY RUN] No backups would be deleted (all within retention period)")
		} else {
			fmt.Printf("[DRY RUN] Would delete %d old backup(s)\n", deletedCount)
		}
	} else if policy.Verbose {
		if deletedCount == 0 {
			fmt.Println("No backups to delete (all within retention period)")
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
