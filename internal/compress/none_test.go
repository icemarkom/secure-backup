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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoneCompressor_Type(t *testing.T) {
	compressor := NewNoneCompressor()
	assert.Equal(t, None, compressor.Type())
}

func TestNoneCompressor_Extension(t *testing.T) {
	compressor := NewNoneCompressor()
	assert.Equal(t, "", compressor.Extension())
}

func TestNoneCompressor_RoundTrip(t *testing.T) {
	compressor := NewNoneCompressor()
	input := []byte("hello world - this data should pass through unchanged")

	// Compress (passthrough)
	compressed, err := compressor.Compress(bytes.NewReader(input))
	require.NoError(t, err)

	compressedData, err := io.ReadAll(compressed)
	require.NoError(t, err)
	assert.Equal(t, input, compressedData, "compressed data should equal input (passthrough)")

	// Decompress (passthrough)
	decompressed, err := compressor.Decompress(bytes.NewReader(compressedData))
	require.NoError(t, err)

	decompressedData, err := io.ReadAll(decompressed)
	require.NoError(t, err)
	assert.Equal(t, input, decompressedData, "decompressed data should equal input (passthrough)")
}

func TestNoneCompressor_LargeData(t *testing.T) {
	compressor := NewNoneCompressor()
	input := bytes.Repeat([]byte("large block of data "), 10000)

	compressed, err := compressor.Compress(bytes.NewReader(input))
	require.NoError(t, err)

	result, err := io.ReadAll(compressed)
	require.NoError(t, err)
	assert.Equal(t, len(input), len(result), "passthrough should not change data size")
	assert.Equal(t, input, result)
}

func TestNoneCompressor_EmptyInput(t *testing.T) {
	compressor := NewNoneCompressor()

	compressed, err := compressor.Compress(bytes.NewReader(nil))
	require.NoError(t, err)

	result, err := io.ReadAll(compressed)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestNewCompressor_None(t *testing.T) {
	compressor, err := NewCompressor(Config{Method: None})
	require.NoError(t, err)
	require.NotNil(t, compressor)
	assert.Equal(t, None, compressor.Type())
	assert.Equal(t, "", compressor.Extension())
}
