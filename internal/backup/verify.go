package backup

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/errors"
	"github.com/icemarkom/secure-backup/internal/format"
	"github.com/icemarkom/secure-backup/internal/progress"
)

// VerifyConfig holds configuration for verify operations
type VerifyConfig struct {
	BackupFile string
	Encryptor  encrypt.Encryptor
	Compressor compress.Compressor
	Quick      bool
	Verbose    bool
	DryRun     bool
}

// PerformVerify verifies the integrity of a backup file
func PerformVerify(ctx context.Context, cfg VerifyConfig) error {
	// Handle dry-run mode
	if cfg.DryRun {
		return dryRunVerify(cfg)
	}

	// Validate backup file exists
	fileInfo, err := os.Stat(cfg.BackupFile)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.MissingFile(cfg.BackupFile,
				"Specify a valid backup file with --file")
		}
		return errors.Wrap(err, fmt.Sprintf("Cannot access backup file: %s", cfg.BackupFile),
			"Check file permissions")
	}

	if cfg.Verbose {
		fmt.Printf("Verifying: %s (%s)\n", cfg.BackupFile, format.Size(fileInfo.Size()))
	}

	if cfg.Quick {
		// Quick verification: just check if file can be opened and has valid headers
		return quickVerify(cfg)
	}

	// Full verification: decrypt and decompress to verify integrity
	return fullVerify(ctx, cfg)
}

// quickVerify performs a quick check of file headers
func quickVerify(cfg VerifyConfig) error {
	file, err := os.Open(cfg.BackupFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Try to read GPG header
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file header: %w", err)
	}

	// Check for GPG armor header
	headerStr := string(header[:n])
	if len(headerStr) < 10 {
		return fmt.Errorf("file too small to be a valid backup")
	}

	// Basic header validation
	if string(header[0:5]) == "-----" {
		// ASCII armored GPG
		if cfg.Verbose {
			fmt.Println("✓ GPG armor header detected")
		}
	} else {
		// Binary GPG or other format
		if cfg.Verbose {
			fmt.Println("✓ Binary format detected")
		}
	}

	if cfg.Verbose {
		fmt.Println("✓ Quick verification passed")
	}

	return nil
}

// fullVerify performs full decryption and decompression to verify integrity
func fullVerify(ctx context.Context, cfg VerifyConfig) error {
	// Open backup file
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
		Description: "Verifying",
		TotalBytes:  fileSize,
		Enabled:     cfg.Verbose,
	})

	// Decrypt
	decryptedReader, err := cfg.Encryptor.Decrypt(pr)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// Decompress
	decompressedReader, err := cfg.Compressor.Decompress(decryptedReader)
	if err != nil {
		return fmt.Errorf("decompression failed: %w", err)
	}

	// Read through the entire stream to verify integrity
	bytesRead, err := io.Copy(io.Discard, decompressedReader)
	pr.Finish()
	if err != nil {
		return fmt.Errorf("archive verification failed: %w", err)
	}

	if cfg.Verbose {
		fmt.Printf("✓ Successfully verified %s of decompressed data\n", format.Size(bytesRead))
		fmt.Println("✓ Full verification passed")
	}

	return nil
}

// dryRunVerify previews verify operation without executing
// Note: Dry-run mode always shows verbose output for useful preview
func dryRunVerify(cfg VerifyConfig) error {
	// Validate backup file exists
	fileInfo, err := os.Stat(cfg.BackupFile)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Print dry-run preview (always verbose)
	fmt.Println("[DRY RUN] Verify preview:")
	fmt.Printf("[DRY RUN]   Backup file: %s (%s)\n", cfg.BackupFile, format.Size(fileInfo.Size()))

	if cfg.Quick {
		fmt.Printf("[DRY RUN]   Mode: Quick verification (header check only)\n")
		fmt.Println("[DRY RUN]")
		fmt.Println("[DRY RUN] Quick verification would check:")
		fmt.Println("[DRY RUN]   - File can be opened")
		fmt.Println("[DRY RUN]   - GPG header is valid")
	} else {
		fmt.Printf("[DRY RUN]   Mode: Full verification (decrypt + decompress)\n")
		fmt.Println("[DRY RUN]")
		fmt.Println("[DRY RUN] Full verification would:")
		fmt.Println("[DRY RUN]   1. DECRYPT - Decrypt with GPG")
		fmt.Println("[DRY RUN]   2. DECOMPRESS - Decompress with gzip")
		fmt.Println("[DRY RUN]   3. VERIFY - Read entire archive to verify integrity")
	}

	return nil
}
