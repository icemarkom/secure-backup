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

package retention

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/icemarkom/secure-backup/internal/common"
	"github.com/icemarkom/secure-backup/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTestManifest creates a valid manifest JSON file for testing.
func writeTestManifest(t *testing.T, backupPath, sourcePath, hostname string) {
	t.Helper()
	m := manifest.Manifest{
		CreatedAt: time.Now(),
		CreatedBy: manifest.CreatedBy{
			Tool:     "secure-backup",
			Version:  "test",
			Hostname: hostname,
		},
		SourcePath:        sourcePath,
		BackupFile:        filepath.Base(backupPath),
		Compression:       "gzip",
		Encryption:        "gpg",
		ChecksumAlgorithm: "sha256",
		ChecksumValue:     "test-checksum",
	}
	data, err := json.Marshal(m)
	require.NoError(t, err)

	manifestPath := manifest.ManifestPath(backupPath)
	require.NoError(t, os.WriteFile(manifestPath, data, 0644))
}

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
			got := common.Age(tt.duration)
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
			got := common.Size(tt.bytes)
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
			name:     "no compression gpg",
			filename: "backup_20240207_183000.tar.gpg",
			want:     true,
		},
		{
			name:     "no compression age",
			filename: "backup_20240207_183000.tar.age",
			want:     true,
		},
		{
			name:     "gzip age",
			filename: "backup_20240207_183000.tar.gz.age",
			want:     true,
		},
		{
			name:     "zstd gpg",
			filename: "backup_20240207_183000.tar.zst.gpg",
			want:     true,
		},
		{
			name:     "zstd age",
			filename: "backup_20240207_183000.tar.zst.age",
			want:     true,
		},
		{
			name:     "lz4 gpg",
			filename: "backup_20240207_183000.tar.lz4.gpg",
			want:     true,
		},
		{
			name:     "lz4 age",
			filename: "backup_20240207_183000.tar.lz4.age",
			want:     true,
		},
		{
			name:     "wrong extension",
			filename: "backup_20240207_183000.tar.bz2.gpg",
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
		Verbose:   false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should delete 0 files from empty directory")
}

// TestApplyPolicy_DeleteOldBackups tests that excess backups beyond KeepLast are deleted
// (single source, all managed — basic retention within one group)
func TestApplyPolicy_DeleteOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 6 backup files with different ages, all from same source
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

		writeTestManifest(t, path, "/data", "host1")
	}

	// Keep last 3
	policy := Policy{
		KeepLast:  3,
		BackupDir: tempDir,
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
	backups, err := ListBackups(tempDir)
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

	backups, err := ListBackups(tempDir)
	require.NoError(t, err)
	assert.Equal(t, 0, len(backups), "should find 0 backups in empty directory")
}

