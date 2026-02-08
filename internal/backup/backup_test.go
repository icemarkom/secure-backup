package backup

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/icemarkom/secure-backup/internal/compress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/openpgp"
)

// Mock encryptor for testing
type mockEncryptor struct{}

func (m *mockEncryptor) Encrypt(plaintext io.Reader) (io.Reader, error) {
	// Just pass through for testing
	data, err := io.ReadAll(plaintext)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (m *mockEncryptor) Decrypt(ciphertext io.Reader) (io.Reader, error) {
	// Just pass through for testing
	data, err := io.ReadAll(ciphertext)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (m *mockEncryptor) Type() string {
	return "mock"
}

// Helper to generate real test keys
func generateTestKeys(t *testing.T) (publicKeyPath, privateKeyPath string) {
	tmpDir := t.TempDir()

	entity, err := openpgp.NewEntity("Test User", "test", "test@example.com", nil)
	require.NoError(t, err)

	publicKeyPath = filepath.Join(tmpDir, "public.asc")
	pubFile, err := os.Create(publicKeyPath)
	require.NoError(t, err)
	defer pubFile.Close()
	err = entity.Serialize(pubFile)
	require.NoError(t, err)

	privateKeyPath = filepath.Join(tmpDir, "private.asc")
	privFile, err := os.Create(privateKeyPath)
	require.NoError(t, err)
	defer privFile.Close()
	err = entity.SerializePrivate(privFile, nil)
	require.NoError(t, err)

	return publicKeyPath, privateKeyPath
}

func TestPerformBackup_WithMockEncryptor(t *testing.T) {
	// Create source directory with test files
	srcDir := t.TempDir()
	err := os.WriteFile(filepath.Join(srcDir, "test1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(srcDir, "test2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)

	// Create destination directory
	destDir := t.TempDir()

	// Create compressor
	compressor, err := compress.NewGzipCompressor(0)
	require.NoError(t, err)

	// Perform backup
	cfg := Config{
		SourcePath: srcDir,
		DestDir:    destDir,
		Encryptor:  &mockEncryptor{},
		Compressor: compressor,
		Verbose:    false,
	}

	outputPath, err := PerformBackup(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, outputPath)

	// Verify backup file was created
	_, err = os.Stat(outputPath)
	assert.NoError(t, err)

	// Verify filename format
	assert.Contains(t, filepath.Base(outputPath), "backup_")
	assert.Contains(t, filepath.Base(outputPath), ".tar.gz.gpg")
}

func TestPerformBackup_InvalidSource(t *testing.T) {
	destDir := t.TempDir()
	compressor, err := compress.NewGzipCompressor(0)
	require.NoError(t, err)

	cfg := Config{
		SourcePath: "/nonexistent/path",
		DestDir:    destDir,
		Encryptor:  &mockEncryptor{},
		Compressor: compressor,
	}

	_, err = PerformBackup(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source path")
}

func TestPerformRestore_WithMockEncryptor(t *testing.T) {
	// Create a test backup first
	srcDir := t.TempDir()
	err := os.WriteFile(filepath.Join(srcDir, "testfile.txt"), []byte("test content"), 0644)
	require.NoError(t, err)

	backupDir := t.TempDir()
	compressor, err := compress.NewGzipCompressor(0)
	require.NoError(t, err)

	// Create backup
	backupCfg := Config{
		SourcePath: srcDir,
		DestDir:    backupDir,
		Encryptor:  &mockEncryptor{},
		Compressor: compressor,
		Verbose:    false,
	}

	backupFile, err := PerformBackup(backupCfg)
	require.NoError(t, err)

	// Restore to new location
	restoreDir := t.TempDir()
	restoreCfg := RestoreConfig{
		BackupFile: backupFile,
		DestPath:   restoreDir,
		Encryptor:  &mockEncryptor{},
		Compressor: compressor,
		Verbose:    false,
	}

	err = PerformRestore(restoreCfg)
	require.NoError(t, err)

	// Verify restored file
	restoredFile := filepath.Join(restoreDir, filepath.Base(srcDir), "testfile.txt")
	content, err := os.ReadFile(restoredFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestPerformRestore_InvalidBackupFile(t *testing.T) {
	restoreDir := t.TempDir()
	compressor, err := compress.NewGzipCompressor(0)
	require.NoError(t, err)

	cfg := RestoreConfig{
		BackupFile: "/nonexistent/backup.tar.gz.gpg",
		DestPath:   restoreDir,
		Encryptor:  &mockEncryptor{},
		Compressor: compressor,
	}

	err = PerformRestore(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup file not found")
}

func TestPerformVerify_Quick(t *testing.T) {
	// Create a test backup
	srcDir := t.TempDir()
	err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	backupDir := t.TempDir()
	compressor, err := compress.NewGzipCompressor(0)
	require.NoError(t, err)

	backupCfg := Config{
		SourcePath: srcDir,
		DestDir:    backupDir,
		Encryptor:  &mockEncryptor{},
		Compressor: compressor,
	}

	backupFile, err := PerformBackup(backupCfg)
	require.NoError(t, err)

	// Quick verify
	verifyCfg := VerifyConfig{
		BackupFile: backupFile,
		Quick:      true,
		Verbose:    false,
	}

	err = PerformVerify(verifyCfg)
	assert.NoError(t, err)
}

func TestPerformVerify_Full(t *testing.T) {
	// Create a test backup
	srcDir := t.TempDir()
	err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	backupDir := t.TempDir()
	compressor, err := compress.NewGzipCompressor(0)
	require.NoError(t, err)

	backupCfg := Config{
		SourcePath: srcDir,
		DestDir:    backupDir,
		Encryptor:  &mockEncryptor{},
		Compressor: compressor,
	}

	backupFile, err := PerformBackup(backupCfg)
	require.NoError(t, err)

	// Full verify
	verifyCfg := VerifyConfig{
		BackupFile: backupFile,
		Encryptor:  &mockEncryptor{},
		Compressor: compressor,
		Quick:      false,
		Verbose:    false,
	}

	err = PerformVerify(verifyCfg)
	assert.NoError(t, err)
}

func TestPerformVerify_InvalidBackupFile(t *testing.T) {
	cfg := VerifyConfig{
		BackupFile: "/nonexistent/backup.tar.gz.gpg",
		Quick:      true,
	}

	err := PerformVerify(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup file not found")
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1048576, "1.0 MiB"},
	}

	for _, tt := range tests {
		result := formatSize(tt.bytes)
		assert.Equal(t, tt.expected, result)
	}
}
