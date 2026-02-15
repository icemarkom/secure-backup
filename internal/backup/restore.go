package backup

import (
	"fmt"
	"os"

	"github.com/icemarkom/secure-backup/internal/archive"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/errors"
	"github.com/icemarkom/secure-backup/internal/format"
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
func PerformRestore(cfg RestoreConfig) error {
	// Handle dry-run mode
	if cfg.DryRun {
		return dryRunRestore(cfg)
	}

	// Validate backup file exists
	if _, err := os.Stat(cfg.BackupFile); err != nil {
		if os.IsNotExist(err) {
			return errors.MissingFile(cfg.BackupFile,
				"Specify a valid backup file with --file")
		}
		return errors.Wrap(err, fmt.Sprintf("Cannot access backup file: %s", cfg.BackupFile),
			"Check file permissions")
	}

	// Check if destination directory is non-empty (safety check)
	nonEmpty, err := isDirectoryNonEmpty(cfg.DestPath)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Cannot check destination directory: %s", cfg.DestPath),
			"Check directory permissions")
	}

	if nonEmpty && !cfg.Force {
		return errors.New(
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
	if err := executeRestorePipeline(cfg); err != nil {
		return fmt.Errorf("restore pipeline failed: %w", err)
	}

	if cfg.Verbose {
		fmt.Printf("Restore completed successfully\n")
	}

	return nil
}

// executeRestorePipeline runs the restore pipeline
func executeRestorePipeline(cfg RestoreConfig) error {
	// Step 1: Open encrypted backup file
	backupFile, err := os.Open(cfg.BackupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer backupFile.Close()

	// Step 2: Decrypt the file
	decryptedReader, err := cfg.Encryptor.Decrypt(backupFile)
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
		return fmt.Errorf("tar extraction failed: %w", err)
	}

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
	fmt.Printf("[DRY RUN]   Backup file: %s (%s)\n", cfg.BackupFile, format.Size(fileInfo.Size()))
	fmt.Printf("[DRY RUN]   Destination: %s\n", cfg.DestPath)
	fmt.Println("[DRY RUN]")
	fmt.Println("[DRY RUN] Pipeline stages that would execute:")
	fmt.Println("[DRY RUN]   1. DECRYPT - Decrypt backup file with GPG")
	fmt.Println("[DRY RUN]   2. DECOMPRESS - Decompress with gzip")
	fmt.Println("[DRY RUN]   3. EXTRACT - Extract tar archive to destination")

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
