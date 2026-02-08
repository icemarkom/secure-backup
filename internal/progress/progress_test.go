package progress

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReader_Disabled(t *testing.T) {
	r := NewReader(strings.NewReader("test data"), Config{
		Description: "Test",
		TotalBytes:  9,
		Enabled:     false,
	})

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "test data", string(data))
}

func TestNewReader_Enabled(t *testing.T) {
	r := NewReader(strings.NewReader("test data"), Config{
		Description: "Test",
		TotalBytes:  9,
		Enabled:     true,
	})

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "test data", string(data))
}

func TestNewWriter_Disabled(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewWriter(buf, Config{
		Description: "Test",
		TotalBytes:  9,
		Enabled:     false,
	})

	n, err := w.Write([]byte("test data"))
	require.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, "test data", buf.String())
}

func TestNewWriter_Enabled(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewWriter(buf, Config{
		Description: "Test",
		TotalBytes:  9,
		Enabled:     true,
	})

	n, err := w.Write([]byte("test data"))
	require.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, "test data", buf.String())
}

// Edge case tests

func TestReader_ZeroSize(t *testing.T) {
	// Zero size should use indeterminate progress
	r := NewReader(strings.NewReader(""), Config{
		Description: "Empty",
		TotalBytes:  0,
		Enabled:     true,
	})

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestReader_Finish(t *testing.T) {
	r := NewReader(strings.NewReader("test"), Config{
		Description: "Test",
		TotalBytes:  4,
		Enabled:     true,
	})

	// Read all data
	_, err := io.ReadAll(r)
	require.NoError(t, err)

	// Finish should not panic
	r.Finish()
}

func TestReader_FinishDisabled(t *testing.T) {
	r := NewReader(strings.NewReader("test"), Config{
		Enabled: false,
	})

	// Finish with disabled progress should not panic
	r.Finish()
}

func TestWriter_MultipleWrites(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewWriter(buf, Config{
		Description: "Test",
		TotalBytes:  10,
		Enabled:     true,
	})

	// Multiple small writes
	n1, err1 := w.Write([]byte("hello"))
	n2, err2 := w.Write([]byte("world"))

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, 5, n1)
	assert.Equal(t, 5, n2)
	assert.Equal(t, "helloworld", buf.String())
}

func TestWriter_Finish(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewWriter(buf, Config{
		Description: "Test",
		TotalBytes:  5,
		Enabled:     true,
	})

	_, err := w.Write([]byte("hello"))
	require.NoError(t, err)

	// Finish should not panic
	w.Finish()
}

func TestWriter_FinishDisabled(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewWriter(buf, Config{
		Enabled: false,
	})

	// Finish with disabled progress should not panic
	w.Finish()
}

func TestReader_LargeData(t *testing.T) {
	// Test with larger data to exercise progress bar
	data := bytes.Repeat([]byte("A"), 1024*1024) // 1MB

	r := NewReader(bytes.NewReader(data), Config{
		Description: "Large test",
		TotalBytes:  int64(len(data)),
		Enabled:     true,
	})

	result, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, len(data), len(result))
	r.Finish()
}

func TestWriter_LargeData(t *testing.T) {
	// Test with larger data to exercise progress bar
	data := bytes.Repeat([]byte("B"), 1024*1024) // 1MB

	buf := &bytes.Buffer{}
	w := NewWriter(buf, Config{
		Description: "Large test",
		TotalBytes:  int64(len(data)),
		Enabled:     true,
	})

	n, err := w.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	w.Finish()
}

func TestReader_NilBar(t *testing.T) {
	// Disabled progress has nil bar - should still work
	r := &Reader{
		reader: strings.NewReader("test"),
		bar:    nil, // No progress bar
	}

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "test", string(data))

	// Finish with nil bar should not panic
	r.Finish()
}

func TestWriter_NilBar(t *testing.T) {
	// Disabled progress has nil bar - should still work
	buf := &bytes.Buffer{}
	w := &Writer{
		writer: buf,
		bar:    nil, // No progress bar
	}

	n, err := w.Write([]byte("test"))
	require.NoError(t, err)
	assert.Equal(t, 4, n)

	// Finish with nil bar should not panic
	w.Finish()
}
