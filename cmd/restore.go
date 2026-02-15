package cmd

import (
	"fmt"
	"strings"

	"github.com/icemarkom/secure-backup/internal/backup"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/errors"
	"github.com/icemarkom/secure-backup/internal/manifest"
	"github.com/icemarkom/secure-backup/internal/passphrase"
	"github.com/spf13/cobra"
)

var (
	restoreFile           string
	restoreDest           string
	restorePrivateKey     string
	restorePassphrase     string
	restorePassphraseFile string
	restoreVerbose        bool
	restoreDryRun         bool
	restoreSkipManifest   bool
	restoreForce          bool
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
	restoreCmd.Flags().StringVar(&restorePassphrase, "passphrase", "", "GPG key passphrase (insecure - use env var or file instead)")
	restoreCmd.Flags().StringVar(&restorePassphraseFile, "passphrase-file", "", "Path to file containing GPG key passphrase")
	restoreCmd.Flags().BoolVarP(&restoreVerbose, "verbose", "v", false, "Verbose output")
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "Preview restore without executing")
	restoreCmd.Flags().BoolVar(&restoreSkipManifest, "skip-manifest", false, "Skip manifest validation (use for old backups without manifests)")
	restoreCmd.Flags().BoolVar(&restoreForce, "force", false, "Allow restore to non-empty directory")

	restoreCmd.MarkFlagRequired("file")
	restoreCmd.MarkFlagRequired("dest")
}

func runRestore(cmd *cobra.Command, args []string) error {
	// Validate manifest first (unless skipped or dry-run)
	if !restoreSkipManifest && !restoreDryRun {
		if err := validateManifest(restoreFile, restoreVerbose); err != nil {
			return err
		}
	}

	// Create compressor (gzip - should match backup)
	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  0,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to initialize compressor",
			"This is an internal error - please report if it persists")
	}

	// Retrieve passphrase using priority order: flag → env → file
	passphraseValue, err := passphrase.Get(
		restorePassphrase,
		"SECURE_BACKUP_PASSPHRASE",
		restorePassphraseFile,
	)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve passphrase",
			"Provide passphrase via one method only: --passphrase (insecure), SECURE_BACKUP_PASSPHRASE env var, or --passphrase-file")
	}

	// Create encryptor for decryption
	encryptCfg := encrypt.Config{
		Method:     "gpg",
		PrivateKey: restorePrivateKey,
		Passphrase: passphraseValue,
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
		Force:      restoreForce,
	}

	if err = backup.PerformRestore(restoreCfg); err != nil {
		return err // PerformRestore already returns user-friendly errors
	}

	// Silent by default - verbose output handled in backup package
	return nil
}

// validateManifest validates the manifest file for the backup
func validateManifest(backupFile string, verbose bool) error {
	manifestPath := strings.TrimSuffix(backupFile, ".tar.gz.gpg") + ".json"

	m, err := manifest.Read(manifestPath)
	if err != nil {
		return errors.New(
			fmt.Sprintf("Manifest not found: %s", manifestPath),
			"Use --skip-manifest to restore without validation (not recommended for old backups)",
		)
	}

	if err := m.Validate(); err != nil {
		return errors.Wrap(err, "Invalid manifest file",
			"The manifest may be corrupted. Use --skip-manifest to bypass (not recommended)")
	}

	if err := m.ValidateChecksum(backupFile); err != nil {
		return errors.New(
			"Backup file checksum mismatch",
			"File may be corrupted. Use --skip-manifest to bypass (not recommended)",
		)
	}

	if verbose {
		fmt.Println("✓ Manifest validation passed")
	}

	return nil
}
