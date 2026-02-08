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

// TestPerformVerify_InvalidFile tests error handling for invalid files
func TestPerformVerify_InvalidFile(t *testing.T) {
	tests := []struct {
		name       string
		backupFile string
		wantErrMsg string
	}{
		{
			name:       "nonexistent file",
			backupFile: "/nonexistent/backup.tar.gz.gpg",
			wantErrMsg: "backup file not found",
		},
		{
			name:       "empty path",
			backupFile: "",
			wantErrMsg: "backup file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Create minimal compressor and encryptor
			compressor, err := compress.NewCompressor(compress.Config{
				Method: "gzip",
				Level:  6,
			})
			require.NoError(t, err)

			keyFile := filepath.Join(tempDir, "dummy.asc")
			err = os.WriteFile(keyFile, []byte("dummy key"), 0644)
			require.NoError(t, err)

			encryptor, err := encrypt.NewEncryptor(encrypt.Config{
				Method:    "gpg",
				PublicKey: keyFile,
			})
			require.NoError(t, err)

			cfg := VerifyConfig{
				BackupFile: tt.backupFile,
				Encryptor:  encryptor,
				Compressor: compressor,
				Quick:      false,
				Verbose:    false,
			}

			err = PerformVerify(cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

// TestQuickVerify_SmallFile tests quick verification with files too small
func TestQuickVerify_SmallFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file that's too small to be a valid backup
	smallFile := filepath.Join(tempDir, "small.gpg")
	err := os.WriteFile(smallFile, []byte("tiny"), 0644)
	require.NoError(t, err)

	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  6,
	})
	require.NoError(t, err)

	keyFile := filepath.Join(tempDir, "dummy.asc")
	err = os.WriteFile(keyFile, []byte("dummy key"), 0644)
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:    "gpg",
		PublicKey: keyFile,
	})
	require.NoError(t, err)

	cfg := VerifyConfig{
		BackupFile: smallFile,
		Encryptor:  encryptor,
		Compressor: compressor,
		Quick:      true,
		Verbose:    false,
	}

	err = PerformVerify(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file too small")
}

// TestQuickVerify_ValidFile tests quick verification with a valid-looking file
func TestQuickVerify_ValidFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file with GPG armor header (ASCII armored format)
	armoredFile := filepath.Join(tempDir, "armored.gpg")
	header := "-----BEGIN PGP MESSAGE-----\nVersion: GnuPG\n\n"
	header += "somebase64encodeddata here to make it long enough\n"
	err := os.WriteFile(armoredFile, []byte(header), 0644)
	require.NoError(t, err)

	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  6,
	})
	require.NoError(t, err)

	keyFile := filepath.Join(tempDir, "dummy.asc")
	err = os.WriteFile(keyFile, []byte("dummy key"), 0644)
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:    "gpg",
		PublicKey: keyFile,
	})
	require.NoError(t, err)

	cfg := VerifyConfig{
		BackupFile: armoredFile,
		Encryptor:  encryptor,
		Compressor: compressor,
		Quick:      true,
		Verbose:    false,
	}

	// Quick verify should pass (it only checks headers, not actual decryption)
	err = PerformVerify(cfg)
	assert.NoError(t, err, "quick verify should pass for valid-looking GPG file")
}

// TestFullVerify_WithRealBackup tests full verification with an actual backup
func TestFullVerify_WithRealBackup(t *testing.T) {
	tempRoot := t.TempDir()

	// Create source directory
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("verification test content"), 0644)
	require.NoError(t, err)

	backupDir := filepath.Join(tempRoot, "backups")

	// Generate test keys
	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping test: GPG key generation failed")
	}

	// Create a real backup
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

	// Test quick verify
	t.Run("quick verify", func(t *testing.T) {
		verifyCfg := VerifyConfig{
			BackupFile: backupPath,
			Encryptor:  encryptor,
			Compressor: compressor,
			Quick:      true,
			Verbose:    false,
		}

		err = PerformVerify(verifyCfg)
		assert.NoError(t, err, "quick verify should pass")
	})

	// Test full verify
	t.Run("full verify", func(t *testing.T) {
		verifyCfg := VerifyConfig{
			BackupFile: backupPath,
			Encryptor:  encryptor,
			Compressor: compressor,
			Quick:      false,
			Verbose:    false,
		}

		err = PerformVerify(verifyCfg)
		assert.NoError(t, err, "full verify should pass")
	})
}

// TestPerformVerify_DryRun_Quick tests dry-run mode with quick verification
func TestPerformVerify_DryRun_Quick(t *testing.T) {
	tempDir := t.TempDir()

	// Create a dummy backup file
	backupFile := filepath.Join(tempDir, "backup.tar.gz.gpg")
	err := os.WriteFile(backupFile, []byte("dummy backup content for testing"), 0644)
	require.NoError(t, err)

	// Create dummy keys for dry-run (won't be used)
	keyFile := filepath.Join(tempDir, "dummy.asc")
	err = os.WriteFile(keyFile, []byte("dummy key"), 0644)
	require.NoError(t, err)

	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:    "gpg",
		PublicKey: keyFile,
	})
	require.NoError(t, err)

	cfg := VerifyConfig{
		BackupFile: backupFile,
		Encryptor:  encryptor,
		Compressor: compressor,
		Quick:      true,
		Verbose:    false,
		DryRun:     true,
	}

	err = PerformVerify(cfg)
	require.NoError(t, err)

	// In dry-run mode, no actual verification is performed
	// Just validates the file exists and shows what would be checked
}

// TestPerformVerify_DryRun_Full tests dry-run mode with full verification
func TestPerformVerify_DryRun_Full(t *testing.T) {
	tempDir := t.TempDir()

	// Create a dummy backup file
	backupFile := filepath.Join(tempDir, "backup.tar.gz.gpg")
	err := os.WriteFile(backupFile, []byte("dummy backup content for testing"), 0644)
	require.NoError(t, err)

	// Create dummy keys for dry-run (won't be used)
	keyFile := filepath.Join(tempDir, "dummy.asc")
	err = os.WriteFile(keyFile, []byte("dummy key"), 0644)
	require.NoError(t, err)

	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:    "gpg",
		PublicKey: keyFile,
	})
	require.NoError(t, err)

	cfg := VerifyConfig{
		BackupFile: backupFile,
		Encryptor:  encryptor,
		Compressor: compressor,
		Quick:      false,
		Verbose:    false,
		DryRun:     true,
	}

	err = PerformVerify(cfg)
	require.NoError(t, err)

	// In dry-run mode, no actual decryption/decompression is performed
	// Just validates the file exists and shows what would be verified
}

// TestPerformVerify_DryRun_InvalidFile tests dry-run with invalid file
func TestPerformVerify_DryRun_InvalidFile(t *testing.T) {
	tempDir := t.TempDir()

	keyFile := filepath.Join(tempDir, "dummy.asc")
	err := os.WriteFile(keyFile, []byte("dummy key"), 0644)
	require.NoError(t, err)

	compressor, err := compress.NewCompressor(compress.Config{
		Method: "gzip",
		Level:  6,
	})
	require.NoError(t, err)

	encryptor, err := encrypt.NewEncryptor(encrypt.Config{
		Method:    "gpg",
		PublicKey: keyFile,
	})
	require.NoError(t, err)

	cfg := VerifyConfig{
		BackupFile: "/nonexistent/backup.tar.gz.gpg",
		Encryptor:  encryptor,
		Compressor: compressor,
		Quick:      false,
		DryRun:     true,
	}

	err = PerformVerify(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup file not found")
}
