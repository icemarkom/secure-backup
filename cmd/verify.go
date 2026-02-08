package cmd

import (
	"fmt"

	"github.com/icemarkom/secure-backup/internal/backup"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/spf13/cobra"
)

var (
	verifyFile       string
	verifyPrivateKey string
	verifyPassphrase string
	verifyQuick      bool
	verifyVerbose    bool
	verifyDryRun     bool
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
	verifyCmd.Flags().StringVar(&verifyPassphrase, "passphrase", "", "GPG key passphrase (for full verify)")
	verifyCmd.Flags().BoolVar(&verifyQuick, "quick", false, "Quick verification (headers only)")
	verifyCmd.Flags().BoolVarP(&verifyVerbose, "verbose", "v", false, "Verbose output")
	verifyCmd.Flags().BoolVar(&verifyDryRun, "dry-run", false, "Preview verification without executing")

	verifyCmd.MarkFlagRequired("file")
}

func runVerify(cmd *cobra.Command, args []string) error {
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

	// Full verification requires decryption
	if verifyPrivateKey == "" {
		return fmt.Errorf("--private-key required for full verification (or use --quick)")
	}

	// Create compressor
	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  0,
	})
	if err != nil {
		return fmt.Errorf("failed to create compressor: %w", err)
	}

	// Create encryptor
	encryptCfg := encrypt.Config{
		Method:     "gpg",
		PrivateKey: verifyPrivateKey,
		Passphrase: verifyPassphrase,
	}

	encryptor, err := encrypt.NewEncryptor(encryptCfg)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
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

	if err := backup.PerformVerify(verifyCfg); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Silent by default - verbose output handled in backup package
	return nil
}
