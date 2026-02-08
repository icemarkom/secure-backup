package backup

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/icemarkom/secure-backup/internal/encrypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_BackupRestoreCycle tests the complete backup → restore cycle
func TestIntegration_BackupRestoreCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempRoot := t.TempDir()

	// Create source directory with test files
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	// Create nested directory structure
	nestedDir := filepath.Join(sourceDir, "subdir")
	err = os.Mkdir(nestedDir, 0755)
	require.NoError(t, err)

	// Create test files with various content
	testFiles := map[string]string{
		"file1.txt":        "Hello, World!",
		"file2.txt":        "This is a test file with more content.",
		"subdir/file3.txt": "Nested file content",
		"subdir/file4.txt": string(bytes.Repeat([]byte("Large content "), 1000)), // ~14KB
	}

	for relPath, content := range testFiles {
		fullPath := filepath.Join(sourceDir, relPath)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	backupDir := filepath.Join(tempRoot, "backups")
	restoreDir := filepath.Join(tempRoot, "restore")

	// Generate test GPG keys
	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping integration test: GPG key generation failed")
	}

	// Create compressor and encryptor
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

	// Step 1: Perform backup
	backupCfg := Config{
		SourcePath: sourceDir,
		DestDir:    backupDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	backupPath, err := PerformBackup(backupCfg)
	require.NoError(t, err)
	assert.FileExists(t, backupPath, "backup file should exist")

	// Verify backup file is not empty
	info, err := os.Stat(backupPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "backup file should not be empty")

	// Step 2: Perform restore
	restoreCfg := RestoreConfig{
		BackupFile: backupPath,
		DestPath:   restoreDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	err = PerformRestore(restoreCfg)
	require.NoError(t, err)

	// Step 3: Verify restored files match source
	for relPath, expectedContent := range testFiles {
		// Restored files are in restoreDir/source/... because tar preserves structure
		restoredPath := filepath.Join(restoreDir, "source", relPath)
		content, err := os.ReadFile(restoredPath)
		require.NoError(t, err, "restored file %s should exist", relPath)
		assert.Equal(t, expectedContent, string(content), "content should match for %s", relPath)
	}
}

// TestIntegration_BackupVerifyCycle tests backup → quick verify → full verify
func TestIntegration_BackupVerifyCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempRoot := t.TempDir()

	// Create simple source directory
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("Integration test content"), 0644)
	require.NoError(t, err)

	backupDir := filepath.Join(tempRoot, "backups")

	// Generate test keys
	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping integration test: GPG key generation failed")
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

	// Create backup
	backupCfg := Config{
		SourcePath: sourceDir,
		DestDir:    backupDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	backupPath, err := PerformBackup(backupCfg)
	require.NoError(t, err)

	// Test quick verification
	t.Run("quick verify", func(t *testing.T) {
		verifyCfg := VerifyConfig{
			BackupFile: backupPath,
			Encryptor:  encryptor,
			Compressor: compressor,
			Quick:      true,
			Verbose:    false,
		}

		err := PerformVerify(verifyCfg)
		assert.NoError(t, err, "quick verification should pass")
	})

	// Test full verification
	t.Run("full verify", func(t *testing.T) {
		verifyCfg := VerifyConfig{
			BackupFile: backupPath,
			Encryptor:  encryptor,
			Compressor: compressor,
			Quick:      false,
			Verbose:    false,
		}

		err := PerformVerify(verifyCfg)
		assert.NoError(t, err, "full verification should pass")
	})
}

// TestIntegration_VerboseMode tests verbose output in backup and restore
func TestIntegration_VerboseMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempRoot := t.TempDir()

	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("verbose test"), 0644)
	require.NoError(t, err)

	backupDir := filepath.Join(tempRoot, "backups")

	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping integration test: GPG key generation failed")
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

	// Test verbose backup
	t.Run("verbose backup", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		backupCfg := Config{
			SourcePath: sourceDir,
			DestDir:    backupDir,
			Encryptor:  encryptor,
			Compressor: compressor,
			Verbose:    true, // Enable verbose mode
		}

		backupPath, err := PerformBackup(backupCfg)
		require.NoError(t, err)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Verify verbose output contains expected messages
		assert.Contains(t, output, "Starting backup", "verbose should show starting message")
		assert.Contains(t, output, "Backup completed", "verbose should show completion message")

		// Store backup path for restore test
		// Cleanup for restore test
		os.Stdout = oldStdout

		// Test verbose restore
		t.Run("verbose restore", func(t *testing.T) {
			restoreDir := filepath.Join(tempRoot, "restore")

			// Capture stdout again
			r, w, _ := os.Pipe()
			oldStdout := os.Stdout
			os.Stdout = w

			restoreCfg := RestoreConfig{
				BackupFile: backupPath,
				DestPath:   restoreDir,
				Encryptor:  encryptor,
				Compressor: compressor,
				Verbose:    true,
			}

			err := PerformRestore(restoreCfg)
			require.NoError(t, err)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Verify verbose restore output
			assert.Contains(t, output, "Restoring from", "verbose should show restoring message")
			assert.Contains(t, output, "Restore completed", "verbose should show completion")
		})
	})
}

// TestIntegration_CorruptedBackup tests error handling with a corrupted backup
func TestIntegration_CorruptedBackup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempRoot := t.TempDir()

	// Create a valid backup first
	sourceDir := filepath.Join(tempRoot, "source")
	err := os.Mkdir(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	backupDir := filepath.Join(tempRoot, "backups")

	keyPaths, err := generateTestKeys(t, tempRoot)
	if err != nil {
		t.Skip("Skipping integration test: GPG key generation failed")
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

	backupCfg := Config{
		SourcePath: sourceDir,
		DestDir:    backupDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	backupPath, err := PerformBackup(backupCfg)
	require.NoError(t, err)

	// Corrupt the backup file by truncating it
	err = os.Truncate(backupPath, 100) // Truncate to just 100 bytes
	require.NoError(t, err)

	// Try to restore corrupted backup
	restoreDir := filepath.Join(tempRoot, "restore")
	restoreCfg := RestoreConfig{
		BackupFile: backupPath,
		DestPath:   restoreDir,
		Encryptor:  encryptor,
		Compressor: compressor,
		Verbose:    false,
	}

	err = PerformRestore(restoreCfg)
	assert.Error(t, err, "restore should fail with corrupted backup")
	assert.Contains(t, err.Error(), "restore pipeline failed", "error should indicate pipeline failure")
}
