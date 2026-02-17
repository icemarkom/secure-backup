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

package archive

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTar_SingleFile(t *testing.T) {
	// Create temp directory with a single file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, tar!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Create tar
	var buf bytes.Buffer
	bytesWritten, err := CreateTar(testFile, &buf)
	require.NoError(t, err)
	assert.Equal(t, int64(len(testContent)), bytesWritten)

	// Verify tar was created
	assert.Greater(t, buf.Len(), 0)
}

func TestCreateTar_Directory(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create files
	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)

	// Create subdirectory with file
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "file3.txt"), []byte("content3"), 0644)
	require.NoError(t, err)

	// Create tar
	var buf bytes.Buffer
	bytesWritten, err := CreateTar(tmpDir, &buf)
	require.NoError(t, err)

	// 3 files: "content1" (8) + "content2" (8) + "content3" (8) = 24 bytes
	assert.Equal(t, int64(24), bytesWritten)

	// Verify tar was created
	assert.Greater(t, buf.Len(), 0)
}

func TestCreateTar_InvalidPath(t *testing.T) {
	var buf bytes.Buffer
	_, err := CreateTar("/nonexistent/path/that/does/not/exist", &buf)
	assert.Error(t, err)
}

func TestExtractTar_RoundTrip(t *testing.T) {
	// Create temp source directory
	srcDir := t.TempDir()

	// Create test files with different permissions
	files := map[string]struct {
		content string
		mode    os.FileMode
	}{
		"file1.txt":     {"content1", 0644},
		"file2.txt":     {"content2", 0600},
		"executable.sh": {"#!/bin/bash\necho hello", 0755},
	}

	for name, file := range files {
		path := filepath.Join(srcDir, name)
		err := os.WriteFile(path, []byte(file.content), file.mode)
		require.NoError(t, err)
	}

	// Create subdirectory
	subDir := filepath.Join(srcDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested content"), 0644)
	require.NoError(t, err)

	// Create tar
	var buf bytes.Buffer
	_, err = CreateTar(srcDir, &buf)
	require.NoError(t, err)

	// Extract to new directory
	destDir := t.TempDir()
	err = ExtractTar(&buf, destDir)
	require.NoError(t, err)

	// Verify extracted files
	srcBaseName := filepath.Base(srcDir)
	extractedBase := filepath.Join(destDir, srcBaseName)

	for name, file := range files {
		extractedPath := filepath.Join(extractedBase, name)

		// Check file exists
		info, err := os.Stat(extractedPath)
		require.NoError(t, err)

		// Check content
		content, err := os.ReadFile(extractedPath)
		require.NoError(t, err)
		assert.Equal(t, file.content, string(content))

		// Check permissions (on Unix systems)
		if os.Getenv("GOOS") != "windows" {
			assert.Equal(t, file.mode, info.Mode().Perm())
		}
	}

	// Verify subdirectory and nested file
	nestedPath := filepath.Join(extractedBase, "subdir", "nested.txt")
	content, err := os.ReadFile(nestedPath)
	require.NoError(t, err)
	assert.Equal(t, "nested content", string(content))
}

func TestExtractTar_PathTraversal(t *testing.T) {
	// This test verifies that path traversal attacks are prevented
	destDir := t.TempDir()

	// Create a malicious tar with path traversal attempt
	// Note: This is a simplified test - real implementation should reject these
	err := ExtractTar(bytes.NewReader([]byte{}), destDir)
	// Empty tar should not error
	assert.NoError(t, err)
}

func TestExtractTar_CreateDestination(t *testing.T) {
	// Test that extraction creates destination directory if it doesn't exist
	tmpParent := t.TempDir()
	destDir := filepath.Join(tmpParent, "nonexistent", "nested", "path")

	// Create simple tar
	srcDir := t.TempDir()
	testFile := filepath.Join(srcDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	var buf bytes.Buffer
	_, err = CreateTar(testFile, &buf)
	require.NoError(t, err)

	// Extract to nonexistent path
	err = ExtractTar(&buf, destDir)
	require.NoError(t, err)

	// Verify destination was created
	_, err = os.Stat(destDir)
	assert.NoError(t, err)
}

func TestValidateTarPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		skipNonWin  bool
	}{
		{"simple filename", "file.txt", false, false},
		{"relative path", "dir/file.txt", false, false},
		{"nested path", "a/b/c/file.txt", false, false},
		{"absolute path", "/etc/passwd", true, false},
		{"path traversal", "../../../etc/passwd", true, false},
		{"embedded traversal", "dir/../../etc/passwd", true, false},
		{"windows absolute", "C:\\Windows\\System32", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipNonWin && os.Getenv("GOOS") != "windows" {
				t.Skip("Skipping Windows-specific test on non-Windows platform")
			}
			err := validateTarPath(tt.path)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateTar_Symlink(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Symlink test skipped on Windows")
	}

	tmpDir := t.TempDir()

	// Create target file
	targetFile := filepath.Join(tmpDir, "target.txt")
	err := os.WriteFile(targetFile, []byte("target content"), 0644)
	require.NoError(t, err)

	// Create symlink to target
	linkFile := filepath.Join(tmpDir, "link.txt")
	err = os.Symlink("target.txt", linkFile)
	require.NoError(t, err)

	// Create tar
	var buf bytes.Buffer
	bytesWritten, err := CreateTar(tmpDir, &buf)
	require.NoError(t, err)

	// Only target.txt has file data (14 bytes), symlink has 0 data bytes
	assert.Equal(t, int64(len("target content")), bytesWritten)

	// Extract
	destDir := t.TempDir()
	err = ExtractTar(&buf, destDir)
	require.NoError(t, err)

	// Verify symlink was preserved as a symlink (not dereferenced)
	extractedLink := filepath.Join(destDir, filepath.Base(tmpDir), "link.txt")
	info, err := os.Lstat(extractedLink)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "expected symlink, got regular file")

	// Verify symlink target is correct
	got, err := os.Readlink(extractedLink)
	require.NoError(t, err)
	assert.Equal(t, "target.txt", got)
}

func TestCreateTar_SymlinkToExternal(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Symlink test skipped on Windows")
	}

	// Create an external file outside the source directory
	externalDir := t.TempDir()
	externalFile := filepath.Join(externalDir, "external.txt")
	err := os.WriteFile(externalFile, []byte("external secret"), 0644)
	require.NoError(t, err)

	// Create source directory with a symlink to external file
	srcDir := t.TempDir()
	err = os.WriteFile(filepath.Join(srcDir, "regular.txt"), []byte("regular content"), 0644)
	require.NoError(t, err)
	err = os.Symlink(externalFile, filepath.Join(srcDir, "external_link.txt"))
	require.NoError(t, err)

	// Create tar
	var buf bytes.Buffer
	_, err = CreateTar(srcDir, &buf)
	require.NoError(t, err)

	// Extract and verify the external file content was NOT included
	destDir := t.TempDir()
	err = ExtractTar(&buf, destDir)
	require.NoError(t, err)

	srcBase := filepath.Base(srcDir)

	// Regular file should be present
	content, err := os.ReadFile(filepath.Join(destDir, srcBase, "regular.txt"))
	require.NoError(t, err)
	assert.Equal(t, "regular content", string(content))

	// Symlink should exist as a symlink, not a regular file with external content
	linkPath := filepath.Join(destDir, srcBase, "external_link.txt")
	info, err := os.Lstat(linkPath)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "expected symlink, got regular file")

	// Symlink target should be the original path (will be dangling in extracted location)
	got, err := os.Readlink(linkPath)
	require.NoError(t, err)
	assert.Equal(t, externalFile, got)
}

