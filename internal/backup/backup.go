package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/icemarkom/secure-backup/internal/archive"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
)

// Config holds configuration for backup operations
type Config struct {
	SourcePath string
	DestDir    string
	Encryptor  encrypt.Encryptor
	Compressor compress.Compressor
	Verbose    bool
}

// PerformBackup executes the backup pipeline: TAR → COMPRESS → ENCRYPT
func PerformBackup(cfg Config) (string, error) {
	// Validate source
	_, err := os.Stat(cfg.SourcePath)
	if err != nil {
		return "", fmt.Errorf("invalid source path: %w", err)
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

	if cfg.Verbose {
		fmt.Printf("Starting backup of %s (%s)\n", cfg.SourcePath, formatSize(getDirectorySize(cfg.SourcePath)))
		fmt.Printf("Destination: %s\n", outputPath)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		outFile.Close()
		// Clean up on error
		if err != nil {
			os.Remove(outputPath)
		}
	}()

	// Execute the pipeline: TAR → COMPRESS → ENCRYPT → FILE
	if err = executePipeline(cfg, outFile); err != nil {
		return "", fmt.Errorf("backup pipeline failed: %w", err)
	}

	// Get final file size
	finalInfo, _ := outFile.Stat()

	if cfg.Verbose {
		fmt.Printf("Backup completed successfully: %s\n", outputPath)
		if finalInfo != nil {
			fmt.Printf("Backup size: %s\n", formatSize(finalInfo.Size()))
		}
	}

	return outputPath, nil
}

// executePipeline runs the backup pipeline
func executePipeline(cfg Config, output io.Writer) error {
	// Step 1: Create tar reader pipe
	tarPR, tarPW := io.Pipe()

	// Step 2: Compress pipe
	var compressPR io.Reader
	var compressErr error

	// Step 3: Encrypt pipe
	var encryptPR io.Reader
	var encryptErr error

	// Error channel to catch goroutine errors
	errChan := make(chan error, 3)

	// Goroutine 1: Create TAR archive
	go func() {
		defer tarPW.Close()
		if err := archive.CreateTar(cfg.SourcePath, tarPW); err != nil {
			tarPW.CloseWithError(err)
			errChan <- fmt.Errorf("tar creation failed: %w", err)
			return
		}
		errChan <- nil
	}()

	// Step 2: Compress the tar stream
	compressPR, compressErr = cfg.Compressor.Compress(tarPR)
	if compressErr != nil {
		return fmt.Errorf("failed to create compressor: %w", compressErr)
	}

	// Step 3: Encrypt the compressed stream
	encryptPR, encryptErr = cfg.Encryptor.Encrypt(compressPR)
	if encryptErr != nil {
		return fmt.Errorf("failed to create encryptor: %w", encryptErr)
	}

	// Step 4: Write encrypted stream to output file
	if _, err := io.Copy(output, encryptPR); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	// Wait for tar goroutine to complete
	if err := <-errChan; err != nil {
		return err
	}

	return nil
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
