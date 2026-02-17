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

package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPerformRestore_InvalidBackupFile tests error handling for invalid backup files
func TestPerformRestore_InvalidBackupFile(t *testing.T) {
	tests := []struct {
		name       string
		backupFile string
		wantErrMsg string
	}{
		{
			name:       "nonexistent backup file",
			backupFile: "/nonexistent/backup/file.tar.gz.gpg",
			wantErrMsg: "File not found",
		},
		{
			name:       "empty backup path",
			backupFile: "",
			wantErrMsg: "File not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destPath := t.TempDir()

			// Create minimal valid compressor and encryptor (won't be used due to early error)
			compressor, err := compress.NewCompressor(compress.Config{
				Method: compress.Gzip,
				Level:  6,
			})
			require.NoError(t, err)

			// Create dummy key file
			keyFile := filepath.Join(destPath, "dummy.asc")
			err = os.WriteFile(keyFile, []byte("dummy key"), 0644)
			require.NoError(t, err)

			encryptor, err := encrypt.NewEncryptor(encrypt.Config{
				Method:    encrypt.GPG,
				PublicKey: keyFile,
			})
			require.NoError(t, err)

			cfg := RestoreConfig{
				BackupFile: tt.backupFile,
				DestPath:   destPath,
				Encryptor:  encryptor,
				Compressor: compressor,
				Verbose:    false,
			}

			err = PerformRestore(context.Background(), cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

// TestPerformRestore_DestinationCreation tests that destination directory is created if it doesn't exist
func TestPerformRestore_DestinationCreation(t *testing.T) {
	tempRoot := t.TempDir()

	// First create a real backup file
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	backupDir := filepath.Join(tempRoot, "backups")

	// Generate test keys
	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping test: GPG key generation failed")
	}

	// Create a backup
	compressor, err := compress.NewCompressor(compress.Config{
		Method: compress.Gzip,
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:     encrypt.GPG,
		PublicKey:  keyPaths.PublicKey,
		PrivateKey: keyPaths.PrivateKey,
	})
	require.NoError(t, err)

	backupCfg := Config{
		SourcePath: sourceDir,
		DestDir:    backupDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	backupPath, _, err := PerformBackup(context.Background(), backupCfg)
	require.NoError(t, err)

	// Now restore to a nested destination path that doesn't exist
	restoreDir := filepath.Join(tempRoot, "restore", "nested", "path")
	// Don't create it - PerformRestore should create it

	restoreCfg := RestoreConfig{
		BackupFile: backupPath,
		DestPath:   restoreDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	err = PerformRestore(context.Background(), restoreCfg)
	require.NoError(t, err)

	// Verify destination directory was created
	_, err = os.Stat(restoreDir)
	assert.NoError(t, err, "destination directory should be created")

	// Verify files were restored
	restoredFile := filepath.Join(restoreDir, "source", "test.txt")
	content, err := os.ReadFile(restoredFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

// TestPerformRestore_DryRun tests that dry-run mode doesn't extract files
func TestPerformRestore_DryRun(t *testing.T) {
	tempRoot := t.TempDir()

	// Create a dummy backup file (won't be read in dry-run)
	backupFile := filepath.Join(tempRoot, "backup.tar.gz.gpg")
	err := os.WriteFile(backupFile, []byte("dummy backup content"), 0644)
	require.NoError(t, err)

	restoreDir := filepath.Join(tempRoot, "restore")

	// Create dummy keys for dry-run (won't be used)
	keyFile := filepath.Join(tempRoot, "dummy.asc")
	err = os.WriteFile(keyFile, []byte("dummy key"), 0644)
	require.NoError(t, err)

	compressor, err := compress.NewCompressor(compress.Config{
		Method: compress.Gzip,
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:    encrypt.GPG,
		PublicKey: keyFile,
	})
	require.NoError(t, err)

	cfg := RestoreConfig{
		BackupFile: backupFile,
		DestPath:   restoreDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
		DryRun:     true,
	}

	err = PerformRestore(context.Background(), cfg)
	require.NoError(t, err)

	// Verify no files were extracted
	_, err = os.Stat(restoreDir)
	assert.True(t, os.IsNotExist(err), "restore directory should NOT be created in dry-run mode")
}

// TestPerformRestore_DryRun_InvalidFile tests dry-run with invalid backup file
func TestPerformRestore_DryRun_InvalidFile(t *testing.T) {
	tempRoot := t.TempDir()
	restoreDir := filepath.Join(tempRoot, "restore")

	keyFile := filepath.Join(tempRoot, "dummy.asc")
	err := os.WriteFile(keyFile, []byte("dummy key"), 0644)
	require.NoError(t, err)

	compressor, err := compress.NewCompressor(compress.Config{
		Method: compress.Gzip,
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:    encrypt.GPG,
		PublicKey: keyFile,
	})
	require.NoError(t, err)

	cfg := RestoreConfig{
		BackupFile: "/nonexistent/backup.tar.gz.gpg",
		DestPath:   restoreDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		DryRun:     true,
	}

	err = PerformRestore(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup file not found")
}

// TestIsDirectoryNonEmpty tests the isDirectoryNonEmpty helper function
func TestIsDirectoryNonEmpty(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(t *testing.T, tempDir string) string
		wantNonEmpty    bool
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "nonexistent directory",
			setup: func(t *testing.T, tempDir string) string {
				return filepath.Join(tempDir, "nonexistent")
			},
			wantNonEmpty: false,
			wantErr:      false,
		},
		{
			name: "empty directory",
			setup: func(t *testing.T, tempDir string) string {
				emptyDir := filepath.Join(tempDir, "empty")
				err := os.Mkdir(emptyDir, 0755)
				require.NoError(t, err)
				return emptyDir
			},
			wantNonEmpty: false,
			wantErr:      false,
		},
		{
			name: "directory with files",
			setup: func(t *testing.T, tempDir string) string {
				dir := filepath.Join(tempDir, "with-files")
				err := os.Mkdir(dir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
				require.NoError(t, err)
				return dir
			},
			wantNonEmpty: true,
			wantErr:      false,
		},
		{
			name: "directory with subdirectories",
			setup: func(t *testing.T, tempDir string) string {
				dir := filepath.Join(tempDir, "with-subdirs")
				err := os.Mkdir(dir, 0755)
				require.NoError(t, err)
				err = os.Mkdir(filepath.Join(dir, "subdir"), 0755)
				require.NoError(t, err)
				return dir
			},
			wantNonEmpty: true,
			wantErr:      false,
		},
		{
			name: "path is a file not directory",
			setup: func(t *testing.T, tempDir string) string {
				filePath := filepath.Join(tempDir, "file.txt")
				err := os.WriteFile(filePath, []byte("content"), 0644)
				require.NoError(t, err)
				return filePath
			},
			wantNonEmpty:    false,
			wantErr:         true,
			wantErrContains: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			path := tt.setup(t, tempDir)

			gotNonEmpty, err := isDirectoryNonEmpty(path)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					assert.Contains(t, err.Error(), tt.wantErrContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantNonEmpty, gotNonEmpty)
			}
		})
	}
}

// TestPerformRestore_NonEmptyDestination_WithoutForce tests that restore fails for non-empty directory without --force
func TestPerformRestore_NonEmptyDestination_WithoutForce(t *testing.T) {
	tempRoot := t.TempDir()

	// Create a backup
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	backupDir := filepath.Join(tempRoot, "backups")

	// Generate test keys
	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping test: GPG key generation failed")
	}

	// Create a backup
	compressor, err := compress.NewCompressor(compress.Config{
		Method: compress.Gzip,
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:     encrypt.GPG,
		PublicKey:  keyPaths.PublicKey,
		PrivateKey: keyPaths.PrivateKey,
	})
	require.NoError(t, err)

	backupCfg := Config{
		SourcePath: sourceDir,
		DestDir:    backupDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	backupPath, _, err := PerformBackup(context.Background(), backupCfg)
	require.NoError(t, err)

	// Create non-empty restore destination
	restoreDir := filepath.Join(tempRoot, "restore")
	err = os.Mkdir(restoreDir, 0755)
	require.NoError(t, err)

	existingFile := filepath.Join(restoreDir, "existing.txt")
	err = os.WriteFile(existingFile, []byte("existing content"), 0644)
	require.NoError(t, err)

	// Try to restore without --force (should fail)
	restoreCfg := RestoreConfig{
		BackupFile: backupPath,
		DestPath:   restoreDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
		Force:      false,
	}

	err = PerformRestore(context.Background(), restoreCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Destination directory is not empty")
	assert.Contains(t, err.Error(), "--force")
}

// TestPerformRestore_NonEmptyDestination_WithForce tests that restore succeeds for non-empty directory with --force
func TestPerformRestore_NonEmptyDestination_WithForce(t *testing.T) {
	tempRoot := t.TempDir()

	// Create a backup
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	backupDir := filepath.Join(tempRoot, "backups")

	// Generate test keys
	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping test: GPG key generation failed")
	}

	// Create a backup
	compressor, err := compress.NewCompressor(compress.Config{
		Method: compress.Gzip,
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:     encrypt.GPG,
		PublicKey:  keyPaths.PublicKey,
		PrivateKey: keyPaths.PrivateKey,
	})
	require.NoError(t, err)

	backupCfg := Config{
		SourcePath: sourceDir,
		DestDir:    backupDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	backupPath, _, err := PerformBackup(context.Background(), backupCfg)
	require.NoError(t, err)

	// Create non-empty restore destination
	restoreDir := filepath.Join(tempRoot, "restore")
	err = os.Mkdir(restoreDir, 0755)
	require.NoError(t, err)

	existingFile := filepath.Join(restoreDir, "existing.txt")
	err = os.WriteFile(existingFile, []byte("existing content"), 0644)
	require.NoError(t, err)

	// Restore with --force (should succeed)
	restoreCfg := RestoreConfig{
		BackupFile: backupPath,
		DestPath:   restoreDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
		Force:      true,
	}

	err = PerformRestore(context.Background(), restoreCfg)
	require.NoError(t, err)

	// Verify both files exist
	restoredFile := filepath.Join(restoreDir, "source", "test.txt")
	content, err := os.ReadFile(restoredFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	// Existing file should still be there
	existingContent, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Equal(t, "existing content", string(existingContent))
}

// TestPerformRestore_EmptyDestination_WithoutForce tests that restore succeeds for empty directory without --force
func TestPerformRestore_EmptyDestination_WithoutForce(t *testing.T) {
	tempRoot := t.TempDir()

	// Create a backup
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	backupDir := filepath.Join(tempRoot, "backups")

	// Generate test keys
	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping test: GPG key generation failed")
	}

	// Create a backup
	compressor, err := compress.NewCompressor(compress.Config{
		Method: compress.Gzip,
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:     encrypt.GPG,
		PublicKey:  keyPaths.PublicKey,
		PrivateKey: keyPaths.PrivateKey,
	})
	require.NoError(t, err)

	backupCfg := Config{
		SourcePath: sourceDir,
		DestDir:    backupDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	backupPath, _, err := PerformBackup(context.Background(), backupCfg)
	require.NoError(t, err)

	// Create empty restore destination
	restoreDir := filepath.Join(tempRoot, "restore")
	err = os.Mkdir(restoreDir, 0755)
	require.NoError(t, err)

	// Restore without --force (should succeed because directory is empty)
	restoreCfg := RestoreConfig{
		BackupFile: backupPath,
		DestPath:   restoreDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
		Force:      false,
	}

	err = PerformRestore(context.Background(), restoreCfg)
	require.NoError(t, err)

	// Verify files were restored
	restoredFile := filepath.Join(restoreDir, "source", "test.txt")
	content, err := os.ReadFile(restoredFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

// TestPerformRestore_NonexistentDestination_WithoutForce tests that restore succeeds for non-existent directory without --force
func TestPerformRestore_NonexistentDestination_WithoutForce(t *testing.T) {
	tempRoot := t.TempDir()

	// Create a backup
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	backupDir := filepath.Join(tempRoot, "backups")

	// Generate test keys
	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping test: GPG key generation failed")
	}

	// Create a backup
	compressor, err := compress.NewCompressor(compress.Config{
		Method: compress.Gzip,
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:     encrypt.GPG,
		PublicKey:  keyPaths.PublicKey,
		PrivateKey: keyPaths.PrivateKey,
	})
	require.NoError(t, err)

	backupCfg := Config{
		SourcePath: sourceDir,
		DestDir:    backupDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	backupPath, _, err := PerformBackup(context.Background(), backupCfg)
	require.NoError(t, err)

	// Don't create restore directory - it should be created automatically
	restoreDir := filepath.Join(tempRoot, "restore")

	// Restore without --force (should succeed because directory doesn't exist)
	restoreCfg := RestoreConfig{
		BackupFile: backupPath,
		DestPath:   restoreDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
		Force:      false,
	}

	err = PerformRestore(context.Background(), restoreCfg)
	require.NoError(t, err)

	// Verify files were restored
	restoredFile := filepath.Join(restoreDir, "source", "test.txt")
	content, err := os.ReadFile(restoredFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}
