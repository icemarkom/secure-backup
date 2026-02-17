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
	"strings"
)

// Method represents a supported compression method.
type Method int

const (
	// Gzip is the gzip compression method (using pgzip for parallelism).
	Gzip Method = iota
	// None disables compression (passthrough).
	None
)

// String names for compression methods, used in CLI flags and user-facing output.
const (
	MethodGzip = "gzip"
	MethodNone = "none"
)

// String returns the lowercase name of the compression method.
func (m Method) String() string {
	switch m {
	case Gzip:
		return MethodGzip
	case None:
		return MethodNone
	default:
		return fmt.Sprintf("unknown(%d)", int(m))
	}
}

// ValidMethods returns all supported compression methods.
func ValidMethods() []Method {
	return []Method{Gzip, None}
}

// ValidMethodNames returns a comma-separated string of valid method names.
// Useful for CLI help text and error messages.
func ValidMethodNames() string {
	methods := ValidMethods()
	names := make([]string, len(methods))
	for i, m := range methods {
		names[i] = m.String()
	}
	return strings.Join(names, ", ")
}

// ParseMethod converts a string to a Method. Returns an error for unknown methods.
func ParseMethod(s string) (Method, error) {
	switch strings.ToLower(s) {
	case MethodGzip:
		return Gzip, nil
	case MethodNone:
		return None, nil
	default:
		return 0, fmt.Errorf("unknown compression method: %s", s)
	}
}

// Compressor defines the interface for compression/decompression operations
type Compressor interface {
	// Compress compresses the input stream
	Compress(input io.Reader) (io.Reader, error)

	// Decompress decompresses the input stream
	Decompress(input io.Reader) (io.Reader, error)

	// Type returns the compression method type
	Type() Method

	// Extension returns file extension (".gz", ".zst")
	Extension() string
}

// Config holds compression configuration
type Config struct {
	Method Method // Compression method
	Level  int    // Compression level (method-specific)
}

// NewCompressor creates a compressor based on config
func NewCompressor(cfg Config) (Compressor, error) {
	switch cfg.Method {
	case Gzip:
		return NewGzipCompressor(cfg.Level)
	case None:
		return NewNoopCompressor(), nil
	default:
		return nil, fmt.Errorf("unknown compression method: %s", cfg.Method)
	}
}
