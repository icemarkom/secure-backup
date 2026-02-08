package backup

import (
	"fmt"
	"os"

	"github.com/icemarkom/secure-backup/internal/archive"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
)

// RestoreConfig holds configuration for restore operations
type RestoreConfig struct {
	BackupFile string
	DestPath   string
	Encryptor  encrypt.Encryptor
	Compressor compress.Compressor
	Verbose    bool
	DryRun     bool
}

// PerformRestore executes the restore pipeline: DECRYPT → DECOMPRESS → EXTRACT
func PerformRestore(cfg RestoreConfig) error {
	// Handle dry-run mode
	if cfg.DryRun {
		return dryRunRestore(cfg)
	}

	// Validate backup file exists
	if _, err := os.Stat(cfg.BackupFile); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(cfg.DestPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
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
	fmt.Printf("[DRY RUN]   Backup file: %s (%s)\n", cfg.BackupFile, formatSize(fileInfo.Size()))
	fmt.Printf("[DRY RUN]   Destination: %s\n", cfg.DestPath)
	fmt.Println("[DRY RUN]")
	fmt.Println("[DRY RUN] Pipeline stages that would execute:")
	fmt.Println("[DRY RUN]   1. DECRYPT - Decrypt backup file with GPG")
	fmt.Println("[DRY RUN]   2. DECOMPRESS - Decompress with gzip")
	fmt.Println("[DRY RUN]   3. EXTRACT - Extract tar archive to destination")

	return nil
}
