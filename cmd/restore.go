package cmd

import (
	"github.com/icemarkom/secure-backup/internal/backup"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/errors"
	"github.com/spf13/cobra"
)

var (
	restoreFile       string
	restoreDest       string
	restorePrivateKey string
	restorePassphrase string
	restoreVerbose    bool
	restoreDryRun     bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore from an encrypted backup",
	Long: `Restore files from an encrypted backup.

The restore pipeline follows this order (reverse of backup):
  1. DECRYPT - Decrypt the backup file (GPG)
  2. DECOMPRESS - Decompress the decrypted data (gzip)
  3. EXTRACT - Extract the tar archive to destination

The backup format is automatically detected from the file extension.`,
	RunE: runRestore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVar(&restoreFile, "file", "", "Backup file to restore (required)")
	restoreCmd.Flags().StringVar(&restoreDest, "dest", "", "Destination directory for restored files (required)")
	restoreCmd.Flags().StringVar(&restorePrivateKey, "private-key", "", "Path to GPG private key file")
	restoreCmd.Flags().StringVar(&restorePassphrase, "passphrase", "", "GPG key passphrase")
	restoreCmd.Flags().BoolVarP(&restoreVerbose, "verbose", "v", false, "Verbose output")
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "Preview restore without executing")

	restoreCmd.MarkFlagRequired("file")
	restoreCmd.MarkFlagRequired("dest")
}

func runRestore(cmd *cobra.Command, args []string) error {
	// Create compressor (gzip - should match backup)
	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  0,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to initialize compressor",
			"This is an internal error - please report if it persists")
	}

	// Create encryptor for decryption
	encryptCfg := encrypt.Config{
		Method:     "gpg",
		PrivateKey: restorePrivateKey,
		Passphrase: restorePassphrase,
	}

	// If no explicit private key file, try to use system keyring
	if encryptCfg.PrivateKey == "" {
		// For now, we'll require explicit private key path
		// Future: integrate with GPG keyring
		return errors.MissingRequired("--private-key",
			"Export your GPG private key with: gpg --export-secret-keys your@email.com > ~/.gnupg/backup-priv.asc")
	}

	encryptor, err := encrypt.NewEncryptor(encryptCfg)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize decryption",
			"Check that your private key file exists and is a valid GPG key")
	}

	// Execute restore
	restoreCfg := backup.RestoreConfig{
		BackupFile: restoreFile,
		DestPath:   restoreDest,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    restoreVerbose,
		DryRun:     restoreDryRun,
	}

	if err = backup.PerformRestore(restoreCfg); err != nil {
		return err // PerformRestore already returns user-friendly errors
	}

	// Silent by default - verbose output handled in backup package
	return nil
}
