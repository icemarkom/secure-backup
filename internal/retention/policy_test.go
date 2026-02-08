package retention

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "less than 1 hour",
			duration: 30 * time.Minute,
			expected: "30m",
		},
		{
			name:     "exactly 1 hour",
			duration: 1 * time.Hour,
			expected: "1h",
		},
		{
			name:     "hours only",
			duration: 12 * time.Hour,
			expected: "12h",
		},
		{
			name:     "1 day",
			duration: 24 * time.Hour,
			expected: "1d0h",
		},
		{
			name:     "multiple days",
			duration: 5*24*time.Hour + 3*time.Hour,
			expected: "5d3h",
		},
		{
			name:     "30 days",
			duration: 30 * 24 * time.Hour,
			expected: "30d0h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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
			bytes:    1024,
			expected: "1.0 KiB",
		},
		{
			name:     "megabytes",
			bytes:    1024 * 1024,
			expected: "1.0 MiB",
		},
		{
			name:     "gigabytes",
			bytes:    5 * 1024 * 1024 * 1024,
			expected: "5.0 GiB",
		},
		{
			name:     "terabytes",
			bytes:    2 * 1024 * 1024 * 1024 * 1024,
			expected: "2.0 TiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBackupFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "valid backup file",
			filename: "backup_20240207_183000.tar.gz.gpg",
			expected: true,
		},
		{
			name:     "tar.gz file only",
			filename: "backup_20240207_183000.tar.gz",
			expected: false,
		},
		{
			name:     "random file",
			filename: "random.txt",
			expected: false,
		},
		{
			name:     "partial match",
			filename: "mybackup.tar.gz.gpg",
			expected: false,
		},
		{
			name:     "wrong extension",
			filename: "backup_20240207_183000.tar.gpg",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBackupFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}
