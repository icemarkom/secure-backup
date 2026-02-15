package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/icemarkom/secure-backup/internal/backup"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/errors"
	"github.com/icemarkom/secure-backup/internal/lock"
	"github.com/icemarkom/secure-backup/internal/manifest"
	"github.com/icemarkom/secure-backup/internal/retention"
	"github.com/spf13/cobra"
)

var (
	backupSource       string
	backupDest         string
	backupRecipient    string
	backupPublicKey    string
	backupVerbose      bool
	backupDryRun       bool
	backupEncryption   string
	backupRetention    int
	backupSkipManifest bool
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create an encrypted backup",
	Long: `Create an encrypted, compressed backup of a directory or Docker volume.

The backup pipeline follows this order (critical for compression):
  1. TAR - Archive the source directory
  2. COMPRESS - Compress the tar archive (gzip by default)
  3. ENCRYPT - Encrypt the compressed archive (GPG)

This order is critical because encrypted data cannot be compressed.`,
	RunE: runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVar(&backupSource, "source", "", "Source directory to backup (required)")
	backupCmd.Flags().StringVar(&backupDest, "dest", "", "Destination directory for backup file (required)")
	backupCmd.Flags().StringVar(&backupRecipient, "recipient", "", "GPG recipient email or key ID")
	backupCmd.Flags().StringVar(&backupPublicKey, "public-key", "", "Path to GPG public key file")
	backupCmd.Flags().StringVar(&backupEncryption, "encryption", "gpg", "Encryption method (gpg, age)")
	backupCmd.Flags().IntVar(&backupRetention, "retention", 0, "Retention period in days (0 = keep all backups)")
	backupCmd.Flags().BoolVarP(&backupVerbose, "verbose", "v", false, "Verbose output")
	backupCmd.Flags().BoolVar(&backupDryRun, "dry-run", false, "Preview backup without executing")
	backupCmd.Flags().BoolVar(&backupSkipManifest, "skip-manifest", false, "Skip manifest generation (not recommended for production)")

	backupCmd.MarkFlagRequired("source")
	backupCmd.MarkFlagRequired("dest")
}

func runBackup(cmd *cobra.Command, args []string) error {
	// Validate flags
	if backupRecipient == "" && backupPublicKey == "" {
		return errors.MissingRequired("--public-key",
			"Specify your GPG public key with --public-key ~/.gnupg/backup-pub.asc")
	}

	// Acquire lock to prevent concurrent backups to same destination
	lockPath, err := lock.Acquire(backupDest)
	if err != nil {
		return err // Already wrapped with helpful message
	}
	defer lock.Release(lockPath) // Always release on exit

	// Create compressor (gzip by default)
	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  0, // Use default (level 6)
	})
	if err != nil {
		return fmt.Errorf("failed to create compressor: %w", err)
	}

	// Create encryptor
	encryptCfg := encrypt.Config{
		Method:    backupEncryption,
		Recipient: backupRecipient,
		PublicKey: backupPublicKey,
	}

	// If no explicit public key file, try to use system keyring with recipient
	if encryptCfg.PublicKey == "" && encryptCfg.Recipient != "" {
		// For now, we'll require explicit public key path
		// Future: integrate with GPG keyring
		return errors.MissingRequired("--public-key",
			"Export your GPG public key with: gpg --export your@email.com > ~/.gnupg/backup-pub.asc")
	}

	encryptor, err := encrypt.NewEncryptor(encryptCfg)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize encryption",
			"Check that your public key file exists and is a valid GPG key")
	}

	// Execute backup
	backupCfg := backup.Config{
		SourcePath: backupSource,
		DestDir:    backupDest,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    backupVerbose,
		DryRun:     backupDryRun,
	}

	outputPath, err := backup.PerformBackup(backupCfg)
	if err != nil {
		return err // PerformBackup already returns user-friendly errors
	}

	// Generate manifest by default (unless dry-run or skip-manifest)
	if !backupDryRun && !backupSkipManifest {
		if err := generateManifest(outputPath, backupSource, backupVerbose); err != nil {
			// Warn but don't fail the backup
			fmt.Fprintf(os.Stderr, "Warning: Failed to create manifest: %v\n", err)
		}
	}

	// Silent by default - verbose output handled in backup package

	// Apply retention policy if specified
	if backupRetention > 0 {
		retentionPolicy := retention.Policy{
			RetentionDays: backupRetention,
			BackupDir:     backupDest,
			Pattern:       "backup_*.tar.gz.gpg",
			Verbose:       backupVerbose,
			DryRun:        backupDryRun,
		}

		deletedCount, err := retention.ApplyPolicy(retentionPolicy)
		if err != nil {
			// Don't fail the backup if retention cleanup fails
			fmt.Fprintf(os.Stderr, "Warning: retention cleanup failed: %v\n", err)
		} else if deletedCount > 0 && !backupVerbose {
			fmt.Printf("Deleted %d old backup(s) (retention: %d days)\n", deletedCount, backupRetention)
		}
	}

	return nil
}

// generateManifest creates a manifest file for the backup
func generateManifest(backupPath, sourcePath string, verbose bool) error {
	// Create manifest
	m, err := manifest.New(sourcePath, filepath.Base(backupPath), GetVersion())
	if err != nil {
		return fmt.Errorf("failed to create manifest: %w", err)
	}

	// Compute checksum
	checksum, err := manifest.ComputeChecksum(backupPath)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	m.ChecksumValue = checksum

	// Get file size
	info, err := os.Stat(backupPath)
	if err == nil {
		m.SizeBytes = info.Size()
	}

	// Write manifest
	manifestPath := getManifestPath(backupPath)
	if err := m.Write(manifestPath); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	if verbose {
		fmt.Printf("Manifest created: %s\n", manifestPath)
	}

	return nil
}

// getManifestPath returns the manifest path for a given backup file
func getManifestPath(backupPath string) string {
	return manifest.ManifestPath(backupPath)
}
