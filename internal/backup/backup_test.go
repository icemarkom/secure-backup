package backup

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

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
