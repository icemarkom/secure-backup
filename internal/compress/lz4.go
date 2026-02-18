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
	"github.com/pierrec/lz4/v4"
)

// Lz4Compressor implements the Compressor interface using lz4.
type Lz4Compressor struct {
	level lz4.CompressionLevel
}

// NewLz4Compressor creates a new lz4 compressor with the specified level.
// If level is 0, uses lz4.Fast (default).
// Valid levels: 0 (fast/default), 1-9 (increasing compression).
func NewLz4Compressor(level int) (*Lz4Compressor, error) {
	if level < 0 || level > 9 {
		return nil, fmt.Errorf("invalid lz4 compression level: %d (must be 0-9)", level)
	}

	var compLevel lz4.CompressionLevel
	switch {
	case level == 0:
		compLevel = lz4.Fast
	default:
		compLevel = lz4.CompressionLevel(1 << (8 + level))
	}

	return &Lz4Compressor{level: compLevel}, nil
}

// Compress compresses the input stream using lz4.
func (c *Lz4Compressor) Compress(input io.Reader) (io.Reader, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		enc := lz4.NewWriter(pw)
		enc.Apply(lz4.CompressionLevelOption(c.level))

		if _, err := io.CopyBuffer(enc, input, common.NewBuffer()); err != nil {
			pw.CloseWithError(fmt.Errorf("compression failed: %w", err))
			return
		}

		if err := enc.Close(); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to close lz4 writer: %w", err))
			return
		}
	}()

	return pr, nil
}

// Decompress decompresses the input stream using lz4.
func (c *Lz4Compressor) Decompress(input io.Reader) (io.Reader, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		dec := lz4.NewReader(input)

		if _, err := io.CopyBuffer(pw, dec, common.NewBuffer()); err != nil {
			pw.CloseWithError(fmt.Errorf("decompression failed: %w", err))
			return
		}
	}()

	return pr, nil
}

// Type returns the compression method type.
func (c *Lz4Compressor) Type() Method {
	return Lz4
}

// Extension returns the file extension for lz4 compressed files.
func (c *Lz4Compressor) Extension() string {
	return ".lz4"
}
