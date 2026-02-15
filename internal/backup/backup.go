package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/icemarkom/secure-backup/internal/archive"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/errors"
	"golang.org/x/sync/errgroup"
)

// Config holds configuration for backup operations
type Config struct {
	SourcePath string
	DestDir    string
	Encryptor  encrypt.Encryptor
	Compressor compress.Compressor
	Verbose    bool
	DryRun     bool
}

// PerformBackup executes the backup pipeline: TAR → COMPRESS → ENCRYPT
func PerformBackup(cfg Config) (string, error) {
	// Handle dry-run mode
	if cfg.DryRun {
		return dryRunBackup(cfg)
	}

	// Validate source
	_, err := os.Stat(cfg.SourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.MissingFile(cfg.SourcePath,
				"Check that the path exists and you have permission to read it")
		}
		return "", errors.Wrap(err, fmt.Sprintf("Cannot access source: %s", cfg.SourcePath),
			"Verify the path and check file permissions")
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(cfg.DestDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	sourceName := filepath.Base(cfg.SourcePath)
	filename := fmt.Sprintf("backup_%s_%s.tar%s.gpg",
		sourceName,
		timestamp,
		cfg.Compressor.Extension())
	outputPath := filepath.Join(cfg.DestDir, filename)
	tmpPath := outputPath + ".tmp"

	if cfg.Verbose {
		fmt.Printf("Starting backup of %s (%s)\n", cfg.SourcePath, formatSize(getDirectorySize(cfg.SourcePath)))
		fmt.Printf("Destination: %s\n", outputPath)
	}

	// Create temp output file for atomic operation
	outFile, err := os.Create(tmpPath)
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
	if err = executePipeline(cfg, outFile); err != nil {
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
			fmt.Printf("Backup size: %s\n", formatSize(finalInfo.Size()))
		}
	}

	return outputPath, nil
}

// executePipeline runs the backup pipeline with comprehensive error propagation
func executePipeline(cfg Config, output io.Writer) error {
	// Create context for pipeline coordination
	ctx := context.Background()
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

	// Step 2: Compress the tar stream
	// Note: Compressor.Compress spawns its own goroutine internally
	compressPR, err := cfg.Compressor.Compress(tarPR)
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
	if _, err := io.Copy(output, encryptPR); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

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

// formatSize formats bytes as human-readable string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
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
	filename := fmt.Sprintf("backup_%s_%s.tar%s.gpg",
		sourceName,
		timestamp,
		cfg.Compressor.Extension())
	outputPath := filepath.Join(cfg.DestDir, filename)

	// Calculate source size
	sourceSize := getDirectorySize(cfg.SourcePath)

	// Print dry-run preview (always verbose)
	fmt.Println("[DRY RUN] Backup preview:")
	fmt.Printf("[DRY RUN]   Source: %s (%s)\n", cfg.SourcePath, formatSize(sourceSize))
	fmt.Printf("[DRY RUN]   Destination: %s\n", outputPath)
	fmt.Printf("[DRY RUN]   Compression: %s\n", "gzip")
	fmt.Printf("[DRY RUN]   Encryption: GPG\n")
	fmt.Println("[DRY RUN]")
	fmt.Println("[DRY RUN] Pipeline stages that would execute:")
	fmt.Println("[DRY RUN]   1. TAR - Archive source directory")
	fmt.Println("[DRY RUN]   2. COMPRESS - Compress with gzip")
	fmt.Println("[DRY RUN]   3. ENCRYPT - Encrypt with GPG")
	fmt.Println("[DRY RUN]   4. WRITE - Write to destination file")

	return outputPath, nil
}
