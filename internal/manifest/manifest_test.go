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

package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestPath(t *testing.T) {
	tests := []struct {
		name       string
		backupPath string
		want       string
	}{
		{
			name:       "gpg encrypted gzip",
			backupPath: "backup_data_20260215_120000.tar.gz.gpg",
			want:       "backup_data_20260215_120000_manifest.json",
		},
		{
			name:       "unimplemented compression fallback",
			backupPath: "backup_data_20260215_120000.tar.zst.gpg",
			want:       "backup_data_20260215_120000.tar.zst.gpg_manifest.json",
		},
		{
			name:       "age encrypted gzip",
			backupPath: "backup_data_20260215_120000.tar.gz.age",
			want:       "backup_data_20260215_120000_manifest.json",
		},
		{
			name:       "full path",
			backupPath: "/backups/daily/backup_data_20260215_120000.tar.gz.gpg",
			want:       "/backups/daily/backup_data_20260215_120000_manifest.json",
		},
		{
			name:       "gpg encrypted no compression",
			backupPath: "backup_data_20260215_120000.tar.gpg",
			want:       "backup_data_20260215_120000_manifest.json",
		},
		{
			name:       "age encrypted no compression",
			backupPath: "backup_data_20260215_120000.tar.age",
			want:       "backup_data_20260215_120000_manifest.json",
		},
		{
			name:       "unknown extension fallback",
			backupPath: "backup_data_20260215_120000.unknown",
			want:       "backup_data_20260215_120000.unknown_manifest.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ManifestPath(tt.backupPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNew(t *testing.T) {
	m, err := New("/path/to/source", "backup_test_20260214.tar.gz.gpg", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)
	assert.NotNil(t, m)

	// Verify all fields are populated
	assert.Equal(t, "/path/to/source", m.SourcePath)
	assert.Equal(t, "backup_test_20260214.tar.gz.gpg", m.BackupFile)
	assert.Equal(t, "gzip", m.Compression)
	assert.Equal(t, "gpg", m.Encryption)
	assert.Equal(t, "sha256", m.ChecksumAlgorithm)
	assert.Equal(t, "secure-backup", m.CreatedBy.Tool)
	assert.Equal(t, "v1.0.0", m.CreatedBy.Version)
	assert.NotEmpty(t, m.CreatedBy.Hostname) // Should be set or "unknown"
	assert.WithinDuration(t, time.Now().UTC(), m.CreatedAt, 2*time.Second)
}

func TestWrite(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.json")

	m, err := New("/test/source", "backup.tar.gz.gpg", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)
	m.ChecksumValue = "abc123"
	m.SizeBytes = 1024

	err = m.Write(manifestPath, nil)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(manifestPath)
	require.NoError(t, err)

	// Verify file is readable JSON
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"source_path\": \"/test/source\"")
	assert.Contains(t, string(data), "\"checksum_value\": \"abc123\"")
}

func TestRead(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.json")

	// Create a manifest
	original, err := New("/test/source", "backup.tar.gz.gpg", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)
	original.ChecksumValue = "abc123"
	original.SizeBytes = 2048

	err = original.Write(manifestPath, nil)
	require.NoError(t, err)

	// Read it back
	m, err := Read(manifestPath)
	require.NoError(t, err)
	assert.NotNil(t, m)

	// Verify fields match
	assert.Equal(t, original.SourcePath, m.SourcePath)
	assert.Equal(t, original.BackupFile, m.BackupFile)
	assert.Equal(t, original.ChecksumValue, m.ChecksumValue)
	assert.Equal(t, original.SizeBytes, m.SizeBytes)
	assert.Equal(t, original.CreatedBy.Tool, m.CreatedBy.Tool)
	assert.Equal(t, original.CreatedBy.Version, m.CreatedBy.Version)
}

func TestReadWrite_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "roundtrip.json")

	// Create manifest
	m1, err := New("/source/path", "backup_file.tar.gz.gpg", "v2.0.0", "gzip", "gpg")
	require.NoError(t, err)
	m1.ChecksumValue = "def456"
	m1.SizeBytes = 4096

	// Write
	err = m1.Write(manifestPath, nil)
	require.NoError(t, err)

	// Read
	m2, err := Read(manifestPath)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, m1.SourcePath, m2.SourcePath)
	assert.Equal(t, m1.BackupFile, m2.BackupFile)
	assert.Equal(t, m1.Compression, m2.Compression)
	assert.Equal(t, m1.Encryption, m2.Encryption)
	assert.Equal(t, m1.ChecksumAlgorithm, m2.ChecksumAlgorithm)
	assert.Equal(t, m1.ChecksumValue, m2.ChecksumValue)
	assert.Equal(t, m1.SizeBytes, m2.SizeBytes)
	assert.Equal(t, m1.CreatedBy.Tool, m2.CreatedBy.Tool)
	assert.Equal(t, m1.CreatedBy.Version, m2.CreatedBy.Version)
	assert.Equal(t, m1.CreatedBy.Hostname, m2.CreatedBy.Hostname)
}

func TestValidate_Valid(t *testing.T) {
	m, err := New("/source", "backup.tar.gz.gpg", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)
	m.ChecksumValue = "abc123"

	err = m.Validate()
	assert.NoError(t, err)
}

