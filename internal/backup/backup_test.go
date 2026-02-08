package backup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    1500,
			expected: "1.5 KiB",
		},
		{
			name:     "megabytes",
			bytes:    2 * 1024 * 1024,
			expected: "2.0 MiB",
		},
		{
			name:     "gigabytes",
			bytes:    3 * 1024 * 1024 * 1024,
			expected: "3.0 GiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDirectorySize(t *testing.T) {
	// Create temp directory with known files
	tempDir := t.TempDir()

	// Create a file with known size
	file1 := filepath.Join(tempDir, "file1.txt")
	err := os.WriteFile(file1, bytes.Repeat([]byte("a"), 1000), 0644)
	require.NoError(t, err)

	// Create subdirectory with another file
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	file2 := filepath.Join(subDir, "file2.txt")
	err = os.WriteFile(file2, bytes.Repeat([]byte("b"), 500), 0644)
	require.NoError(t, err)

	// Get directory size
	size := getDirectorySize(tempDir)

	// Should be 1000 + 500 = 1500 bytes
	assert.Equal(t, int64(1500), size)
}

func TestGetDirectorySize_NonexistentDirectory(t *testing.T) {
	// Should return 0 for nonexistent directory (walks nothing)
	size := getDirectorySize("/nonexistent/directory/path")
	assert.Equal(t, int64(0), size)
}

func TestGetDirectorySize_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	size := getDirectorySize(tempDir)
	assert.Equal(t, int64(0), size)
}

