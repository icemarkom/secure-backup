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
	"fmt"
	"io"

	"github.com/icemarkom/secure-backup/internal/common"
	"github.com/klauspost/compress/zstd"
)

// ZstdCompressor implements the Compressor interface using zstd.
type ZstdCompressor struct {
	level zstd.EncoderLevel
}

// NewZstdCompressor creates a new zstd compressor with the specified level.
// If level is 0, uses zstd.SpeedDefault.
// Valid levels: 1 (fastest), 2 (default), 3 (better compression), 4 (best compression).
func NewZstdCompressor(level int) (*ZstdCompressor, error) {
	var encLevel zstd.EncoderLevel
	switch level {
	case 0, 2:
		encLevel = zstd.SpeedDefault
	case 1:
		encLevel = zstd.SpeedFastest
	case 3:
		encLevel = zstd.SpeedBetterCompression
	case 4:
		encLevel = zstd.SpeedBestCompression
	default:
		return nil, fmt.Errorf("invalid zstd compression level: %d (must be 0-4)", level)
	}

	return &ZstdCompressor{level: encLevel}, nil
}

// Compress compresses the input stream using zstd.
func (c *ZstdCompressor) Compress(input io.Reader) (io.Reader, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		enc, err := zstd.NewWriter(pw, zstd.WithEncoderLevel(c.level))
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to create zstd writer: %w", err))
			return
		}
		defer enc.Close()

		if _, err := io.CopyBuffer(enc, input, common.NewBuffer()); err != nil {
			pw.CloseWithError(fmt.Errorf("compression failed: %w", err))
			return
		}

		if err := enc.Close(); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to close zstd writer: %w", err))
			return
		}
	}()

	return pr, nil
}

// Decompress decompresses the input stream using zstd.
func (c *ZstdCompressor) Decompress(input io.Reader) (io.Reader, error) {
	dec, err := zstd.NewReader(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd reader: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		defer dec.Close()

		if _, err := io.CopyBuffer(pw, dec, common.NewBuffer()); err != nil {
			pw.CloseWithError(fmt.Errorf("decompression failed: %w", err))
			return
		}
	}()

	return pr, nil
}

// Type returns the compression method type.
func (c *ZstdCompressor) Type() Method {
	return Zstd
}

// Extension returns the file extension for zstd compressed files.
func (c *ZstdCompressor) Extension() string {
	return ".zst"
}
