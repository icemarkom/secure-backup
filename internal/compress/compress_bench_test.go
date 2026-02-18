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
)

// benchData generates ~1MB of realistic, compressible data (repeated English text
// simulating log files and configuration data).
func benchData(b *testing.B) []byte {
	b.Helper()
	const line = "2026-02-17T22:30:00Z INFO  backup.pipeline: processing source=/var/lib/data size=1073741824 compression=gzip encryption=gpg retention=30 host=prod-server-01\n"
	target := 1024 * 1024 // 1 MB
	reps := target / len(line)
	data := []byte(strings.Repeat(line, reps))
	return data
}

// benchCompress benchmarks compression for a given method at its default level.
func benchCompress(b *testing.B, method Method) {
	data := benchData(b)

	comp, err := NewCompressor(Config{Method: method, Level: 0})
	if err != nil {
		b.Fatalf("NewCompressor: %v", err)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, err := comp.Compress(bytes.NewReader(data))
		if err != nil {
			b.Fatalf("Compress: %v", err)
		}
		if _, err := io.Copy(io.Discard, reader); err != nil {
			b.Fatalf("Read compressed: %v", err)
		}
	}
}

// benchDecompress benchmarks decompression for a given method at its default level.
func benchDecompress(b *testing.B, method Method) {
	data := benchData(b)

	comp, err := NewCompressor(Config{Method: method, Level: 0})
	if err != nil {
		b.Fatalf("NewCompressor: %v", err)
	}

	// Pre-compress the data once
	reader, err := comp.Compress(bytes.NewReader(data))
	if err != nil {
		b.Fatalf("Compress: %v", err)
	}
	compressed, err := io.ReadAll(reader)
	if err != nil {
		b.Fatalf("ReadAll: %v", err)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, err := comp.Decompress(bytes.NewReader(compressed))
		if err != nil {
			b.Fatalf("Decompress: %v", err)
		}
		if _, err := io.Copy(io.Discard, reader); err != nil {
			b.Fatalf("Read decompressed: %v", err)
		}
	}
}

// benchRatio measures compression ratio (compressed/original) for a given method.
func benchRatio(b *testing.B, method Method) {
	data := benchData(b)

	comp, err := NewCompressor(Config{Method: method, Level: 0})
	if err != nil {
		b.Fatalf("NewCompressor: %v", err)
	}

	reader, err := comp.Compress(bytes.NewReader(data))
	if err != nil {
		b.Fatalf("Compress: %v", err)
	}
	compressed, err := io.ReadAll(reader)
	if err != nil {
		b.Fatalf("ReadAll: %v", err)
	}

	ratio := float64(len(compressed)) / float64(len(data)) * 100
	b.ReportMetric(ratio, "%_of_original")
	b.ReportMetric(float64(len(data)-len(compressed)), "bytes_saved")
}

// --- Compress benchmarks ---

func BenchmarkGzipCompress(b *testing.B) { benchCompress(b, Gzip) }
func BenchmarkZstdCompress(b *testing.B) { benchCompress(b, Zstd) }
func BenchmarkLz4Compress(b *testing.B)  { benchCompress(b, Lz4) }

// --- Decompress benchmarks ---

func BenchmarkGzipDecompress(b *testing.B) { benchDecompress(b, Gzip) }
func BenchmarkZstdDecompress(b *testing.B) { benchDecompress(b, Zstd) }
func BenchmarkLz4Decompress(b *testing.B)  { benchDecompress(b, Lz4) }

// --- Compression ratio benchmarks ---

func BenchmarkGzipRatio(b *testing.B) { benchRatio(b, Gzip) }
func BenchmarkZstdRatio(b *testing.B) { benchRatio(b, Zstd) }
func BenchmarkLz4Ratio(b *testing.B)  { benchRatio(b, Lz4) }