// TestApplyPolicy_DryRun tests that dry-run mode doesn't delete files
func TestApplyPolicy_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 4 backups with manifests (same source)
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

		writeTestManifest(t, path, "/data", "host1")
	}

	// Keep 2, dry-run
	policy := Policy{
		KeepLast:  2,
		BackupDir: tempDir,
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

	// Create backup + manifest pairs with different ages (same source)
	pairs := []struct {
		backup string
		age    time.Duration
	}{
		{"backup_new1_20240207_120000.tar.gz.gpg", 1 * 24 * time.Hour},
		{"backup_old1_20240101_120000.tar.gz.gpg", 10 * 24 * time.Hour},
		{"backup_old2_20240102_120000.tar.gz.gpg", 15 * 24 * time.Hour},
	}

	for _, p := range pairs {
		backupPath := filepath.Join(tempDir, p.backup)
		require.NoError(t, os.WriteFile(backupPath, []byte("fake backup"), 0644))

		modTime := now.Add(-p.age)
		require.NoError(t, os.Chtimes(backupPath, modTime, modTime))

		writeTestManifest(t, backupPath, "/data", "host1")
	}

	// Keep 1 (newest), delete 2
	policy := Policy{
		KeepLast:  1,
		BackupDir: tempDir,
		Verbose:   false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should delete 2 old backups")

	// Verify newest backup AND manifest were kept
	newestPath := filepath.Join(tempDir, pairs[0].backup)
	_, err = os.Stat(newestPath)
	assert.NoError(t, err, "new backup should be kept")
	_, err = os.Stat(manifest.ManifestPath(newestPath))
	assert.NoError(t, err, "new manifest should be kept")

	// Verify old backups AND manifests were deleted
	for _, p := range pairs[1:] {
		backupPath := filepath.Join(tempDir, p.backup)
		_, err := os.Stat(backupPath)
		assert.True(t, os.IsNotExist(err), "old backup %s should be deleted", p.backup)

		_, err = os.Stat(manifest.ManifestPath(backupPath))
		assert.True(t, os.IsNotExist(err), "old manifest for %s should be deleted", p.backup)
	}
}

// TestApplyPolicy_DryRun_ReportsManifests tests that dry-run reports manifest deletion
func TestApplyPolicy_DryRun_ReportsManifests(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 2 backups + manifests (keep 1, report 1 for deletion)
	newBackup := "backup_new1_20240207_120000.tar.gz.gpg"
	oldBackup := "backup_old1_20240101_120000.tar.gz.gpg"

	newPath := filepath.Join(tempDir, newBackup)
	oldPath := filepath.Join(tempDir, oldBackup)

	require.NoError(t, os.WriteFile(newPath, []byte("new"), 0644))
	require.NoError(t, os.WriteFile(oldPath, []byte("old"), 0644))

	newTime := now.Add(-1 * time.Hour)
	oldTime := now.Add(-10 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(newPath, newTime, newTime))
	require.NoError(t, os.Chtimes(oldPath, oldTime, oldTime))

	writeTestManifest(t, newPath, "/data", "host1")
	writeTestManifest(t, oldPath, "/data", "host1")

	// Keep 1, dry-run
	policy := Policy{
		KeepLast:  1,
		DryRun:    true,
		BackupDir: tempDir,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should report 1 backup would be deleted")

	// Verify nothing was actually deleted
	_, err = os.Stat(newPath)
	assert.NoError(t, err, "new backup should NOT be deleted in dry-run")
	_, err = os.Stat(oldPath)
	assert.NoError(t, err, "old backup should NOT be deleted in dry-run")
}

// TestApplyPolicy_KeepMoreThanExist tests that nothing is deleted when N > total files
func TestApplyPolicy_KeepMoreThanExist(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 2 backups with manifests
	files := []string{
		"backup_a_20240207_120000.tar.gz.gpg",
		"backup_b_20240206_120000.tar.gz.gpg",
	}
	for i, name := range files {
		path := filepath.Join(tempDir, name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-time.Duration(i+1) * time.Hour)
		require.NoError(t, os.Chtimes(path, modTime, modTime))

		writeTestManifest(t, path, "/data", "host1")
	}

	// Keep 10 (more than we have)
	policy := Policy{
		KeepLast:  10,
		BackupDir: tempDir,
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

	// Create 4 backups with manifests (same source)
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

		writeTestManifest(t, path, "/data", "host1")
	}

	// Keep only 1
	policy := Policy{
		KeepLast:  1,
		BackupDir: tempDir,
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

// TestApplyPolicy_MixedExtensions tests that retention counts ALL valid backup
// extensions together within the same (host, source) group (the core fix for issue #43).
func TestApplyPolicy_MixedExtensions(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create backups with different compression/encryption combos, all same source
	allFiles := []struct {
		name string
		age  time.Duration
	}{
		{"backup_data_20240207_120000.tar.gz.gpg", 1 * time.Hour},      // newest — keep
		{"backup_data_20240207_110000.tar.zst.gpg", 2 * time.Hour},     // 2nd — keep
		{"backup_data_20240207_100000.tar.lz4.gpg", 3 * time.Hour},     // 3rd — keep
		{"backup_data_20240207_090000.tar.gpg", 4 * time.Hour},         // 4th — keep
		{"backup_data_20240206_120000.tar.gz.age", 24 * time.Hour},     // 5th — delete
		{"backup_data_20240205_120000.tar.zst.age", 48 * time.Hour},    // 6th — delete
		{"backup_data_20240204_120000.tar.lz4.age", 72 * time.Hour},    // 7th — delete
		{"backup_data_20240203_120000.tar.age", 96 * time.Hour},        // 8th — delete
		{"backup_data_20240201_120000.tar.gz.gpg", 7 * 24 * time.Hour}, // 9th — delete
	}

	for _, f := range allFiles {
		path := filepath.Join(tempDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte("fake backup"), 0644))
		modTime := now.Add(-f.age)
		require.NoError(t, os.Chtimes(path, modTime, modTime))

		writeTestManifest(t, path, "/data", "host1")
	}

	// Keep 4 — should keep the 4 newest regardless of their extension
	policy := Policy{
		KeepLast:  4,
		BackupDir: tempDir,
		Verbose:   false,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 5, count, "should delete 5 oldest backups across all extensions")

	// Verify 4 newest are kept
	for _, f := range allFiles[:4] {
		_, err := os.Stat(filepath.Join(tempDir, f.name))
		assert.NoError(t, err, "file %s should be kept", f.name)
	}

	// Verify 5 oldest are deleted
	for _, f := range allFiles[4:] {
		_, err := os.Stat(filepath.Join(tempDir, f.name))
		assert.True(t, os.IsNotExist(err), "file %s should be deleted", f.name)
	}
}

// TestListBackups_MixedExtensions tests that ListBackups finds all valid backup types
func TestListBackups_MixedExtensions(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{
		"backup_test1_20240207_120000.tar.gz.gpg",
		"backup_test2_20240206_120000.tar.gpg",
		"backup_test3_20240205_120000.tar.age",
		"backup_test4_20240204_120000.tar.gz.age",
		"backup_test5_20240203_120000.tar.zst.gpg",
		"backup_test6_20240202_120000.tar.zst.age",
		"backup_test7_20240201_120000.tar.lz4.gpg",
		"backup_test8_20240131_120000.tar.lz4.age",
	}

	for _, name := range files {
		path := filepath.Join(tempDir, name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
	}

	// Also create non-backup files (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "other.txt"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "backup_manifest.json"), []byte("{}"), 0644))

	backups, err := ListBackups(tempDir)
	require.NoError(t, err)
	assert.Equal(t, 8, len(backups), "should find all 8 backup files regardless of extension")
}

// ═══════════════════════════════════════════
// SCOPED RETENTION TESTS (issue #45)
// ═══════════════════════════════════════════

// TestApplyPolicy_TwoSources tests retention scoping with two different source paths.
func TestApplyPolicy_TwoSources(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Source A: 3 backups
	sourceAFiles := []struct {
		name string
		age  time.Duration
	}{
		{"backup_srcA_20240207_120000.tar.gz.gpg", 1 * time.Hour},
		{"backup_srcA_20240207_110000.tar.gz.gpg", 2 * time.Hour},
		{"backup_srcA_20240207_100000.tar.gz.gpg", 3 * time.Hour}, // should be deleted
	}
	for _, f := range sourceAFiles {
		path := filepath.Join(tempDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-f.age)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
		writeTestManifest(t, path, "/data", "host1")
	}

	// Source B: 3 backups
	sourceBFiles := []struct {
		name string
		age  time.Duration
	}{
		{"backup_srcB_20240207_120000.tar.gz.gpg", 1 * time.Hour},
		{"backup_srcB_20240207_110000.tar.gz.gpg", 2 * time.Hour},
		{"backup_srcB_20240207_100000.tar.gz.gpg", 3 * time.Hour}, // should be deleted
	}
	for _, f := range sourceBFiles {
		path := filepath.Join(tempDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-f.age)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
		writeTestManifest(t, path, "/logs", "host1")
	}

	// Keep 2 per source
	policy := Policy{
		KeepLast:  2,
		BackupDir: tempDir,
		Verbose:   true,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should delete 1 from each source (2 total)")

	// Verify 2 newest per source are kept
	for _, f := range sourceAFiles[:2] {
		_, err := os.Stat(filepath.Join(tempDir, f.name))
		assert.NoError(t, err, "source A file %s should be kept", f.name)
	}
	for _, f := range sourceBFiles[:2] {
		_, err := os.Stat(filepath.Join(tempDir, f.name))
		assert.NoError(t, err, "source B file %s should be kept", f.name)
	}

	// Verify oldest per source are deleted
	_, err = os.Stat(filepath.Join(tempDir, sourceAFiles[2].name))
	assert.True(t, os.IsNotExist(err), "oldest source A backup should be deleted")
	_, err = os.Stat(filepath.Join(tempDir, sourceBFiles[2].name))
	assert.True(t, os.IsNotExist(err), "oldest source B backup should be deleted")
}

// TestApplyPolicy_SameSourceDifferentHosts tests retention scoping with same source
// but different hostnames — each host is a separate retention group.
func TestApplyPolicy_SameSourceDifferentHosts(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Host1: 3 backups from /data
	host1Files := []struct {
		name string
		age  time.Duration
	}{
		{"backup_h1a_20240207_120000.tar.gz.gpg", 1 * time.Hour},
		{"backup_h1b_20240207_110000.tar.gz.gpg", 2 * time.Hour},
		{"backup_h1c_20240207_100000.tar.gz.gpg", 3 * time.Hour}, // delete
	}
	for _, f := range host1Files {
		path := filepath.Join(tempDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-f.age)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
		writeTestManifest(t, path, "/data", "host1")
	}

	// Host2: 3 backups from /data
	host2Files := []struct {
		name string
		age  time.Duration
	}{
		{"backup_h2a_20240207_120000.tar.gz.gpg", 1 * time.Hour},
		{"backup_h2b_20240207_110000.tar.gz.gpg", 2 * time.Hour},
		{"backup_h2c_20240207_100000.tar.gz.gpg", 3 * time.Hour}, // delete
	}
	for _, f := range host2Files {
		path := filepath.Join(tempDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-f.age)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
		writeTestManifest(t, path, "/data", "host2")
	}

	// Keep 2 per group
	policy := Policy{
		KeepLast:  2,
		BackupDir: tempDir,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should delete 1 from each host (2 total)")

	// Verify oldest from each host deleted
	_, err = os.Stat(filepath.Join(tempDir, host1Files[2].name))
	assert.True(t, os.IsNotExist(err), "oldest host1 backup should be deleted")
	_, err = os.Stat(filepath.Join(tempDir, host2Files[2].name))
	assert.True(t, os.IsNotExist(err), "oldest host2 backup should be deleted")
}

// TestApplyPolicy_AllOrphans tests that all-orphan directories result in zero deletions
// with stderr warnings.
func TestApplyPolicy_AllOrphans(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 5 backups WITHOUT manifests (orphans)
	orphans := []string{
		"backup_orphan1_20240207_120000.tar.gz.gpg",
		"backup_orphan2_20240206_120000.tar.gz.gpg",
		"backup_orphan3_20240205_120000.tar.gz.gpg",
		"backup_orphan4_20240204_120000.tar.gz.gpg",
		"backup_orphan5_20240203_120000.tar.gz.gpg",
	}

	for i, name := range orphans {
		path := filepath.Join(tempDir, name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-time.Duration(i+1) * time.Hour)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
	}

	policy := Policy{
		KeepLast:  3,
		BackupDir: tempDir,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should delete 0 orphan backups")

	// Verify all orphans still exist
	for _, name := range orphans {
		_, err := os.Stat(filepath.Join(tempDir, name))
		assert.NoError(t, err, "orphan %s should be kept (untouched)", name)
	}
}

// TestApplyPolicy_MixedManagedAndOrphans tests that orphans are excluded from
// retention while managed backups are scoped correctly.
func TestApplyPolicy_MixedManagedAndOrphans(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 4 managed backups (same source)
	managed := []struct {
		name string
		age  time.Duration
	}{
		{"backup_mgd1_20240207_120000.tar.gz.gpg", 1 * time.Hour},      // keep
		{"backup_mgd2_20240207_110000.tar.gz.gpg", 2 * time.Hour},      // keep
		{"backup_mgd3_20240207_100000.tar.gz.gpg", 3 * time.Hour},      // delete
		{"backup_mgd4_20240206_120000.tar.gz.gpg", 1 * 24 * time.Hour}, // delete
	}
	for _, f := range managed {
		path := filepath.Join(tempDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-f.age)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
		writeTestManifest(t, path, "/data", "host1")
	}

	// Create 3 orphan backups
	orphans := []string{
		"backup_orp1_20240207_090000.tar.gz.gpg",
		"backup_orp2_20240205_120000.tar.gz.gpg",
		"backup_orp3_20240204_120000.tar.gz.gpg",
	}
	for i, name := range orphans {
		path := filepath.Join(tempDir, name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-time.Duration(i+4) * time.Hour)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
	}

	// Keep 2 — only affects managed
	policy := Policy{
		KeepLast:  2,
		BackupDir: tempDir,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should delete 2 managed backups beyond keep count")

	// Verify 2 newest managed are kept
	for _, f := range managed[:2] {
		_, err := os.Stat(filepath.Join(tempDir, f.name))
		assert.NoError(t, err, "managed file %s should be kept", f.name)
	}

	// Verify 2 oldest managed are deleted
	for _, f := range managed[2:] {
		_, err := os.Stat(filepath.Join(tempDir, f.name))
		assert.True(t, os.IsNotExist(err), "managed file %s should be deleted", f.name)
	}

	// Verify ALL orphans are untouched
	for _, name := range orphans {
		_, err := os.Stat(filepath.Join(tempDir, name))
		assert.NoError(t, err, "orphan %s should not be touched by retention", name)
	}
}

// TestApplyPolicy_OrphanNotCountedTowardN verifies that orphan backups are not
// counted toward the keep-N limit of managed groups.
func TestApplyPolicy_OrphanNotCountedTowardN(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 2 managed backups
	managed := []struct {
		name string
		age  time.Duration
	}{
		{"backup_mgd1_20240207_120000.tar.gz.gpg", 1 * time.Hour},
		{"backup_mgd2_20240207_110000.tar.gz.gpg", 2 * time.Hour},
	}
	for _, f := range managed {
		path := filepath.Join(tempDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte("backup"), 0644))
		modTime := now.Add(-f.age)
		require.NoError(t, os.Chtimes(path, modTime, modTime))
		writeTestManifest(t, path, "/data", "host1")
	}

	// Create 5 orphan backups
	orphans := make([]string, 5)
	for i := range 5 {
		name := filepath.Join(tempDir, "backup_orp"+string(rune('a'+i))+"_20240207_120000.tar.gz.gpg")
		require.NoError(t, os.WriteFile(name, []byte("backup"), 0644))
		modTime := now.Add(-time.Duration(i+3) * time.Hour)
		require.NoError(t, os.Chtimes(name, modTime, modTime))
		orphans[i] = name
	}

	// Keep 3 — could fit all 2 managed + 5 orphans if orphans were counted
	// But orphans should NOT be counted, so 2 managed < 3, nothing deleted
	policy := Policy{
		KeepLast:  3,
		BackupDir: tempDir,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should delete 0 — only 2 managed backups, keep 3")

	// Verify all managed backups still exist
	for _, f := range managed {
		_, err := os.Stat(filepath.Join(tempDir, f.name))
		assert.NoError(t, err, "managed %s should be kept", f.name)
	}
}

// TestApplyPolicy_CorruptManifest tests that a backup with a corrupt manifest is
// treated as an orphan.
func TestApplyPolicy_CorruptManifest(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 1 backup with valid manifest
	validPath := filepath.Join(tempDir, "backup_valid_20240207_120000.tar.gz.gpg")
	require.NoError(t, os.WriteFile(validPath, []byte("backup"), 0644))
	modTime := now.Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(validPath, modTime, modTime))
	writeTestManifest(t, validPath, "/data", "host1")

	// Create 1 backup with corrupt manifest (invalid JSON)
	corruptPath := filepath.Join(tempDir, "backup_corrupt_20240206_120000.tar.gz.gpg")
	require.NoError(t, os.WriteFile(corruptPath, []byte("backup"), 0644))
	modTime = now.Add(-24 * time.Hour)
	require.NoError(t, os.Chtimes(corruptPath, modTime, modTime))
	corruptManifest := manifest.ManifestPath(corruptPath)
	require.NoError(t, os.WriteFile(corruptManifest, []byte("NOT VALID JSON{{{"), 0644))

	// Keep 1 — only 1 managed backup exists, corrupt treated as orphan
	policy := Policy{
		KeepLast:  1,
		BackupDir: tempDir,
	}

	count, err := ApplyPolicy(policy)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should delete 0 — only 1 managed, 1 orphan (corrupt)")

	// Verify both files still exist
	_, err = os.Stat(validPath)
	assert.NoError(t, err, "valid backup should be kept")
	_, err = os.Stat(corruptPath)
	assert.NoError(t, err, "corrupt (orphan) backup should be untouched")
}
