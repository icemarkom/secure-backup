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

func TestLz4Compressor_CompressDecompress(t *testing.T) {
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
			level: 3,
		},
		{
			name:  "binary-like data",
			data:  string([]byte{0, 1, 2, 3, 255, 254, 253}),
			level: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create compressor
			compressor, err := NewLz4Compressor(tt.level)
			require.NoError(t, err)
			assert.Equal(t, ".lz4", compressor.Extension())

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

func TestLz4Compressor_InvalidLevel(t *testing.T) {
	tests := []struct {
		name  string
		level int
	}{
		{"negative", -1},
		{"too high", 10},
		{"way too high", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLz4Compressor(tt.level)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid lz4 compression level")
		})
	}
}

func TestLz4Compressor_ValidLevels(t *testing.T) {
	validLevels := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	for _, level := range validLevels {
		compressor, err := NewLz4Compressor(level)
		require.NoError(t, err)
		assert.NotNil(t, compressor)
	}
}

func TestLz4Compressor_CompressionRatio(t *testing.T) {
	// Test that compression actually reduces size for repetitive data
	data := strings.Repeat("AAAA", 10000) // Highly compressible

	compressor, err := NewLz4Compressor(0)
	require.NoError(t, err)

	compressed, err := compressor.Compress(strings.NewReader(data))
	require.NoError(t, err)

	compressedData, err := io.ReadAll(compressed)
	require.NoError(t, err)

	// Should compress to much less than original size
	assert.Less(t, len(compressedData), len(data)/10, "compression ratio should be significant for repetitive data")
}

func TestLz4Compressor_InvalidData(t *testing.T) {
	compressor, err := NewLz4Compressor(0)
	require.NoError(t, err)

	// Try to decompress invalid lz4 data — NewReader succeeds, error comes on read
	reader, err := compressor.Decompress(bytes.NewReader([]byte("this is not lz4 data")))
	if err != nil {
		// Some implementations error on NewReader
		return
	}
	_, err = io.ReadAll(reader)
	assert.Error(t, err)
}

func TestLz4Compressor_Type(t *testing.T) {
	compressor, err := NewLz4Compressor(0)
	require.NoError(t, err)
	assert.Equal(t, Lz4, compressor.Type())
}

func TestNewCompressor_Lz4(t *testing.T) {
	compressor, err := NewCompressor(Config{Method: Lz4, Level: 0})
	require.NoError(t, err)
	require.NotNil(t, compressor)
	assert.Equal(t, Lz4, compressor.Type())
	assert.Equal(t, ".lz4", compressor.Extension())
}
