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

package backup

import (
	"context"
	"fmt"
	"os"

	"github.com/icemarkom/secure-backup/internal/archive"
	"github.com/icemarkom/secure-backup/internal/common"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/progress"
)

// RestoreConfig holds configuration for restore operations
type RestoreConfig struct {
	BackupFile string
	DestPath   string
	Encryptor  encrypt.Encryptor
	Compressor compress.Compressor
	Verbose    bool
	DryRun     bool
	Force      bool
}

// PerformRestore executes the restore pipeline: DECRYPT → DECOMPRESS → EXTRACT
func PerformRestore(ctx context.Context, cfg RestoreConfig) error {
	// Handle dry-run mode
	if cfg.DryRun {
		return dryRunRestore(cfg)
	}

	// Validate backup file exists
	if _, err := os.Stat(cfg.BackupFile); err != nil {
		if os.IsNotExist(err) {
			return common.MissingFile(cfg.BackupFile,
				"Specify a valid backup file with --file")
		}
		return common.Wrap(err, fmt.Sprintf("Cannot access backup file: %s", cfg.BackupFile),
			"Check file permissions")
	}

	// Check if destination directory is non-empty (safety check)
	nonEmpty, err := isDirectoryNonEmpty(cfg.DestPath)
	if err != nil {
		return common.Wrap(err, fmt.Sprintf("Cannot check destination directory: %s", cfg.DestPath),
			"Check directory permissions")
	}

	if nonEmpty && !cfg.Force {
		return common.New(
			fmt.Sprintf("Destination directory is not empty: %s", cfg.DestPath),
			"Use --force to overwrite existing files (this will replace files with the same names)",
		)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(cfg.DestPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if cfg.Verbose && nonEmpty && cfg.Force {
		fmt.Println("WARNING: Restoring to non-empty directory - existing files may be overwritten")
	}

	if cfg.Verbose {
		fmt.Printf("Restoring from: %s\n", cfg.BackupFile)
		fmt.Printf("Destination: %s\n", cfg.DestPath)
	}

	// Execute the restore pipeline: FILE → DECRYPT → DECOMPRESS → EXTRACT
	if err := executeRestorePipeline(ctx, cfg); err != nil {
		return fmt.Errorf("restore pipeline failed: %w", err)
	}

	if cfg.Verbose {
		fmt.Printf("Restore completed successfully\n")
	}

	return nil
}

// executeRestorePipeline runs the restore pipeline
func executeRestorePipeline(ctx context.Context, cfg RestoreConfig) error {
	// Step 1: Open encrypted backup file
	backupFile, err := os.Open(cfg.BackupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer backupFile.Close()

	// Wrap with progress tracking (measures encrypted bytes read)
	fileInfo, _ := backupFile.Stat()
	var fileSize int64
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}
	pr := progress.NewReader(backupFile, progress.Config{
		Description: "Restoring",
		TotalBytes:  fileSize,
		Enabled:     cfg.Verbose,
	})

	// Step 2: Decrypt the file
	decryptedReader, err := cfg.Encryptor.Decrypt(pr)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// Step 3: Decompress
	decompressedReader, err := cfg.Compressor.Decompress(decryptedReader)
	if err != nil {
		return fmt.Errorf("decompression failed: %w", err)
	}

	// Step 4: Extract tar archive
	if err := archive.ExtractTar(decompressedReader, cfg.DestPath); err != nil {
		pr.Finish()
		return fmt.Errorf("tar extraction failed: %w", err)
	}
	pr.Finish()

	return nil
}

// dryRunRestore previews restore operation without executing
// Note: Dry-run mode always shows verbose output for useful preview
func dryRunRestore(cfg RestoreConfig) error {
	// Validate backup file exists
	fileInfo, err := os.Stat(cfg.BackupFile)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Print dry-run preview (always verbose)
	fmt.Println("[DRY RUN] Restore preview:")
	fmt.Printf("[DRY RUN]   Backup file: %s (%s)\n", cfg.BackupFile, common.Size(fileInfo.Size()))
	fmt.Printf("[DRY RUN]   Destination: %s\n", cfg.DestPath)
	fmt.Println("[DRY RUN]")
	fmt.Println("[DRY RUN] Pipeline stages that would execute:")
	fmt.Printf("[DRY RUN]   - DECRYPT - Decrypt backup file with %s\n", cfg.Encryptor.Type())
	if cfg.Compressor.Type() != compress.None {
		fmt.Printf("[DRY RUN]   - DECOMPRESS - Decompress with %s\n", cfg.Compressor.Type())
	}
	fmt.Println("[DRY RUN]   - EXTRACT - Extract tar archive to destination")

	return nil
}

// isDirectoryNonEmpty checks if a directory exists and is non-empty
func isDirectoryNonEmpty(path string) (bool, error) {
	// Check if directory exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Directory doesn't exist, so it's not non-empty
		}
		return false, err // Some other error
	}

	// Check if it's a directory
	if !info.IsDir() {
		return false, fmt.Errorf("path exists but is not a directory")
	}

	// Check if directory is empty
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	return len(entries) > 0, nil
}
