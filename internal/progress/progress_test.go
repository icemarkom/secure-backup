package progress

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestReader_Silent(t *testing.T) {
	data := []byte("test data")
	r := NewReader(bytes.NewReader(data), Config{
		Description: "Testing",
		TotalBytes:  int64(len(data)),
		Enabled:     false, // Silent mode
	})

	buf := make([]byte, len(data))
	n, err := r.Read(buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected %d bytes, got %d", len(data), n)
	}
	if !bytes.Equal(buf, data) {
		t.Fatalf("data mismatch")
	}
}

func TestReader_Enabled(t *testing.T) {
	data := []byte(strings.Repeat("x", 1000))
	r := NewReader(bytes.NewReader(data), Config{
		Description: "Testing",
		TotalBytes:  int64(len(data)),
		Enabled:     true, // Progress enabled
	})

	buf := make([]byte, len(data))
	n, err := r.Read(buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected %d bytes, got %d", len(data), n)
	}
	if !bytes.Equal(buf, data) {
		t.Fatalf("data mismatch")
	}
}

func TestWriter_Silent(t *testing.T) {
	data := []byte("test data")
	buf := &bytes.Buffer{}
	w := NewWriter(buf, Config{
		Description: "Testing",
		TotalBytes:  int64(len(data)),
		Enabled:     false,
	})

	n, err := w.Write(data)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected %d bytes, got %d", len(data), n)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Fatalf("data mismatch")
	}
}

func TestReader_MultipleReads(t *testing.T) {
	data := []byte(strings.Repeat("x", 1000))
	r := NewReader(bytes.NewReader(data), Config{
		Description: "Testing",
		TotalBytes:  int64(len(data)),
		Enabled:     true,
	})

	// Read in chunks
	total := 0
	buf := make([]byte, 100)
	for {
		n, err := r.Read(buf)
		total += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if total != len(data) {
		t.Fatalf("expected %d total bytes, got %d", len(data), total)
	}
}
