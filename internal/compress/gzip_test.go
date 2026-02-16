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

package compress

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipCompressor_CompressDecompress(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		level int
	}{
		{
			name:  "simple text",
			data:  "Hello, World!",
			level: 0, // default
		},
		{
			name:  "empty string",
			data:  "",
			level: 0,
		},
		{
			name:  "large text",
			data:  strings.Repeat("The quick brown fox jumps over the lazy dog. ", 1000),
			level: 6,
		},
		{
			name:  "binary-like data",
			data:  string([]byte{0, 1, 2, 3, 255, 254, 253}),
			level: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create compressor
			compressor, err := NewGzipCompressor(tt.level)
			require.NoError(t, err)
			assert.Equal(t, ".gz", compressor.Extension())

			// Compress
			input := bytes.NewReader([]byte(tt.data))
			compressed, err := compressor.Compress(input)
			require.NoError(t, err)

			// Read compressed data
			compressedData, err := io.ReadAll(compressed)
			require.NoError(t, err)

			// Decompress
			decompressed, err := compressor.Decompress(bytes.NewReader(compressedData))
			require.NoError(t, err)

			// Read decompressed data
			decompressedData, err := io.ReadAll(decompressed)
			require.NoError(t, err)

			// Verify round-trip
			assert.Equal(t, tt.data, string(decompressedData))
		})
	}
}

func TestGzipCompressor_InvalidLevel(t *testing.T) {
	tests := []struct {
		name  string
		level int
	}{
		{"too low", -5},
		{"too high", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGzipCompressor(tt.level)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid gzip compression level")
		})
	}
}

func TestGzipCompressor_ValidLevels(t *testing.T) {
	// Test all valid compression levels
	validLevels := []int{-1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	for _, level := range validLevels {
		t.Run(string(rune(level)), func(t *testing.T) {
			compressor, err := NewGzipCompressor(level)
			require.NoError(t, err)
			assert.NotNil(t, compressor)
		})
	}
}

func TestGzipCompressor_CompressionRatio(t *testing.T) {
	// Test that compression actually reduces size for repetitive data
	data := strings.Repeat("AAAA", 10000) // Highly compressible

	compressor, err := NewGzipCompressor(6)
	require.NoError(t, err)

	compressed, err := compressor.Compress(strings.NewReader(data))
	require.NoError(t, err)

	compressedData, err := io.ReadAll(compressed)
	require.NoError(t, err)

	// Should compress to much less than original size
	assert.Less(t, len(compressedData), len(data)/10, "compression ratio should be significant for repetitive data")
}

func TestNewCompressor(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "default (empty method)",
			config:      Config{Method: "", Level: 0},
			expectError: false,
		},
		{
			name:        "explicit gzip",
			config:      Config{Method: "gzip", Level: 6},
			expectError: false,
		},
		{
			name:        "zstd not implemented",
			config:      Config{Method: "zstd", Level: 3},
			expectError: true,
			errorMsg:    "not yet implemented",
		},
		{
			name:        "unknown method",
			config:      Config{Method: "bzip2", Level: 9},
			expectError: true,
			errorMsg:    "unknown compression method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor, err := NewCompressor(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, compressor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, compressor)
			}
		})
	}
}

func TestGzipCompressor_InvalidData(t *testing.T) {
	compressor, err := NewGzipCompressor(0)
	require.NoError(t, err)

	// Try to decompress invalid gzip data
	invalidData := []byte("this is not gzip data")
	_, err = compressor.Decompress(bytes.NewReader(invalidData))
	assert.Error(t, err)
}
