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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/icemarkom/secure-backup/internal/backup"
	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/icemarkom/secure-backup/internal/errors"
	"github.com/icemarkom/secure-backup/internal/lock"
	"github.com/icemarkom/secure-backup/internal/manifest"
	"github.com/icemarkom/secure-backup/internal/progress"
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
	backupFileMode     string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create an encrypted backup",
	RunE:  runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Long = fmt.Sprintf(`Create an encrypted, compressed backup of a directory or Docker volume.

The backup pipeline follows this order (critical for compression):
  1. TAR - Archive the source directory
  2. COMPRESS - Compress the tar archive (%s)
  3. ENCRYPT - Encrypt the compressed archive (%s)

This order is critical because encrypted data cannot be compressed.

Encryption methods:
  %s (default) - --public-key is a path to a %s public key file (.asc)
  %s           - --public-key is a direct %s recipient string (age1...)`,
		compress.ValidMethodNames(), encrypt.ValidMethodNames(),
		strings.ToUpper(encrypt.MethodGPG), strings.ToUpper(encrypt.MethodGPG),
		strings.ToUpper(encrypt.MethodAGE), strings.ToUpper(encrypt.MethodAGE))

	backupCmd.Flags().StringVar(&backupSource, "source", "", "Source directory to backup (required)")
	backupCmd.Flags().StringVar(&backupDest, "dest", "", "Destination directory for backup file (required)")
	backupCmd.Flags().StringVar(&backupRecipient, "recipient", "", "GPG recipient email or key ID")
	backupCmd.Flags().StringVar(&backupPublicKey, "public-key", "", fmt.Sprintf("Public key: GPG key file path (--encryption %s) or AGE recipient string (--encryption %s)", encrypt.MethodGPG, encrypt.MethodAGE))
	backupCmd.Flags().StringVar(&backupEncryption, "encryption", encrypt.MethodGPG, fmt.Sprintf("Encryption method: %s (default: %s)", encrypt.ValidMethodNames(), encrypt.MethodGPG))
	backupCmd.Flags().IntVar(&backupRetention, "retention", retention.DefaultKeepLast, "Number of backups to keep (0 = keep all)")
	backupCmd.Flags().BoolVarP(&backupVerbose, "verbose", "v", false, "Verbose output")
	backupCmd.Flags().BoolVar(&backupDryRun, "dry-run", false, "Preview backup without executing")
	backupCmd.Flags().BoolVar(&backupSkipManifest, "skip-manifest", false, "Skip manifest generation (not recommended for production)")
	backupCmd.Flags().StringVar(&backupFileMode, "file-mode", "default", `File permissions for backup and manifest files ("default"=0600, "system"=umask, or octal like "0640")`)

	backupCmd.MarkFlagRequired("source")
	backupCmd.MarkFlagRequired("dest")
	backupCmd.MarkFlagRequired("public-key")
}

