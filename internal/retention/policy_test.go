package retention

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPolicy(t *testing.T) {
	tmpDir := t.TempDir()

	// Create backups with different ages
	now := time.Now()

	// Create old backup (40 days old)
	oldFile := filepath.Join(tmpDir, "backup_old_20250101.tar.gz.gpg")
	err := os.WriteFile(oldFile, []byte("old backup"), 0644)
	require.NoError(t, err)
	err = os.Chtimes(oldFile, now.AddDate(0, 0, -40), now.AddDate(0, 0, -40))
	require.NoError(t, err)

	// Create recent backup (10 days old)
	recentFile := filepath.Join(tmpDir, "backup_recent_20260128.tar.gz.gpg")
	err = os.WriteFile(recentFile, []byte("recent backup"), 0644)
	require.NoError(t, err)
	err = os.Chtimes(recentFile, now.AddDate(0, 0, -10), now.AddDate(0, 0, -10))
	require.NoError(t, err)

	// Create current backup (today)
	currentFile := filepath.Join(tmpDir, "backup_current_20260207.tar.gz.gpg")
	err = os.WriteFile(currentFile, []byte("current backup"), 0644)
	require.NoError(t, err)

	// Apply 30-day retention policy
	policy := Policy{
		RetentionDays: 30,
		BackupDir:     tmpDir,
		Pattern:       "backup_*.tar.gz.gpg",
		Verbose:       false,
	}

	deletedCount, err := ApplyPolicy(policy)
	require.NoError(t, err)

	// Should delete only the 40-day-old backup
	assert.Equal(t, 1, deletedCount)

	// Verify old file was deleted
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err))

	// Verify recent file still exists
	_, err = os.Stat(recentFile)
	assert.NoError(t, err)

	// Verify current file still exists
	_, err = os.Stat(currentFile)
	assert.NoError(t, err)
}

func TestApplyPolicy_InvalidRetention(t *testing.T) {
	policy := Policy{
		RetentionDays: 0,
		BackupDir:     "/tmp",
	}

	_, err := ApplyPolicy(policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retention days must be positive")
}

func TestApplyPolicy_NoBackupsToDelete(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only recent backups
	recentFile := filepath.Join(tmpDir, "backup_recent_20260207.tar.gz.gpg")
	err := os.WriteFile(recentFile, []byte("recent backup"), 0644)
	require.NoError(t, err)

	policy := Policy{
		RetentionDays: 30,
		BackupDir:     tmpDir,
		Pattern:       "backup_*.tar.gz.gpg",
		Verbose:       false,
	}

	deletedCount, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 0, deletedCount)

	// File should still exist
	_, err = os.Stat(recentFile)
	assert.NoError(t, err)
}

func TestListBackups(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test backups
	files := []string{
		"backup_app1_20260207.tar.gz.gpg",
		"backup_app2_20260206.tar.gz.gpg",
		"backup_app3_20260205.tar.gz.gpg",
		"other_file.txt", // Should be ignored
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// List backups
	backups, err := ListBackups(tmpDir, "backup_*.tar.gz.gpg")
	require.NoError(t, err)

	// Should find 3 backups (not other_file.txt)
	assert.Len(t, backups, 3)

	// Verify backup info
	for _, backup := range backups {
		assert.NotEmpty(t, backup.Name)
		assert.NotEmpty(t, backup.Path)
		assert.Greater(t, backup.Size, int64(0))
		assert.False(t, backup.ModTime.IsZero())
	}
}

func TestListBackups_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	backups, err := ListBackups(tmpDir, "backup_*.tar.gz.gpg")
	require.NoError(t, err)
	assert.Empty(t, backups)
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
		{1099511627776, "1.0 TiB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBackupFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"backup_app_20260207.tar.gz.gpg", true},
		{"backup_data_20260101.tar.gz.gpg", true},
		{"backup_test.tar.zst.gpg", true},
		{"backup_age.tar.gz.age", true},
		{"not_a_backup.txt", false},
		{"app_backup_20260207.tar.gz.gpg", false}, // doesn't start with backup_
		{"backup_partial.tar.gz", false},          // not encrypted
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsBackupFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyPolicy_CustomPattern(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	// Create files matching custom pattern
	oldFile := filepath.Join(tmpDir, "custom_backup_20250101.tar.gz.gpg")
	err := os.WriteFile(oldFile, []byte("old"), 0644)
	require.NoError(t, err)
	err = os.Chtimes(oldFile, now.AddDate(0, 0, -40), now.AddDate(0, 0, -40))
	require.NoError(t, err)

	// Create file NOT matching pattern (should be preserved)
	otherFile := filepath.Join(tmpDir, "backup_other_20250101.tar.gz.gpg")
	err = os.WriteFile(otherFile, []byte("other"), 0644)
	require.NoError(t, err)
	err = os.Chtimes(otherFile, now.AddDate(0, 0, -40), now.AddDate(0, 0, -40))
	require.NoError(t, err)

	policy := Policy{
		RetentionDays: 30,
		BackupDir:     tmpDir,
		Pattern:       "custom_*.tar.gz.gpg",
		Verbose:       false,
	}

	deletedCount, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 1, deletedCount)

	// Custom pattern file should be deleted
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err))

	// Other pattern file should still exist
	_, err = os.Stat(otherFile)
	assert.NoError(t, err)
}