// TestPerformBackup_InvalidSource tests error handling for invalid source paths
func TestPerformBackup_InvalidSource(t *testing.T) {
	tests := []struct {
		name       string
		sourcePath string
		wantErrMsg string
	}{
		{
			name:       "nonexistent directory",
			sourcePath: "/nonexistent/directory/that/does/not/exist",
			wantErrMsg: "invalid source path",
		},
		{
			name:       "empty source path",
			sourcePath: "",
			wantErrMsg: "invalid source path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destDir := t.TempDir()

			// Create minimal valid compressor and encryptor (won't be used due to early error)
			compressor, err := compress.NewCompressor(compress.Config{
				Method: "gzip",
				Level:  6,
			})
			require.NoError(t, err)

			// For encryptor, we need a valid key path, but it won't be used
			// Create a dummy key file
			keyFile := filepath.Join(destDir, "dummy.asc")
			err = os.WriteFile(keyFile, []byte("dummy key"), 0644)
			require.NoError(t, err)

			encryptor, err := encrypt.NewEncryptor(encrypt.Config{
				Method:    "gpg",
				PublicKey: keyFile,
			})
			require.NoError(t, err)

			cfg := Config{
				SourcePath: tt.sourcePath,
				DestDir:    destDir,
				Encryptor:  encryptor,
				Compressor: compressor,
				Verbose:    false,
			}

			_, err = PerformBackup(cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

// TestPerformBackup_DestinationCreation tests that destination directory is created if it doesn't exist
func TestPerformBackup_DestinationCreation(t *testing.T) {
	tempRoot := t.TempDir()

	// Create source directory with a file
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Use a nested destination path that doesn't exist
	destDir := filepath.Join(tempRoot, "backups", "nested", "path")
	// Don't create it - PerformBackup should create it

	// Setup encryption test keys
	testKeysDir := filepath.Join(tempRoot, "test_keys")
	err = os.Mkdir(testKeysDir, 0755)
	require.NoError(t, err)

	// Generate test GPG keys using the script
	keyPaths, err := generateTestKeys(t, testKeysDir)
	if err != nil {
		t.Skip("Skipping test: GPG key generation failed (GPG may not be installed)")
	}

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

	cfg := Config{
		SourcePath: sourceDir,
		DestDir:    destDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	backupPath, err := PerformBackup(cfg)
	require.NoError(t, err)

	// Verify destination directory was created
	_, err = os.Stat(destDir)
	assert.NoError(t, err, "destination directory should be created")

	// Verify backup file was created
	_, err = os.Stat(backupPath)
	assert.NoError(t, err, "backup file should exist")

	// Verify backup file is in the correct directory
	assert.True(t, strings.HasPrefix(backupPath, destDir), "backup should be in destination directory")
}

// TestPerformBackup_FilenameFormat tests that backup filenames follow the expected format
func TestPerformBackup_FilenameFormat(t *testing.T) {
	tempRoot := t.TempDir()

	// Create source directory
	sourceDir := filepath.Join(tempRoot, "my-source-dir")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	destDir := filepath.Join(tempRoot, "backups")

	// Generate test keys
	testKeysDir := filepath.Join(tempRoot, "test_keys")
	err = os.Mkdir(testKeysDir, 0755)
	require.NoError(t, err)

	keyPaths, err := generateTestKeys(t, testKeysDir)
	if err != nil {
		t.Skip("Skipping test: GPG key generation failed")
	}

	tests := []struct {
		name            string
		compressionType string
		expectedExt     string
	}{
		{
			name:            "gzip compression",
			compressionType: "gzip",
			expectedExt:     ".tar.gz.gpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor, err := compress.NewCompressor(compress.Config{
				Method: tt.compressionType,
				Level:  6,
			})
			require.NoError(t, err)

			encryptor, err := encrypt.NewEncryptor(encrypt.Config{
				Method:     "gpg",
				PublicKey:  keyPaths.PublicKey,
				PrivateKey: keyPaths.PrivateKey,
			})
			require.NoError(t, err)

			cfg := Config{
				SourcePath: sourceDir,
				DestDir:    destDir,
				Encryptor:  encryptor,
				Compressor: compressor,
				Verbose:    false,
			}

			backupPath, err := PerformBackup(cfg)
			require.NoError(t, err)

			filename := filepath.Base(backupPath)

			// Verify format: backup_{sourcename}_{timestamp}.tar{ext}.gpg
			assert.True(t, strings.HasPrefix(filename, "backup_my-source-dir_"),
				"filename should start with 'backup_my-source-dir_'")
			assert.True(t, strings.HasSuffix(filename, tt.expectedExt),
				"filename should end with '%s'", tt.expectedExt)

			// Verify timestamp format (YYYYMMDD_HHMMSS)
			// Filename format: backup_my-source-dir_20260207_153045.tar.gz.gpg
			parts := strings.Split(filename, "_")
			assert.GreaterOrEqual(t, len(parts), 4, "filename should have at least 4 parts (backup, source, date, time)")

			// Timestamp is parts[2]_parts[3] (YYYYMMDD_HHMMSS)
			datePart := parts[2]
			timePart := strings.Split(parts[3], ".")[0] // Remove .tar.gz.gpg extension
			timestamp := datePart + "_" + timePart

			assert.Equal(t, 15, len(timestamp), "timestamp should be 15 characters (YYYYMMDD_HHMMSS)")
			assert.Equal(t, 8, len(datePart), "date part should be 8 characters (YYYYMMDD)")
			assert.Equal(t, 6, len(timePart), "time part should be 6 characters (HHMMSS)")
		})
	}
}

// Helper type for test key paths
type TestKeyPaths struct {
	PublicKey  string
	PrivateKey string
}

// generateTestKeys generates GPG test keys using the test_data script
func generateTestKeys(t *testing.T, outputDir string) (*TestKeyPaths, error) {
	t.Helper()

	// Find the test_data directory (go up from internal/backup to project root)
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "test_data", "generate_test_keys.sh")

	// Check if script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, err
	}

	// The script generates keys in test_data/ by default
	// We need to use the same keys for all tests to avoid regeneration
	publicKeyPath := filepath.Join(projectRoot, "test_data", "test-public.asc")
	privateKeyPath := filepath.Join(projectRoot, "test_data", "test-private.asc")

	// Check if keys already exist
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Logf("Test keys not found, they should be generated via TestMain or CI")
		return nil, err
	}

	return &TestKeyPaths{
		PublicKey:  publicKeyPath,
		PrivateKey: privateKeyPath,
	}, nil
}
