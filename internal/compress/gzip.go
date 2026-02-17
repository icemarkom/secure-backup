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

	gzip "github.com/klauspost/pgzip"
)

// GzipCompressor implements the Compressor interface using gzip
type GzipCompressor struct {
	level int
}

// NewGzipCompressor creates a new gzip compressor with the specified level.
// If level is 0, uses gzip.DefaultCompression (level 6).
// Valid levels: -1 (DefaultCompression), 0 (NoCompression), 1-9 (BestSpeed to BestCompression)
func NewGzipCompressor(level int) (*GzipCompressor, error) {
	if level == 0 {
		level = gzip.DefaultCompression // Level 6
	}

	// Validate level
	if level < gzip.HuffmanOnly || level > gzip.BestCompression {
		return nil, fmt.Errorf("invalid gzip compression level: %d (must be -1 to 9)", level)
	}

	return &GzipCompressor{level: level}, nil
}

// Compress compresses the input stream using gzip
func (c *GzipCompressor) Compress(input io.Reader) (io.Reader, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		gw, err := gzip.NewWriterLevel(pw, c.level)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to create gzip writer: %w", err))
			return
		}
		defer gw.Close()

		if _, err := io.Copy(gw, input); err != nil {
			pw.CloseWithError(fmt.Errorf("compression failed: %w", err))
			return
		}

		if err := gw.Close(); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to close gzip writer: %w", err))
			return
		}
	}()

	return pr, nil
}

// Decompress decompresses the input stream using gzip
func (c *GzipCompressor) Decompress(input io.Reader) (io.Reader, error) {
	gr, err := gzip.NewReader(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		defer gr.Close()

		if _, err := io.Copy(pw, gr); err != nil {
			pw.CloseWithError(fmt.Errorf("decompression failed: %w", err))
			return
		}
	}()

	return pr, nil
}

// Type returns the compression method type
func (c *GzipCompressor) Type() Method {
	return Gzip
}

// Extension returns the file extension for gzip compressed files
func (c *GzipCompressor) Extension() string {
	return ".gz"
}
