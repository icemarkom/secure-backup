// Copyright 2026 Marko Milivojevic
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package retention

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/format"
	"github.com/icemarkom/secure-backup/internal/manifest"
)

// DefaultKeepLast is the default retention count (0 = keep all, retention disabled).
const DefaultKeepLast = 0

// Policy defines retention policy configuration
type Policy struct {
	KeepLast  int
	BackupDir string
	Verbose   bool
	DryRun    bool
}

// ApplyPolicy removes backups beyond the retention count, keeping the newest N.
func ApplyPolicy(policy Policy) (int, error) {
	if policy.KeepLast <= 0 {
		return 0, fmt.Errorf("keep count must be positive")
	}

	if policy.Verbose {
		fmt.Printf("Applying retention policy: keep last %d backup(s)\n", policy.KeepLast)
		fmt.Printf("Backup directory: %s\n", policy.BackupDir)
	}

	// Find all backup files using broad glob + IsBackupFile filter
	pattern := filepath.Join(policy.BackupDir, "backup_*")
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
		if !IsBackupFile(filepath.Base(file)) {
			continue
		}
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

// ListBackups lists all backups in the directory with their age.
// Uses broad glob + IsBackupFile filter to find all valid backup files.
func ListBackups(backupDir string) ([]BackupInfo, error) {
	fullPattern := filepath.Join(backupDir, "backup_*")
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

		// Only include files with recognized backup extensions
		if !IsBackupFile(filepath.Base(file)) {
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

// validBackupExtensions is the set of valid backup file suffixes, computed once
// from the cross-product of supported compression and encryption methods.
var validBackupExtensions []string

func init() {
	for _, c := range compress.ValidMethods() {
		comp, err := compress.NewCompressor(compress.Config{Method: c})
		if err != nil {
			continue
		}
		for _, e := range encrypt.ValidMethods() {
			validBackupExtensions = append(validBackupExtensions,
				fmt.Sprintf(".tar%s.%s", comp.Extension(), e.Extension()))
		}
	}
}

// IsBackupFile checks if a filename matches the backup pattern
func IsBackupFile(filename string) bool {
	if !strings.HasPrefix(filename, "backup_") {
		return false
	}
	for _, ext := range validBackupExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}
