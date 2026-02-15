package format

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSize(t *testing.T) {
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
			bytes: 1500,
			want:  "1.5 KiB",
		},
		{
			name:  "megabytes",
			bytes: 2 * 1024 * 1024,
			want:  "2.0 MiB",
		},
		{
			name:  "gigabytes",
			bytes: 3 * 1024 * 1024 * 1024,
			want:  "3.0 GiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Size(tt.bytes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAge(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{
			name: "minutes only",
			d:    30 * time.Minute,
			want: "30m",
		},
		{
			name: "zero duration",
			d:    0,
			want: "0m",
		},
		{
			name: "hours only",
			d:    5 * time.Hour,
			want: "5h",
		},
		{
			name: "days and hours",
			d:    3*24*time.Hour + 12*time.Hour,
			want: "3d12h",
		},
		{
			name: "days with zero hours",
			d:    7 * 24 * time.Hour,
			want: "7d0h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Age(tt.d)
			assert.Equal(t, tt.want, got)
		})
	}
}