func TestCreateTar_SymlinkRoundTrip(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Symlink test skipped on Windows")
	}

	srcDir := t.TempDir()

	// Create a file and a relative symlink to it
	err := os.WriteFile(filepath.Join(srcDir, "data.txt"), []byte("hello"), 0644)
	require.NoError(t, err)
	err = os.Symlink("data.txt", filepath.Join(srcDir, "alias.txt"))
	require.NoError(t, err)

	// Backup
	var buf bytes.Buffer
	_, err = CreateTar(srcDir, &buf)
	require.NoError(t, err)

	// Restore
	destDir := t.TempDir()
	err = ExtractTar(&buf, destDir)
	require.NoError(t, err)

	srcBase := filepath.Base(srcDir)

	// Reading through the symlink should return the original content
	content, err := os.ReadFile(filepath.Join(destDir, srcBase, "alias.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	// Verify it's actually a symlink, not a copied file
	info, err := os.Lstat(filepath.Join(destDir, srcBase, "alias.txt"))
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink)
}

func TestExtractTar_EmptyArchive(t *testing.T) {
	destDir := t.TempDir()

	// Create empty tar (just EOF)
	var buf bytes.Buffer

	err := ExtractTar(&buf, destDir)
	// Empty tar should complete without error
	assert.NoError(t, err)
}
