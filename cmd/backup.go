package cmd

import (
	"fmt"
	"os"

	"github.com/icemarkom/secure-backup/internal/backup"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/retention"
	"github.com/spf13/cobra"
)

var (
	backupSource     string
	backupDest       string
	backupRecipient  string
	backupPublicKey  string
	backupVerbose    bool
	backupEncryption string
	backupRetention  int
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

	backupCmd.MarkFlagRequired("source")
	backupCmd.MarkFlagRequired("dest")
}

func runBackup(cmd *cobra.Command, args []string) error {
	// Validate flags
	if backupRecipient == "" && backupPublicKey == "" {
		return fmt.Errorf("either --recipient or --public-key must be specified")
	}

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
		return fmt.Errorf("--public-key is required (GPG keyring integration coming soon)")
	}

	encryptor, err := encrypt.NewEncryptor(encryptCfg)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	// Execute backup
	backupCfg := backup.Config{
		SourcePath: backupSource,
		DestDir:    backupDest,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    backupVerbose,
	}

	_, err = backup.PerformBackup(backupCfg)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// Silent by default - verbose output handled in backup package

	// Apply retention policy if specified
	if backupRetention > 0 {
		retentionPolicy := retention.Policy{
			RetentionDays: backupRetention,
			BackupDir:     backupDest,
			Pattern:       "backup_*.tar.gz.gpg",
			Verbose:       backupVerbose,
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
