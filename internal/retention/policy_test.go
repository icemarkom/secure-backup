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

func TestDefaultKeepLast(t *testing.T) {
	assert.Equal(t, 0, DefaultKeepLast, "default should be 0 (keep all)")
}

// TestApplyPolicy_InvalidConfig tests error handling for invalid policy configurations
func TestApplyPolicy_InvalidConfig(t *testing.T) {
	tests := []struct {
		name       string
		policy     Policy
		wantErrMsg string
	}{
		{
			name: "zero keep count",
			policy: Policy{
				KeepLast:  0,
				BackupDir: "/tmp/backups",
			},
			wantErrMsg: "keep count must be positive",
		},
		{
			name: "negative keep count",
			policy: Policy{
				KeepLast:  -5,
				BackupDir: "/tmp/backups",
			},
			wantErrMsg: "keep count must be positive",
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
		KeepLast:  3,
		BackupDir: tempDir,
		Pattern:   "backup_*.tar.gz.gpg",
		Verbose:   false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should delete 0 files from empty directory")
}

// TestApplyPolicy_DeleteOldBackups tests that excess backups beyond KeepLast are deleted
func TestApplyPolicy_DeleteOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 6 backup files with different ages
	allFiles := []struct {
		name string
		age  time.Duration
	}{
		{"backup_newest_20240207_120000.tar.gz.gpg", 1 * time.Hour},       // newest
		{"backup_second_20240207_110000.tar.gz.gpg", 2 * time.Hour},       // 2nd
		{"backup_third_20240207_100000.tar.gz.gpg", 3 * time.Hour},        // 3rd
		{"backup_fourth_20240206_120000.tar.gz.gpg", 1 * 24 * time.Hour},  // 4th (should be deleted)
		{"backup_fifth_20240205_120000.tar.gz.gpg", 2 * 24 * time.Hour},   // 5th (should be deleted)
		{"backup_oldest_20240201_120000.tar.gz.gpg", 10 * 24 * time.Hour}, // 6th (should be deleted)
	}

	for _, f := range allFiles {
		path := filepath.Join(tempDir, f.name)
		err := os.WriteFile(path, []byte("fake backup"), 0644)
		require.NoError(t, err)

		modTime := now.Add(-f.age)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	// Keep last 3
	policy := Policy{
		KeepLast:  3,
		BackupDir: tempDir,
		Pattern:   "backup_*.tar.gz.gpg",
		Verbose:   false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "should delete 3 oldest backups")

	// Verify the 3 newest are kept
	for _, f := range allFiles[:3] {
		path := filepath.Join(tempDir, f.name)
		_, err := os.Stat(path)
		assert.NoError(t, err, "newest file %s should be kept", f.name)
	}

	// Verify the 3 oldest are deleted
	for _, f := range allFiles[3:] {
		path := filepath.Join(tempDir, f.name)
		_, err := os.Stat(path)
		assert.True(t, os.IsNotExist(err), "old file %s should be deleted", f.name)
	}
}

// TestApplyPolicy_DefaultPattern tests that default pattern is used when empty
func TestApplyPolicy_DefaultPattern(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 2 backup files
	files := []string{
		"backup_old_20240101_120000.tar.gz.gpg",
		"backup_new_20240207_120000.tar.gz.gpg",
	}
	for i, name := range files {
		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte("fake backup"), 0644)
		require.NoError(t, err)
		modTime := now.Add(-time.Duration(len(files)-i) * 24 * time.Hour)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	// Apply policy with empty pattern (should use default), keep 1
	policy := Policy{
		KeepLast:  1,
		BackupDir: tempDir,
		Pattern:   "", // Empty - should use default
		Verbose:   false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should delete 1 file using default pattern")

	// Verify newer file was kept
	_, err = os.Stat(filepath.Join(tempDir, files[1]))
	assert.NoError(t, err, "newer file should be kept")

	// Verify older file was deleted
	_, err = os.Stat(filepath.Join(tempDir, files[0]))
	assert.True(t, os.IsNotExist(err), "older file should be deleted")
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
	now := time.Now()

	// Create 4 backups
	files := []struct {
		name string
		age  time.Duration
	}{
		{"backup_new1_20240207_120000.tar.gz.gpg", 1 * time.Hour},
		{"backup_new2_20240206_120000.tar.gz.gpg", 24 * time.Hour},
		{"backup_old1_20240101_120000.tar.gz.gpg", 10 * 24 * time.Hour},
		{"backup_old2_20240102_120000.tar.gz.gpg", 15 * 24 * time.Hour},
	}

	for _, f := range files {
		path := filepath.Join(tempDir, f.name)
		err := os.WriteFile(path, []byte("fake backup"), 0644)
		require.NoError(t, err)

		modTime := now.Add(-f.age)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	// Keep 2, dry-run
	policy := Policy{
		KeepLast:  2,
		BackupDir: tempDir,
		Pattern:   "backup_*.tar.gz.gpg",
		Verbose:   false,
		DryRun:    true,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should report 2 files would be deleted")

	// Verify files were NOT actually deleted
	for _, f := range files {
		path := filepath.Join(tempDir, f.name)
		_, err := os.Stat(path)
		assert.NoError(t, err, "file %s should NOT be deleted in dry-run mode", f.name)
	}
}

// TestApplyPolicy_DeletesManifestFiles tests that manifest files are deleted alongside backups
func TestApplyPolicy_DeletesManifestFiles(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create backup + manifest pairs with different ages
	pairs := []struct {
		backup   string
		manifest string
		age      time.Duration
	}{
		{
			"backup_new1_20240207_120000.tar.gz.gpg",
			"backup_new1_20240207_120000_manifest.json",
			1 * 24 * time.Hour,
		},
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

	for _, p := range pairs {
		backupPath := filepath.Join(tempDir, p.backup)
		manifestPath := filepath.Join(tempDir, p.manifest)
		require.NoError(t, os.WriteFile(backupPath, []byte("fake backup"), 0644))
		require.NoError(t, os.WriteFile(manifestPath, []byte("{}"), 0644))

		modTime := now.Add(-p.age)
		require.NoError(t, os.Chtimes(backupPath, modTime, modTime))
		require.NoError(t, os.Chtimes(manifestPath, modTime, modTime))
	}

	// Keep 1 (newest), delete 2
	policy := Policy{
		KeepLast:  1,
		BackupDir: tempDir,
		Pattern:   "backup_*.tar.gz.gpg",
		Verbose:   false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should delete 2 old backups")

	// Verify newest backup AND manifest were kept
	_, err = os.Stat(filepath.Join(tempDir, pairs[0].backup))
	assert.NoError(t, err, "new backup should be kept")
	_, err = os.Stat(filepath.Join(tempDir, pairs[0].manifest))
	assert.NoError(t, err, "new manifest should be kept")

	// Verify old backups AND manifests were deleted
	for _, p := range pairs[1:] {
		_, err := os.Stat(filepath.Join(tempDir, p.backup))
		assert.True(t, os.IsNotExist(err), "old backup %s should be deleted", p.backup)

		_, err = os.Stat(filepath.Join(tempDir, p.manifest))
		assert.True(t, os.IsNotExist(err), "old manifest %s should be deleted", p.manifest)
	}
}

// TestApplyPolicy_DryRun_ReportsManifests tests that dry-run reports manifest deletion
func TestApplyPolicy_DryRun_ReportsManifests(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 2 backups + manifests (keep 1, report 1 for deletion)
	newBackup := "backup_new1_20240207_120000.tar.gz.gpg"
	newManifest := "backup_new1_20240207_120000_manifest.json"
	oldBackup := "backup_old1_20240101_120000.tar.gz.gpg"
	oldManifest := "backup_old1_20240101_120000_manifest.json"

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, newBackup), []byte("new"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, newManifest), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, oldBackup), []byte("old"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, oldManifest), []byte("{}"), 0644))

	newTime := now.Add(-1 * time.Hour)
	oldTime := now.Add(-10 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(tempDir, newBackup), newTime, newTime))
	require.NoError(t, os.Chtimes(filepath.Join(tempDir, newManifest), newTime, newTime))
	require.NoError(t, os.Chtimes(filepath.Join(tempDir, oldBackup), oldTime, oldTime))
	require.NoError(t, os.Chtimes(filepath.Join(tempDir, oldManifest), oldTime, oldTime))

	// Keep 1, dry-run
	policy := Policy{
		KeepLast:  1,
		BackupDir: tempDir,
		Pattern:   "backup_*.tar.gz.gpg",
		DryRun:    true,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should report 1 backup would be deleted")

	// Verify nothing was actually deleted
	_, err = os.Stat(filepath.Join(tempDir, newBackup))
	assert.NoError(t, err, "new backup should NOT be deleted in dry-run")
	_, err = os.Stat(filepath.Join(tempDir, oldBackup))
	assert.NoError(t, err, "old backup should NOT be deleted in dry-run")
	_, err = os.Stat(filepath.Join(tempDir, oldManifest))
	assert.NoError(t, err, "old manifest should NOT be deleted in dry-run")
}

// TestApplyPolicy_KeepMoreThanExist tests that nothing is deleted when N > total files
func TestApplyPolicy_KeepMoreThanExist(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 2 backups
	files := []string{
		"backup_a_20240207_120000.tar.gz.gpg",
		"backup_b_20240206_120000.tar.gz.gpg",
	}
	for i, name := range files {
		path := filepath.Join(tempDir, name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-time.Duration(i+1) * time.Hour)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
	}

	// Keep 10 (more than we have)
	policy := Policy{
		KeepLast:  10,
		BackupDir: tempDir,
		Pattern:   "backup_*.tar.gz.gpg",
		Verbose:   false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should delete nothing when KeepLast > total files")

	// Verify all files are still there
	for _, name := range files {
		_, err := os.Stat(filepath.Join(tempDir, name))
		assert.NoError(t, err, "file %s should be kept", name)
	}
}

// TestApplyPolicy_KeepOne tests that KeepLast=1 keeps only the newest backup
func TestApplyPolicy_KeepOne(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 4 backups
	files := []struct {
		name string
		age  time.Duration
	}{
		{"backup_newest_20240207_120000.tar.gz.gpg", 1 * time.Hour},
		{"backup_second_20240206_120000.tar.gz.gpg", 24 * time.Hour},
		{"backup_third_20240205_120000.tar.gz.gpg", 48 * time.Hour},
		{"backup_oldest_20240201_120000.tar.gz.gpg", 7 * 24 * time.Hour},
	}

	for _, f := range files {
		path := filepath.Join(tempDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-f.age)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
	}

	// Keep only 1
	policy := Policy{
		KeepLast:  1,
		BackupDir: tempDir,
		Pattern:   "backup_*.tar.gz.gpg",
		Verbose:   false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "should delete 3 backups")

	// Verify only newest is kept
	_, err = os.Stat(filepath.Join(tempDir, files[0].name))
	assert.NoError(t, err, "newest file should be kept")

	// Verify all others are deleted
	for _, f := range files[1:] {
		_, err := os.Stat(filepath.Join(tempDir, f.name))
		assert.True(t, os.IsNotExist(err), "file %s should be deleted", f.name)
	}
}
