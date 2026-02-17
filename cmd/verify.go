// Copyright 2026 Marko Milivojevic
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/icemarkom/secure-backup/internal/backup"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/errors"
	"github.com/icemarkom/secure-backup/internal/format"
	"github.com/icemarkom/secure-backup/internal/manifest"
	"github.com/icemarkom/secure-backup/internal/passphrase"
	"github.com/icemarkom/secure-backup/internal/progress"
	"github.com/spf13/cobra"
)

var (
	verifyFile           string
	verifyPrivateKey     string
	verifyPassphrase     string
	verifyPassphraseFile string
	verifyEncryption     string
	verifyQuick          bool
	verifyVerbose        bool
	verifyDryRun         bool
	verifySkipManifest   bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify backup integrity",
	RunE:  runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)

	verifyCmd.Long = fmt.Sprintf(`Verify the integrity of an encrypted backup.

Quick mode (--quick): Only checks file headers without full decryption
Full mode (default): Decrypts and decompresses entire backup to verify integrity

The encryption method is auto-detected from the file extension (.%s or .%s),
or can be explicitly set with --encryption.

Encryption methods:
  %s (default) - --private-key is a path to a %s private key file (.asc)
  %s           - --private-key is a path to an %s identity file`,
		strings.ToUpper(encrypt.MethodGPG), strings.ToUpper(encrypt.MethodAGE),
		strings.ToUpper(encrypt.MethodGPG), strings.ToUpper(encrypt.MethodGPG),
		strings.ToUpper(encrypt.MethodAGE), strings.ToUpper(encrypt.MethodAGE))

	verifyCmd.Flags().StringVar(&verifyFile, "file", "", "Backup file to verify (required)")
	verifyCmd.Flags().StringVar(&verifyPrivateKey, "private-key", "", "Private key: GPG key file path (.asc) or age identity file")
	verifyCmd.Flags().StringVar(&verifyPassphrase, "passphrase", "", "GPG key passphrase (insecure - use env var or file instead)")
	verifyCmd.Flags().StringVar(&verifyPassphraseFile, "passphrase-file", "", "Path to file containing GPG key passphrase")
	verifyCmd.Flags().StringVar(&verifyEncryption, "encryption", "", fmt.Sprintf("Encryption method: %s (auto-detected from file extension if omitted)", encrypt.ValidMethodNames()))
	verifyCmd.Flags().BoolVar(&verifyQuick, "quick", false, "Quick verification (headers only)")
	verifyCmd.Flags().BoolVarP(&verifyVerbose, "verbose", "v", false, "Verbose output")
	verifyCmd.Flags().BoolVar(&verifyDryRun, "dry-run", false, "Preview verification without executing")
	verifyCmd.Flags().BoolVar(&verifySkipManifest, "skip-manifest", false, "Skip manifest validation")

	verifyCmd.MarkFlagRequired("file")
}

func runVerify(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Validate all required flags BEFORE any output.
	// Full verification requires --private-key; check early to avoid
	// printing partial success (manifest/checksum) before an error.
	if !verifyQuick && verifyPrivateKey == "" {
		return errors.MissingRequired("--private-key",
			"Full verification requires --private-key, or use --quick for header-only check")
	}

	// All flag validation passed — suppress usage for runtime errors from here on
	cmd.SilenceUsage = true

	// Validate manifest (produces output — safe because flags are valid)
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

		if err := backup.PerformVerify(ctx, verifyCfg); err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}

		// Silent by default - verbose output handled in backup package
		return nil
	}

	// Auto-detect compression method from backup filename
	compMethod, err := compress.ResolveMethod(verifyFile)
	if err != nil {
		return errors.Wrap(err, "Failed to detect compression method",
			"Check that the backup file has a recognized extension (.tar.gz.gpg, .tar.gpg, etc.)")
	}

	compressor, err := compress.NewCompressor(compress.Config{
		Method: compMethod,
		Level:  0,
	})
	if err != nil {
		return fmt.Errorf("failed to create compressor: %w", err)
	}

	// Detect encryption method from file extension if not specified
	encryptionMethod, err := encrypt.ResolveMethod(verifyEncryption, verifyFile)
	if err != nil {
		return err
	}

	// Retrieve passphrase (GPG only — age keys don't use passphrases)
	var passphraseValue string
	switch encryptionMethod {
	case encrypt.GPG:
		var err error
		passphraseValue, err = passphrase.Get(
			verifyPassphrase,
			"SECURE_BACKUP_PASSPHRASE",
			verifyPassphraseFile,
		)
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve passphrase",
				"Provide passphrase via one method only: --passphrase (insecure), SECURE_BACKUP_PASSPHRASE env var, or --passphrase-file")
		}
	case encrypt.AGE:
		// age keys don't use passphrases
	default:
		return fmt.Errorf("unexpected encryption method: %s", encryptionMethod)
	}

	// Create encryptor
	encryptCfg := encrypt.Config{
		Method:     encryptionMethod,
		PrivateKey: verifyPrivateKey,
		Passphrase: passphraseValue,
	}

	encryptor, err := encrypt.NewEncryptor(encryptCfg)
	if err != nil {
		var hint string
		switch encryptionMethod {
		case encrypt.GPG:
			hint = "Check that your private key file exists and is a valid GPG key"
		case encrypt.AGE:
			hint = "Check that your --private-key file is a valid age identity file"
		default:
			hint = fmt.Sprintf("Unknown encryption method: %s", encryptionMethod)
		}
		return errors.Wrap(err, "Failed to initialize decryption for verification", hint)
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

	if err = backup.PerformVerify(ctx, verifyCfg); err != nil {
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

	if err := m.ValidateChecksumProgress(backupFile, progress.Config{
		Description: "Validating checksum",
		Enabled:     verbose,
	}); err != nil {
		return nil, errors.New(
			"Backup file checksum mismatch",
			"File may be corrupted",
		)
	}

	// Display manifest info (only in verbose mode — silent by default)
	if verbose {
		fmt.Printf("Manifest: ✓ Found\n")
		fmt.Printf("Checksum: ✓ Valid (%s: %s)\n", m.ChecksumAlgorithm, m.ChecksumValue)
		fmt.Printf("Created:  %s by %s %s on %s\n",
			m.CreatedAt.Format("2006-01-02 15:04:05"),
			m.CreatedBy.Tool, m.CreatedBy.Version, m.CreatedBy.Hostname)
		fmt.Printf("Source:   %s\n", m.SourcePath)
		fmt.Printf("Size:     %s\n", format.Size(m.SizeBytes))
	}

	return m, nil
}