func runBackup(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	ctx := cmd.Context()

	// Acquire lock to prevent concurrent backups (skip in dry-run — no writes)
	if !backupDryRun {
		lockPath, err := lock.Acquire(backupDest)
		if err != nil {
			return err // Already wrapped with helpful message
		}
		defer lock.Release(lockPath) // Always release on exit
	}

	// Create compressor (gzip by default)
	compressor, err := compress.NewCompressor(compress.Config{
		Method: compress.Gzip,
		Level:  0, // Use default (level 6)
	})
	if err != nil {
		return fmt.Errorf("failed to create compressor: %w", err)
	}

	// Parse encryption method
	encMethod, err := encrypt.ParseMethod(backupEncryption)
	if err != nil {
		return err
	}

	// Create encryptor
	encryptCfg := encrypt.Config{
		Method:    encMethod,
		Recipient: backupRecipient,
		PublicKey: backupPublicKey,
	}

	encryptor, err := encrypt.NewEncryptor(encryptCfg)
	if err != nil {
		var hint string
		switch encMethod {
		case encrypt.GPG:
			hint = "Check that your public key file exists and is a valid GPG key"
		case encrypt.AGE:
			hint = "Check that your --public-key value is a valid age recipient string (starts with age1)"
		default:
			hint = fmt.Sprintf("Unknown encryption method: %s", encMethod)
		}
		return errors.Wrap(err, "Failed to initialize encryption", hint)
	}

	// Parse file mode
	fileMode, err := parseFileMode(backupFileMode)
	if err != nil {
		return err
	}

	// Execute backup
	backupCfg := backup.Config{
		SourcePath: backupSource,
		DestDir:    backupDest,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    backupVerbose,
		DryRun:     backupDryRun,
		FileMode:   fileMode,
	}

	outputPath, err := backup.PerformBackup(ctx, backupCfg)
	if err != nil {
		return err // PerformBackup already returns user-friendly errors
	}

	// Generate manifest by default (unless dry-run or skip-manifest)
	if !backupDryRun && !backupSkipManifest {
		if err := generateManifest(outputPath, backupSource, backupVerbose, fileMode); err != nil {
			// Warn but don't fail the backup
			fmt.Fprintf(os.Stderr, "Warning: Failed to create manifest: %v\n", err)
		}
	}

	// Silent by default - verbose output handled in backup package

	// Apply retention policy if specified
	if backupRetention > 0 {
		// Match file extension to encryption method
		var retentionPattern string
		switch encMethod {
		case encrypt.GPG:
			retentionPattern = "backup_*.tar.gz.gpg"
		case encrypt.AGE:
			retentionPattern = "backup_*.tar.gz.age"
		default:
			return fmt.Errorf("unexpected encryption method for retention: %s", encMethod)
		}

		retentionPolicy := retention.Policy{
			KeepLast:  backupRetention,
			BackupDir: backupDest,
			Pattern:   retentionPattern,
			Verbose:   backupVerbose,
			DryRun:    backupDryRun,
		}

		_, err := retention.ApplyPolicy(retentionPolicy)
		if err != nil {
			// Don't fail the backup if retention cleanup fails
			fmt.Fprintf(os.Stderr, "Warning: retention cleanup failed: %v\n", err)
		}
	}

	return nil
}

// generateManifest creates a manifest file for the backup
func generateManifest(backupPath, sourcePath string, verbose bool, fileMode *os.FileMode) error {
	// Create manifest
	m, err := manifest.New(sourcePath, filepath.Base(backupPath), GetVersion())
	if err != nil {
		return fmt.Errorf("failed to create manifest: %w", err)
	}

	// Compute checksum
	checksum, err := manifest.ComputeChecksumProgress(backupPath, progress.Config{
		Description: "Computing checksum",
		Enabled:     verbose,
	})
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
	if err := m.Write(manifestPath, fileMode); err != nil {
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

// parseFileMode parses the --file-mode flag value.
// Returns nil for "system" (use umask), or a concrete os.FileMode for "default" or an octal string.
func parseFileMode(value string) (*os.FileMode, error) {
	switch value {
	case "system":
		return nil, nil
	case "default":
		mode := os.FileMode(0600)
		return &mode, nil
	default:
		parsed, err := strconv.ParseUint(value, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid --file-mode value %q: must be \"default\", \"system\", or an octal mode like \"0640\"", value)
		}
		mode := os.FileMode(parsed)
		if mode > 0777 {
			return nil, fmt.Errorf("invalid --file-mode value %q: must be a valid permission mode (0000-0777)", value)
		}
		// Warn if world-readable
		if mode&0004 != 0 {
			fmt.Fprintf(os.Stderr, "WARNING: --file-mode %04o makes backup files world-readable. "+
				"Consider using 0600 (owner-only) or 0640 (group-readable) instead.\n", mode)
		}
		return &mode, nil
	}
}
