package backup

import (
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
			wantErrMsg: "backup file not found",
		},
		{
			name:       "empty backup path",
			backupFile: "",
			wantErrMsg: "backup file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destPath := t.TempDir()

			// Create minimal valid compressor and encryptor (won't be used due to early error)
			compressor, err := compress.NewCompressor(compress.Config{
				Method: "gzip",
				Level:  6,
			})
			require.NoError(t, err)

			// Create dummy key file
			keyFile := filepath.Join(destPath, "dummy.asc")
			err = os.WriteFile(keyFile, []byte("dummy key"), 0644)
			require.NoError(t, err)

			encryptor, err := encrypt.NewEncryptor(encrypt.Config{
				Method:    "gpg",
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

			err = PerformRestore(cfg)
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
		Method: "gzip",
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:     "gpg",
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

	backupPath, err := PerformBackup(backupCfg)
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

	err = PerformRestore(restoreCfg)
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
