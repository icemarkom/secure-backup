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
