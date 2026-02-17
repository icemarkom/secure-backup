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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/icemarkom/secure-backup/internal/archive"
	"github.com/icemarkom/secure-backup/internal/common"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/progress"
	"golang.org/x/sync/errgroup"
)

// pipeBufferSize uses the shared IO buffer size for inter-stage pipeline pipes.
// Larger buffers let pipeline stages work more independently, reducing
// goroutine contention from zero-buffered io.Pipe().
const pipeBufferSize = common.IOBufferSize

// Config holds configuration for backup operations
type Config struct {
	SourcePath string
	DestDir    string
	Encryptor  encrypt.Encryptor
	Compressor compress.Compressor
	Verbose    bool
	DryRun     bool
	FileMode   *os.FileMode // nil = use system umask (os.Create); non-nil = explicit permissions
}

// PerformBackup executes the backup pipeline: TAR → COMPRESS → ENCRYPT
func PerformBackup(ctx context.Context, cfg Config) (string, error) {
	// Handle dry-run mode
	if cfg.DryRun {
		return dryRunBackup(cfg)
	}

	// Validate source
	_, err := os.Stat(cfg.SourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", common.MissingFile(cfg.SourcePath,
				"Check that the path exists and you have permission to read it")
		}
		return "", common.Wrap(err, fmt.Sprintf("Cannot access source: %s", cfg.SourcePath),
			"Verify the path and check file permissions")
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(cfg.DestDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	sourceName := filepath.Base(cfg.SourcePath)
	filename := fmt.Sprintf("backup_%s_%s.tar%s.%s",
		sourceName,
		timestamp,
		cfg.Compressor.Extension(),
		cfg.Encryptor.Type())
	outputPath := filepath.Join(cfg.DestDir, filename)
	tmpPath := outputPath + ".tmp"

	if cfg.Verbose {
		fmt.Printf("Starting backup of %s (%s)\n", cfg.SourcePath, common.Size(getDirectorySize(cfg.SourcePath)))
		fmt.Printf("Destination: %s\n", outputPath)
	}

	// Create temp output file for atomic operation
	var outFile *os.File
	if cfg.FileMode != nil {
		outFile, err = os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, *cfg.FileMode)
	} else {
		outFile, err = os.Create(tmpPath)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		outFile.Close()
		// Clean up temp file on error (only if we created it in this session)
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	// Execute the pipeline: TAR → COMPRESS → ENCRYPT → FILE
	if err = executePipeline(ctx, cfg, outFile); err != nil {
		return "", fmt.Errorf("backup pipeline failed: %w", err)
	}

	// Close file before rename (required on some platforms)
	if err = outFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close backup file: %w", err)
	}

	// Atomic rename to final path
	if err = os.Rename(tmpPath, outputPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return "", fmt.Errorf("failed to finalize backup file: %w", err)
	}

	// Get final file size
	finalInfo, _ := os.Stat(outputPath)

	if cfg.Verbose {
		fmt.Printf("Backup completed successfully: %s\n", outputPath)
		if finalInfo != nil {
			fmt.Printf("Backup size: %s\n", common.Size(finalInfo.Size()))
		}
	}

	return outputPath, nil
}

// executePipeline runs the backup pipeline with comprehensive error propagation
func executePipeline(ctx context.Context, cfg Config, output io.Writer) error {
	// Use provided context for pipeline coordination
	g, ctx := errgroup.WithContext(ctx)

	// Step 1: Create tar reader pipe
	tarPR, tarPW := io.Pipe()

	// Goroutine 1: Create TAR archive
	g.Go(func() error {
		defer tarPW.Close()
		if err := archive.CreateTar(cfg.SourcePath, tarPW); err != nil {
			tarPW.CloseWithError(err)
			return fmt.Errorf("tar creation failed: %w", err)
		}
		return nil
	})

	// Step 2: Compress the tar stream (with buffered reader to reduce pipe contention)
	// Note: Compressor.Compress spawns its own goroutine internally
	bufferedTarPR := bufio.NewReaderSize(tarPR, pipeBufferSize)

	// Wrap with progress tracking (measures source bytes read through tar)
	pr := progress.NewReader(bufferedTarPR, progress.Config{
		Description: "Backing up",
		TotalBytes:  getDirectorySize(cfg.SourcePath),
		Enabled:     cfg.Verbose,
	})

	compressPR, err := cfg.Compressor.Compress(pr)
	if err != nil {
		return fmt.Errorf("failed to create compressor: %w", err)
	}

	// Step 3: Encrypt the compressed stream
	// Note: Encryptor.Encrypt spawns its own goroutine internally
	encryptPR, err := cfg.Encryptor.Encrypt(compressPR)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	// Step 4: Write encrypted stream to output file
	// This will capture errors from compress/encrypt goroutines via pipe errors
	if _, err := io.CopyBuffer(output, encryptPR, common.NewBuffer()); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	pr.Finish()

	// Wait for tar goroutine to complete
	// Any errors from compress/encrypt will have already been caught by io.Copy above
	return g.Wait()
}

// getDirectorySize calculates the total size of a directory (best effort)
func getDirectorySize(path string) int64 {
	var size int64
	filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// dryRunBackup previews backup operation without executing
// Note: Dry-run mode always shows verbose output for useful preview
func dryRunBackup(cfg Config) (string, error) {
	// Validate source exists
	_, err := os.Stat(cfg.SourcePath)
	if err != nil {
		return "", fmt.Errorf("invalid source path: %w", err)
	}

	// Generate backup filename (same logic as real backup)
	timestamp := time.Now().Format("20060102_150405")
	sourceName := filepath.Base(cfg.SourcePath)
	filename := fmt.Sprintf("backup_%s_%s.tar%s.%s",
		sourceName,
		timestamp,
		cfg.Compressor.Extension(),
		cfg.Encryptor.Type())
	outputPath := filepath.Join(cfg.DestDir, filename)

	// Calculate source size
	sourceSize := getDirectorySize(cfg.SourcePath)

	encType := cfg.Encryptor.Type()

	// Print dry-run preview (always verbose)
	fmt.Println("[DRY RUN] Backup preview:")
	fmt.Printf("[DRY RUN]   Source: %s (%s)\n", cfg.SourcePath, common.Size(sourceSize))
	fmt.Printf("[DRY RUN]   Destination: %s\n", outputPath)
	fmt.Printf("[DRY RUN]   Compression: %s\n", cfg.Compressor.Type())
	fmt.Printf("[DRY RUN]   Encryption: %s\n", encType)
	fmt.Println("[DRY RUN]")
	fmt.Println("[DRY RUN] Pipeline stages that would execute:")
	fmt.Println("[DRY RUN]   - TAR - Archive source directory")
	if cfg.Compressor.Type() != compress.None {
		fmt.Printf("[DRY RUN]   - COMPRESS - Compress with %s\n", cfg.Compressor.Type())
	}
	fmt.Printf("[DRY RUN]   - ENCRYPT - Encrypt with %s\n", encType)
	fmt.Println("[DRY RUN]   - WRITE - Write to destination file")

	return outputPath, nil
}