func TestValidate_MissingFields(t *testing.T) {
	tests := []struct {
		name        string
		modifyFunc  func(*Manifest)
		expectedErr string
	}{
		{
			name:        "missing source_path",
			modifyFunc:  func(m *Manifest) { m.SourcePath = "" },
			expectedErr: "missing source_path",
		},
		{
			name:        "missing backup_file",
			modifyFunc:  func(m *Manifest) { m.BackupFile = "" },
			expectedErr: "missing backup_file",
		},
		{
			name:        "missing checksum_value",
			modifyFunc:  func(m *Manifest) { m.ChecksumValue = "" },
			expectedErr: "missing checksum_value",
		},
		{
			name:        "missing checksum_algorithm",
			modifyFunc:  func(m *Manifest) { m.ChecksumAlgorithm = "" },
			expectedErr: "missing checksum_algorithm",
		},
		{
			name:        "missing created_by.tool",
			modifyFunc:  func(m *Manifest) { m.CreatedBy.Tool = "" },
			expectedErr: "missing created_by.tool",
		},
		{
			name:        "missing created_by.version",
			modifyFunc:  func(m *Manifest) { m.CreatedBy.Version = "" },
			expectedErr: "missing created_by.version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := New("/source", "backup.tar.gz.gpg", "v1.0.0", "gzip", "gpg")
			require.NoError(t, err)
			m.ChecksumValue = "abc123"

			tt.modifyFunc(m)

			err = m.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestComputeChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file with known content
	content := "test data for checksum"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Compute checksum
	checksum, err := ComputeChecksum(testFile)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64) // SHA256 produces 64 hex characters

	// Verify checksum is deterministic
	checksum2, err := ComputeChecksum(testFile)
	require.NoError(t, err)
	assert.Equal(t, checksum, checksum2)
}

func TestComputeChecksum_NonexistentFile(t *testing.T) {
	checksum, err := ComputeChecksum("/nonexistent/file.txt")
	assert.Error(t, err)
	assert.Empty(t, checksum)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestComputeChecksum_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.bin")

	// Create a larger file (1MB)
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	err := os.WriteFile(testFile, data, 0644)
	require.NoError(t, err)

	// Should handle large files via streaming
	checksum, err := ComputeChecksum(testFile)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64)
}

func TestValidateChecksum_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Compute checksum
	checksum, err := ComputeChecksum(testFile)
	require.NoError(t, err)

	// Create manifest with correct checksum
	m, err := New("/source", "test.txt", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)
	m.ChecksumValue = checksum

	// Validate should pass
	err = m.ValidateChecksum(testFile)
	assert.NoError(t, err)
}

func TestValidateChecksum_Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	err := os.WriteFile(testFile, []byte("original content"), 0644)
	require.NoError(t, err)

	// Compute checksum
	checksum, err := ComputeChecksum(testFile)
	require.NoError(t, err)

	// Create manifest
	m, err := New("/source", "test.txt", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)
	m.ChecksumValue = checksum

	// Modify the file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	// Validation should fail
	err = m.ValidateChecksum(testFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestValidateChecksum_FileNotFound(t *testing.T) {
	m, err := New("/source", "backup.tar.gz.gpg", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)
	m.ChecksumValue = "abc123"

	err = m.ValidateChecksum("/nonexistent/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compute file checksum")
}

func TestRead_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	err := os.WriteFile(manifestPath, []byte("not valid json {{{"), 0644)
	require.NoError(t, err)

	// Read should fail
	m, err := Read(manifestPath)
	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestRead_NonexistentFile(t *testing.T) {
	m, err := Read("/nonexistent/manifest.json")
	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Contains(t, err.Error(), "failed to read manifest file")
}

func TestWrite_InvalidPath(t *testing.T) {
	m, err := New("/source", "backup.tar.gz.gpg", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)

	// Try to write to invalid path (directory doesn't exist and can't be created)
	err = m.Write("/root/nonexistent/dir/manifest.json", nil)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "failed to write")
}

// TestWrite_NoTempFilesOnSuccess tests that no .tmp files remain after successful write
func TestWrite_NoTempFilesOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "atomic-test.json")

	// Create and write manifest
	m, err := New("/test/source", "backup.tar.gz.gpg", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)
	m.ChecksumValue = "test123"
	m.SizeBytes = 1024

	err = m.Write(manifestPath, nil)
	require.NoError(t, err)

	// Verify final manifest file exists
	_, err = os.Stat(manifestPath)
	assert.NoError(t, err, "final manifest file should exist")

	// Verify no .tmp file exists
	tmpPath := manifestPath + ".tmp"
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), ".tmp file should not exist after successful write")

	// Verify no .tmp files in directory
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	for _, file := range files {
		assert.False(t, strings.HasSuffix(file.Name(), ".tmp"),
			"no .tmp files should remain in directory: found %s", file.Name())
	}
}

// TestWrite_FilePermissions tests that manifest files get the specified file permissions
func TestWrite_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "permissions-test.json")

	m, err := New("/test/source", "backup.tar.gz.gpg", "v1.0.0", "gzip", "gpg")
	require.NoError(t, err)
	m.ChecksumValue = "abc123"
	m.SizeBytes = 1024

	mode := os.FileMode(0600)
	err = m.Write(manifestPath, &mode)
	require.NoError(t, err)

	// Verify file permissions
	info, err := os.Stat(manifestPath)
	require.NoError(t, err)
	got := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0600), got, "manifest file should have 0600 permissions, got %04o", got)
}
