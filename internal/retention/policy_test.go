package retention

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/icemarkom/secure-backup/internal/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "less than 1 hour",
			duration: 30 * time.Minute,
			want:     "30m",
		},
		{
			name:     "exactly 1 hour",
			duration: 1 * time.Hour,
			want:     "1h",
		},
		{
			name:     "hours only",
			duration: 12 * time.Hour,
			want:     "12h",
		},
		{
			name:     "1 day",
			duration: 24 * time.Hour,
			want:     "1d0h",
		},
		{
			name:     "multiple days",
			duration: 5*24*time.Hour + 3*time.Hour,
			want:     "5d3h",
		},
		{
			name:     "30 days",
			duration: 30 * 24 * time.Hour,
			want:     "30d0h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := format.Age(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "zero bytes",
			bytes: 0,
			want:  "0 B",
		},
		{
			name:  "bytes",
			bytes: 500,
			want:  "500 B",
		},
		{
			name:  "kilobytes",
			bytes: 1024,
			want:  "1.0 KiB",
		},
		{
			name:  "megabytes",
			bytes: 1024 * 1024,
			want:  "1.0 MiB",
		},
		{
			name:  "gigabytes",
			bytes: 5 * 1024 * 1024 * 1024,
			want:  "5.0 GiB",
		},
		{
			name:  "terabytes",
			bytes: 2 * 1024 * 1024 * 1024 * 1024,
			want:  "2.0 TiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := format.Size(tt.bytes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsBackupFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "valid backup file",
			filename: "backup_20240207_183000.tar.gz.gpg",
			want:     true,
		},
		{
			name:     "tar.gz file only",
			filename: "backup_20240207_183000.tar.gz",
			want:     false,
		},
		{
			name:     "random file",
			filename: "random.txt",
			want:     false,
		},
		{
			name:     "partial match",
			filename: "mybackup.tar.gz.gpg",
			want:     false,
		},
		{
			name:     "wrong extension",
			filename: "backup_20240207_183000.tar.gpg",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBackupFile(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestApplyPolicy_InvalidConfig tests error handling for invalid policy configurations
func TestApplyPolicy_InvalidConfig(t *testing.T) {
	tests := []struct {
		name       string
		policy     Policy
		wantErrMsg string
	}{
		{
			name: "zero retention days",
			policy: Policy{
				RetentionDays: 0,
				BackupDir:     "/tmp/backups",
			},
			wantErrMsg: "retention days must be positive",
		},
		{
			name: "negative retention days",
			policy: Policy{
				RetentionDays: -5,
				BackupDir:     "/tmp/backups",
			},
			wantErrMsg: "retention days must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := ApplyPolicy(tt.policy)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
			assert.Equal(t, 0, count)
		})
	}
}

// TestApplyPolicy_EmptyDirectory tests policy application on an empty directory
func TestApplyPolicy_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	policy := Policy{
		RetentionDays: 7,
		BackupDir:     tempDir,
		Pattern:       "backup_*.tar.gz.gpg",
		Verbose:       false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should delete 0 files from empty directory")
}

// TestApplyPolicy_DeleteOldBackups tests that old backups are deleted correctly
func TestApplyPolicy_DeleteOldBackups(t *testing.T) {
	tempDir := t.TempDir()

	// Create fake backup files with different ages
	now := time.Now()

	oldFiles := []struct {
		name string
		age  time.Duration
	}{
		{"backup_old1_20240101_120000.tar.gz.gpg", 10 * 24 * time.Hour}, // 10 days old
		{"backup_old2_20240102_120000.tar.gz.gpg", 15 * 24 * time.Hour}, // 15 days old
		{"backup_old3_20240103_120000.tar.gz.gpg", 30 * 24 * time.Hour}, // 30 days old
	}

	newFiles := []struct {
		name string
		age  time.Duration
	}{
		{"backup_new1_20240207_120000.tar.gz.gpg", 1 * 24 * time.Hour}, // 1 day old
		{"backup_new2_20240206_120000.tar.gz.gpg", 2 * 24 * time.Hour}, // 2 days old
		{"backup_new3_20240205_120000.tar.gz.gpg", 5 * 24 * time.Hour}, // 5 days old
	}

	// Create old files
	for _, f := range oldFiles {
		path := filepath.Join(tempDir, f.name)
		err := os.WriteFile(path, []byte("fake backup"), 0644)
		require.NoError(t, err)

		// Set modification time to the past
		modTime := now.Add(-f.age)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	// Create new files
	for _, f := range newFiles {
		path := filepath.Join(tempDir, f.name)
		err := os.WriteFile(path, []byte("fake backup"), 0644)
		require.NoError(t, err)

		// Set modification time to the past
		modTime := now.Add(-f.age)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	// Apply policy with 7-day retention
	policy := Policy{
		RetentionDays: 7,
		BackupDir:     tempDir,
		Pattern:       "backup_*.tar.gz.gpg",
		Verbose:       false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "should delete 3 old files")

	// Verify old files were deleted
	for _, f := range oldFiles {
		path := filepath.Join(tempDir, f.name)
		_, err := os.Stat(path)
		assert.True(t, os.IsNotExist(err), "old file %s should be deleted", f.name)
	}

	// Verify new files were kept
	for _, f := range newFiles {
		path := filepath.Join(tempDir, f.name)
		_, err := os.Stat(path)
		assert.NoError(t, err, "new file %s should be kept", f.name)
	}
}

// TestApplyPolicy_DefaultPattern tests that default pattern is used when empty
func TestApplyPolicy_DefaultPattern(t *testing.T) {
	tempDir := t.TempDir()

	// Create a backup file with default pattern
	now := time.Now()
	oldFile := filepath.Join(tempDir, "backup_test_20240101_120000.tar.gz.gpg")
	err := os.WriteFile(oldFile, []byte("old backup"), 0644)
	require.NoError(t, err)

	// Set to 10 days old
	modTime := now.Add(-10 * 24 * time.Hour)
	err = os.Chtimes(oldFile, modTime, modTime)
	require.NoError(t, err)

	// Apply policy with empty pattern (should use default)
	policy := Policy{
		RetentionDays: 7,
		BackupDir:     tempDir,
		Pattern:       "", // Empty - should use default
		Verbose:       false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should delete 1 file using default pattern")

	// Verify file was deleted
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err), "old file should be deleted")
}

// TestListBackups tests the ListBackups function
func TestListBackups(t *testing.T) {
	tempDir := t.TempDir()

	// Create fake backup files
	files := []struct {
		name string
		size int64
		age  time.Duration
	}{
		{"backup_test1_20240207_120000.tar.gz.gpg", 1024, 1 * time.Hour},
		{"backup_test2_20240206_120000.tar.gz.gpg", 2048, 24 * time.Hour},
		{"backup_test3_20240205_120000.tar.gz.gpg", 4096, 48 * time.Hour},
	}

	now := time.Now()
	for _, f := range files {
		path := filepath.Join(tempDir, f.name)
		content := make([]byte, f.size)
		err := os.WriteFile(path, content, 0644)
		require.NoError(t, err)

		// Set modification time
		modTime := now.Add(-f.age)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	// Also create a non-backup file (should be ignored)
	nonBackup := filepath.Join(tempDir, "other.txt")
	err := os.WriteFile(nonBackup, []byte("not a backup"), 0644)
	require.NoError(t, err)

	// List backups
	backups, err := ListBackups(tempDir, "backup_*.tar.gz.gpg")
	require.NoError(t, err)
	assert.Equal(t, 3, len(backups), "should find 3 backup files")

	// Verify backup info
	for i, backup := range backups {
		assert.Equal(t, files[i].name, backup.Name)
		assert.Equal(t, files[i].size, backup.Size)
		assert.Contains(t, backup.Path, tempDir)
		assert.NotZero(t, backup.Age)
	}
}

// TestListBackups_EmptyDirectory tests listing backups in an empty directory
func TestListBackups_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	backups, err := ListBackups(tempDir, "backup_*.tar.gz.gpg")
	require.NoError(t, err)
	assert.Equal(t, 0, len(backups), "should find 0 backups in empty directory")
}

// TestListBackups_DefaultPattern tests that default pattern is used when empty
func TestListBackups_DefaultPattern(t *testing.T) {
	tempDir := t.TempDir()

	// Create a backup file
	backupFile := filepath.Join(tempDir, "backup_test_20240207_120000.tar.gz.gpg")
	err := os.WriteFile(backupFile, []byte("backup"), 0644)
	require.NoError(t, err)

	// List with empty pattern (should use default)
	backups, err := ListBackups(tempDir, "")
	require.NoError(t, err)
	assert.Equal(t, 1, len(backups), "should find 1 backup using default pattern")
}

// TestApplyPolicy_DryRun tests that dry-run mode doesn't delete files
func TestApplyPolicy_DryRun(t *testing.T) {
	tempDir := t.TempDir()

	// Create fake backup files with different ages
	now := time.Now()

	oldFiles := []struct {
		name string
		age  time.Duration
	}{
		{"backup_old1_20240101_120000.tar.gz.gpg", 10 * 24 * time.Hour}, // 10 days old
		{"backup_old2_20240102_120000.tar.gz.gpg", 15 * 24 * time.Hour}, // 15 days old
	}

	// Create old files
	for _, f := range oldFiles {
		path := filepath.Join(tempDir, f.name)
		err := os.WriteFile(path, []byte("fake backup"), 0644)
		require.NoError(t, err)

		// Set modification time to the past
		modTime := now.Add(-f.age)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	// Apply policy with dry-run enabled
	policy := Policy{
		RetentionDays: 7,
		BackupDir:     tempDir,
		Pattern:       "backup_*.tar.gz.gpg",
		Verbose:       false,
		DryRun:        true,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should report 2 files would be deleted")

	// Verify files were NOT actually deleted
	for _, f := range oldFiles {
		path := filepath.Join(tempDir, f.name)
		_, err := os.Stat(path)
		assert.NoError(t, err, "file %s should NOT be deleted in dry-run mode", f.name)
	}
}

// TestApplyPolicy_DeletesManifestFiles tests that manifest files are deleted alongside backups
func TestApplyPolicy_DeletesManifestFiles(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create old backup + manifest pairs
	oldBackups := []struct {
		backup   string
		manifest string
		age      time.Duration
	}{
		{
			"backup_old1_20240101_120000.tar.gz.gpg",
			"backup_old1_20240101_120000_manifest.json",
			10 * 24 * time.Hour,
		},
		{
			"backup_old2_20240102_120000.tar.gz.gpg",
			"backup_old2_20240102_120000_manifest.json",
			15 * 24 * time.Hour,
		},
	}

	// Create a new backup + manifest pair (should NOT be deleted)
	newBackup := "backup_new1_20240207_120000.tar.gz.gpg"
	newManifest := "backup_new1_20240207_120000_manifest.json"

	for _, f := range oldBackups {
		backupPath := filepath.Join(tempDir, f.backup)
		manifestPath := filepath.Join(tempDir, f.manifest)
		require.NoError(t, os.WriteFile(backupPath, []byte("fake backup"), 0644))
		require.NoError(t, os.WriteFile(manifestPath, []byte("{}"), 0644))

		modTime := now.Add(-f.age)
		require.NoError(t, os.Chtimes(backupPath, modTime, modTime))
		require.NoError(t, os.Chtimes(manifestPath, modTime, modTime))
	}

	// Create new backup + manifest
	newBackupPath := filepath.Join(tempDir, newBackup)
	newManifestPath := filepath.Join(tempDir, newManifest)
	require.NoError(t, os.WriteFile(newBackupPath, []byte("new backup"), 0644))
	require.NoError(t, os.WriteFile(newManifestPath, []byte("{}"), 0644))

	modTime := now.Add(-1 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(newBackupPath, modTime, modTime))
	require.NoError(t, os.Chtimes(newManifestPath, modTime, modTime))

	// Apply policy with 7-day retention
	policy := Policy{
		RetentionDays: 7,
		BackupDir:     tempDir,
		Pattern:       "backup_*.tar.gz.gpg",
		Verbose:       false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should delete 2 old backups")

	// Verify old backups AND manifests were deleted
	for _, f := range oldBackups {
		_, err := os.Stat(filepath.Join(tempDir, f.backup))
		assert.True(t, os.IsNotExist(err), "old backup %s should be deleted", f.backup)

		_, err = os.Stat(filepath.Join(tempDir, f.manifest))
		assert.True(t, os.IsNotExist(err), "old manifest %s should be deleted", f.manifest)
	}

	// Verify new backup AND manifest were kept
	_, err = os.Stat(newBackupPath)
	assert.NoError(t, err, "new backup should be kept")

	_, err = os.Stat(newManifestPath)
	assert.NoError(t, err, "new manifest should be kept")
}

// TestApplyPolicy_DryRun_ReportsManifests tests that dry-run reports manifest deletion
func TestApplyPolicy_DryRun_ReportsManifests(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create old backup + manifest
	backupName := "backup_old1_20240101_120000.tar.gz.gpg"
	manifestName := "backup_old1_20240101_120000_manifest.json"

	backupPath := filepath.Join(tempDir, backupName)
	manifestPath := filepath.Join(tempDir, manifestName)
	require.NoError(t, os.WriteFile(backupPath, []byte("fake backup"), 0644))
	require.NoError(t, os.WriteFile(manifestPath, []byte("{}"), 0644))

	modTime := now.Add(-10 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(backupPath, modTime, modTime))
	require.NoError(t, os.Chtimes(manifestPath, modTime, modTime))

	// Apply policy with dry-run
	policy := Policy{
		RetentionDays: 7,
		BackupDir:     tempDir,
		Pattern:       "backup_*.tar.gz.gpg",
		DryRun:        true,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should report 1 backup would be deleted")

	// Verify neither backup nor manifest were actually deleted
	_, err = os.Stat(backupPath)
	assert.NoError(t, err, "backup should NOT be deleted in dry-run")

	_, err = os.Stat(manifestPath)
	assert.NoError(t, err, "manifest should NOT be deleted in dry-run")
}
