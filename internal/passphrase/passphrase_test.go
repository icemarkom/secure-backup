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

package passphrase

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_FlagPriority(t *testing.T) {
	// When flag is provided, it takes precedence
	got, err := Get("flag-secret", "", "")
	require.NoError(t, err)
	assert.Equal(t, "flag-secret", got)
}

func TestGet_EnvFallback(t *testing.T) {
	// Set up environment variable
	envName := "TEST_PASSPHRASE"
	os.Setenv(envName, "env-secret")
	defer os.Unsetenv(envName)

	// When flag is empty, env var is used
	got, err := Get("", envName, "")
	require.NoError(t, err)
	assert.Equal(t, "env-secret", got)
}

func TestGet_FileFallback(t *testing.T) {
	// Create temp file with passphrase
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")
	err := os.WriteFile(filePath, []byte("file-secret\n"), 0600)
	require.NoError(t, err)

	// When flag and env are empty, file is used
	got, err := Get("", "", filePath)
	require.NoError(t, err)
	assert.Equal(t, "file-secret", got)
}

func TestGet_EmptyAllowed(t *testing.T) {
	// When all sources are empty, empty string is returned (no error)
	got, err := Get("", "", "")
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestGet_MutuallyExclusive_FlagAndEnv(t *testing.T) {
	// Set up environment variable
	envName := "TEST_PASSPHRASE"
	os.Setenv(envName, "env-secret")
	defer os.Unsetenv(envName)

	// Error when both flag and env are set
	_, err := Get("flag-secret", envName, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple passphrase sources")
	assert.Contains(t, err.Error(), "--passphrase flag")
	assert.Contains(t, err.Error(), "TEST_PASSPHRASE environment variable")
}

func TestGet_MutuallyExclusive_FlagAndFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")
	err := os.WriteFile(filePath, []byte("file-secret"), 0600)
	require.NoError(t, err)

	// Error when both flag and file are set
	_, err = Get("flag-secret", "", filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple passphrase sources")
	assert.Contains(t, err.Error(), "--passphrase flag")
	assert.Contains(t, err.Error(), "--passphrase-file flag")
}

func TestGet_MutuallyExclusive_EnvAndFile(t *testing.T) {
	// Set up environment variable
	envName := "TEST_PASSPHRASE"
	os.Setenv(envName, "env-secret")
	defer os.Unsetenv(envName)

	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")
	err := os.WriteFile(filePath, []byte("file-secret"), 0600)
	require.NoError(t, err)

	// Error when both env and file are set
	_, err = Get("", envName, filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple passphrase sources")
	assert.Contains(t, err.Error(), "TEST_PASSPHRASE environment variable")
	assert.Contains(t, err.Error(), "--passphrase-file flag")
}

func TestGet_MutuallyExclusive_All(t *testing.T) {
	// Set up environment variable
	envName := "TEST_PASSPHRASE"
	os.Setenv(envName, "env-secret")
	defer os.Unsetenv(envName)

	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")
	err := os.WriteFile(filePath, []byte("file-secret"), 0600)
	require.NoError(t, err)

	// Error when all three are set
	_, err = Get("flag-secret", envName, filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple passphrase sources")
}

func TestGet_SecurityWarning(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Use flag value
	_, err := Get("flag-secret", "", "")
	require.NoError(t, err)

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify warning was printed
	output := buf.String()
	assert.Contains(t, output, "WARNING")
	assert.Contains(t, output, "insecure")
	assert.Contains(t, output, "SECURE_BACKUP_PASSPHRASE")
	assert.Contains(t, output, "--passphrase-file")
}

func TestGet_EnvEmpty(t *testing.T) {
	// Set up environment variable with empty value
	envName := "TEST_PASSPHRASE"
	os.Setenv(envName, "")
	defer os.Unsetenv(envName)

	// Empty env var should be treated as not provided
	got, err := Get("", envName, "")
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestReadFromFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")
	err := os.WriteFile(filePath, []byte("my-secret-passphrase"), 0600)
	require.NoError(t, err)

	got, err := readFromFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "my-secret-passphrase", got)
}

func TestReadFromFile_TrimWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")

	// Test with various whitespace combinations
	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{"trailing newline", "secret\n", "secret"},
		{"leading spaces", "  secret", "secret"},
		{"trailing spaces", "secret  ", "secret"},
		{"both", "  secret  \n", "secret"},
		{"multiple newlines", "secret\n\n\n", "secret"},
		{"tabs", "\tsecret\t", "secret"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := os.WriteFile(filePath, []byte(tc.content), 0600)
			require.NoError(t, err)

			got, err := readFromFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestReadFromFile_NotFound(t *testing.T) {
	_, err := readFromFile("/nonexistent/path/passphrase.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReadFromFile_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")
	err := os.WriteFile(filePath, []byte(""), 0600)
	require.NoError(t, err)

	_, err = readFromFile(filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestReadFromFile_OnlyWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")
	err := os.WriteFile(filePath, []byte("  \n\t\n  "), 0600)
	require.NoError(t, err)

	_, err = readFromFile(filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestReadFromFile_PermissionWarning(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")

	// Create file with world-readable permissions
	err := os.WriteFile(filePath, []byte("secret"), 0644)
	require.NoError(t, err)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Read file
	got, err := readFromFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "secret", got)

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify warning was printed
	output := buf.String()
	assert.Contains(t, output, "WARNING")
	assert.Contains(t, output, "world-readable")
	assert.Contains(t, output, "chmod 600")
}

func TestReadFromFile_CorrectPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "passphrase.txt")

	// Create file with correct permissions
	err := os.WriteFile(filePath, []byte("secret"), 0600)
	require.NoError(t, err)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Read file
	got, err := readFromFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "secret", got)

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify NO warning was printed
	output := buf.String()
	assert.NotContains(t, strings.ToLower(output), "warning")
}
