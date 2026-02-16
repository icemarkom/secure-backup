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
)

// Compressor defines the interface for compression/decompression operations
type Compressor interface {
	// Compress compresses the input stream
	Compress(input io.Reader) (io.Reader, error)

	// Decompress decompresses the input stream
	Decompress(input io.Reader) (io.Reader, error)

	// Extension returns file extension (".gz", ".zst")
	Extension() string
}

// Config holds compression configuration
type Config struct {
	Method string // "gzip", "zstd", "lz4"
	Level  int    // Compression level (method-specific)
}

// NewCompressor creates a compressor based on config
func NewCompressor(cfg Config) (Compressor, error) {
	// Default to gzip if method not specified
	if cfg.Method == "" {
		cfg.Method = "gzip"
	}

	switch cfg.Method {
	case "gzip":
		return NewGzipCompressor(cfg.Level)
	case "zstd":
		// Future implementation
		return nil, fmt.Errorf("zstd compression not yet implemented")
	case "lz4":
		// Future implementation
		return nil, fmt.Errorf("lz4 compression not yet implemented")
	default:
		return nil, fmt.Errorf("unknown compression method: %s", cfg.Method)
	}
}
