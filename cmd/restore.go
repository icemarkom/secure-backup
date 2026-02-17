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
	"github.com/icemarkom/secure-backup/internal/common"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/manifest"
	"github.com/icemarkom/secure-backup/internal/passphrase"
	"github.com/icemarkom/secure-backup/internal/progress"
	"github.com/spf13/cobra"
)

var (
	restoreFile           string
	restoreDest           string
	restorePrivateKey     string
	restorePassphrase     string
	restorePassphraseFile string
	restoreEncryption     string
	restoreVerbose        bool
	restoreDryRun         bool
	restoreSkipManifest   bool
	restoreForce          bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore from an encrypted backup",
	RunE:  runRestore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Long = fmt.Sprintf(`Restore files from an encrypted backup.

The restore pipeline follows this order (reverse of backup):
  1. DECRYPT - Decrypt the backup file (%s)
  2. DECOMPRESS - Decompress the decrypted data (%s)
  3. EXTRACT - Extract the tar archive to destination

The encryption method is auto-detected from the file extension (.%s or .%s),
or can be explicitly set with --encryption.

Encryption methods:
  %s (default) - --private-key is a path to a %s private key file (.asc)
  %s           - --private-key is a path to an %s identity file`,
		encrypt.ValidMethodNames(), compress.ValidMethodNames(),
		strings.ToUpper(encrypt.MethodGPG), strings.ToUpper(encrypt.MethodAGE),
		strings.ToUpper(encrypt.MethodGPG), strings.ToUpper(encrypt.MethodGPG),
		strings.ToUpper(encrypt.MethodAGE), strings.ToUpper(encrypt.MethodAGE))

	restoreCmd.Flags().StringVar(&restoreFile, "file", "", "Backup file to restore (required)")
	restoreCmd.Flags().StringVar(&restoreDest, "dest", "", "Destination directory for restored files (required)")
	restoreCmd.Flags().StringVar(&restorePrivateKey, "private-key", "", "Private key: GPG key file path (.asc) or age identity file")
	restoreCmd.Flags().StringVar(&restorePassphrase, "passphrase", "", "GPG key passphrase (insecure - use env var or file instead)")
	restoreCmd.Flags().StringVar(&restorePassphraseFile, "passphrase-file", "", "Path to file containing GPG key passphrase")
	restoreCmd.Flags().StringVar(&restoreEncryption, "encryption", "", fmt.Sprintf("Encryption method: %s (auto-detected from file extension if omitted)", encrypt.ValidMethodNames()))
	restoreCmd.Flags().BoolVarP(&restoreVerbose, "verbose", "v", false, "Verbose output")
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "Preview restore without executing")
	restoreCmd.Flags().BoolVar(&restoreSkipManifest, "skip-manifest", false, "Skip manifest validation (use for old backups without manifests)")
	restoreCmd.Flags().BoolVar(&restoreForce, "force", false, "Allow restore to non-empty directory")

	restoreCmd.MarkFlagRequired("file")
	restoreCmd.MarkFlagRequired("dest")
	restoreCmd.MarkFlagRequired("private-key")
}

func runRestore(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	ctx := cmd.Context()
	// Validate manifest first (unless skipped or dry-run)
	if !restoreSkipManifest && !restoreDryRun {
		if err := validateManifest(restoreFile, restoreVerbose); err != nil {
			return err
		}
	}

	// Auto-detect compression method from backup filename
	compMethod, err := compress.ResolveMethod(restoreFile)
	if err != nil {
		return common.Wrap(err, "Failed to detect compression method",
			"Check that the backup file has a recognized extension (.tar.gz.gpg, .tar.gpg, etc.)")
	}

	compressor, err := compress.NewCompressor(compress.Config{
		Method: compMethod,
		Level:  0,
	})
	if err != nil {
		return common.Wrap(err, "Failed to initialize compressor",
			"This is an internal error - please report if it persists")
	}

	// Detect encryption method from file extension if not specified
	encryptionMethod, err := encrypt.ResolveMethod(restoreEncryption, restoreFile)
	if err != nil {
		return err
	}

	// Retrieve passphrase (GPG only — age keys don't use passphrases)
	var passphraseValue string
	switch encryptionMethod {
	case encrypt.GPG:
		var err error
		passphraseValue, err = passphrase.Get(
			restorePassphrase,
			"SECURE_BACKUP_PASSPHRASE",
			restorePassphraseFile,
		)
		if err != nil {
			return common.Wrap(err, "Failed to retrieve passphrase",
				"Provide passphrase via one method only: --passphrase (insecure), SECURE_BACKUP_PASSPHRASE env var, or --passphrase-file")
		}
	case encrypt.AGE:
		// age keys don't use passphrases
	default:
		return fmt.Errorf("unexpected encryption method: %s", encryptionMethod)
	}

	// Create encryptor for decryption
	encryptCfg := encrypt.Config{
		Method:     encryptionMethod,
		PrivateKey: restorePrivateKey,
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
		return common.Wrap(err, "Failed to initialize decryption", hint)
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

	if err = backup.PerformRestore(ctx, restoreCfg); err != nil {
		return err // PerformRestore already returns user-friendly errors
	}

	// Silent by default - verbose output handled in backup package
	return nil
}

// validateManifest validates the manifest file for the backup
func validateManifest(backupFile string, verbose bool) error {
	manifestPath := manifest.ManifestPath(backupFile)

	m, err := manifest.Read(manifestPath)
	if err != nil {
		return common.New(
			fmt.Sprintf("Manifest not found: %s", manifestPath),
			"Use --skip-manifest to restore without validation (not recommended for old backups)",
		)
	}

	if err := m.Validate(); err != nil {
		return common.Wrap(err, "Invalid manifest file",
			"The manifest may be corrupted. Use --skip-manifest to bypass (not recommended)")
	}

	if err := m.ValidateChecksumProgress(backupFile, progress.Config{
		Description: "Validating checksum",
		Enabled:     verbose,
	}); err != nil {
		return common.New(
			"Backup file checksum mismatch",
			"File may be corrupted. Use --skip-manifest to bypass (not recommended)",
		)
	}

	if verbose {
		fmt.Println("✓ Manifest validation passed")
	}

	return nil
}
