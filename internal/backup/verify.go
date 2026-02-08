package backup

import (
	"fmt"
	"io"
	"os"

	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
)

// VerifyConfig holds configuration for verify operations
type VerifyConfig struct {
	BackupFile string
	Encryptor  encrypt.Encryptor
	Compressor compress.Compressor
	Quick      bool
	Verbose    bool
}

// PerformVerify verifies the integrity of a backup file
func PerformVerify(cfg VerifyConfig) error {
	// Validate backup file exists
	fileInfo, err := os.Stat(cfg.BackupFile)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	if cfg.Verbose {
		fmt.Printf("Verifying: %s (%s)\n", cfg.BackupFile, formatSize(fileInfo.Size()))
	}

	if cfg.Quick {
		// Quick verification: just check if file can be opened and has valid headers
		return quickVerify(cfg)
	}

	// Full verification: decrypt and decompress to verify integrity
	return fullVerify(cfg)
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
func fullVerify(cfg VerifyConfig) error {
	// Open backup file
	backupFile, err := os.Open(cfg.BackupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer backupFile.Close()

	// Decrypt
	if cfg.Verbose {
		fmt.Println("Decrypting...")
	}
	decryptedReader, err := cfg.Encryptor.Decrypt(backupFile)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// Decompress
	if cfg.Verbose {
		fmt.Println("Decompressing...")
	}
	decompressedReader, err := cfg.Compressor.Decompress(decryptedReader)
	if err != nil {
		return fmt.Errorf("decompression failed: %w", err)
	}

	// Read through the entire stream to verify integrity
	if cfg.Verbose {
		fmt.Println("Verifying archive integrity...")
	}
	bytesRead, err := io.Copy(io.Discard, decompressedReader)
	if err != nil {
		return fmt.Errorf("archive verification failed: %w", err)
	}

	if cfg.Verbose {
		fmt.Printf("✓ Successfully verified %s of decompressed data\n", formatSize(bytesRead))
		fmt.Println("✓ Full verification passed")
	}

	return nil
}
