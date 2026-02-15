package cmd

import (
	"fmt"

	"github.com/icemarkom/secure-backup/internal/backup"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/errors"
	"github.com/icemarkom/secure-backup/internal/format"
	"github.com/icemarkom/secure-backup/internal/manifest"
	"github.com/icemarkom/secure-backup/internal/passphrase"
	"github.com/spf13/cobra"
)

var (
	verifyFile           string
	verifyPrivateKey     string
	verifyPassphrase     string
	verifyPassphraseFile string
	verifyQuick          bool
	verifyVerbose        bool
	verifyDryRun         bool
	verifySkipManifest   bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify backup integrity",
	Long: `Verify the integrity of an encrypted backup.

Quick mode (--quick): Only checks file headers without full decryption
Full mode (default): Decrypts and decompresses entire backup to verify integrity`,
	RunE: runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)

	verifyCmd.Flags().StringVar(&verifyFile, "file", "", "Backup file to verify (required)")
	verifyCmd.Flags().StringVar(&verifyPrivateKey, "private-key", "", "Path to GPG private key file (for full verify)")
	verifyCmd.Flags().StringVar(&verifyPassphrase, "passphrase", "", "GPG key passphrase (insecure - use env var or file instead)")
	verifyCmd.Flags().StringVar(&verifyPassphraseFile, "passphrase-file", "", "Path to file containing GPG key passphrase")
	verifyCmd.Flags().BoolVar(&verifyQuick, "quick", false, "Quick verification (headers only)")
	verifyCmd.Flags().BoolVarP(&verifyVerbose, "verbose", "v", false, "Verbose output")
	verifyCmd.Flags().BoolVar(&verifyDryRun, "dry-run", false, "Preview verification without executing")

	verifyCmd.MarkFlagRequired("file")
}

func runVerify(cmd *cobra.Command, args []string) error {
	// Validate manifest first (unless skipped or dry-run)
	if !verifySkipManifest && !verifyDryRun {
		if _, err := validateAndDisplayManifest(verifyFile, verifyVerbose); err != nil {
			return err
		}
	}

	// For quick mode, we don't need encryptor/compressor
	if verifyQuick {
		verifyCfg := backup.VerifyConfig{
			BackupFile: verifyFile,
			Quick:      true,
			Verbose:    verifyVerbose,
			DryRun:     verifyDryRun,
		}

		if err := backup.PerformVerify(verifyCfg); err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}

		// Silent by default - verbose output handled in backup package
		return nil
	}

	// Full verification requires
	if !verifyQuick && verifyPrivateKey == "" {
		return errors.MissingRequired("--private-key",
			"Full verification requires --private-key, or use --quick for header-only check")
	}

	// Create compressor
	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  0,
	})
	if err != nil {
		return fmt.Errorf("failed to create compressor: %w", err)
	}

	// Retrieve passphrase using priority order: flag → env → file
	passphraseValue, err := passphrase.Get(
		verifyPassphrase,
		"SECURE_BACKUP_PASSPHRASE",
		verifyPassphraseFile,
	)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve passphrase",
			"Provide passphrase via one method only: --passphrase (insecure), SECURE_BACKUP_PASSPHRASE env var, or --passphrase-file")
	}

	// Create encryptor
	encryptCfg := encrypt.Config{
		Method:     "gpg",
		PrivateKey: verifyPrivateKey,
		Passphrase: passphraseValue,
	}

	encryptor, err := encrypt.NewEncryptor(encryptCfg)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize decryption for verification",
			"Check that your private key file exists and is a valid GPG key")
	}

	// Execute full verification
	verifyCfg := backup.VerifyConfig{
		BackupFile: verifyFile,
		Encryptor:  encryptor,
		Compressor: compressor,
		Quick:      false,
		Verbose:    verifyVerbose,
		DryRun:     verifyDryRun,
	}

	if err = backup.PerformVerify(verifyCfg); err != nil {
		return err // PerformVerify already returns user-friendly errors
	}

	// Silent by default - verbose output handled in backup package
	return nil
}

// validateAndDisplayManifest validates the manifest and displays metadata
func validateAndDisplayManifest(backupFile string, verbose bool) (*manifest.Manifest, error) {
	manifestPath := manifest.ManifestPath(backupFile)

	m, err := manifest.Read(manifestPath)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Manifest not found: %s", manifestPath),
			"Use --skip-manifest to verify without manifest",
		)
	}

	if err := m.ValidateChecksum(backupFile); err != nil {
		return nil, errors.New(
			"Backup file checksum mismatch",
			"File may be corrupted",
		)
	}

	// Display manifest info
	fmt.Printf("Manifest: ✓ Found\n")
	fmt.Printf("Checksum: ✓ Valid (%s: %s...)\n", m.ChecksumAlgorithm, m.ChecksumValue[:16])
	if verbose {
		fmt.Printf("Created:  %s by %s %s on %s\n",
			m.CreatedAt.Format("2006-01-02 15:04:05"),
			m.CreatedBy.Tool, m.CreatedBy.Version, m.CreatedBy.Hostname)
		fmt.Printf("Source:   %s\n", m.SourcePath)
		fmt.Printf("Size:     %s\n", format.Size(m.SizeBytes))
	}

	return m, nil
}
