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

package progress

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Config holds configuration for progress tracking
type Config struct {
	Description string // Description of the operation
	TotalBytes  int64  // Total bytes to process (0 for indeterminate)
	Enabled     bool   // Only true when --verbose flag is set
}

// Reader wraps an io.Reader with optional progress tracking
type Reader struct {
	reader io.Reader
	bar    *progressbar.ProgressBar
}

// NewReader creates a new progress-tracking reader
// If config.Enabled is false, returns a pass-through reader with no progress bar
func NewReader(r io.Reader, cfg Config) *Reader {
	if !cfg.Enabled {
		// Silent mode - no progress bar
		return &Reader{reader: r}
	}

	// Progress bar writes to stderr to not interfere with stdout
	var bar *progressbar.ProgressBar
	if cfg.TotalBytes > 0 {
		// Determinate progress bar (known size)
		bar = progressbar.NewOptions64(
			cfg.TotalBytes,
			progressbar.OptionSetDescription(cfg.Description),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(40),
			progressbar.OptionThrottle(100*time.Millisecond),
			progressbar.OptionShowCount(),
			progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
			progressbar.OptionSetWriter(os.Stderr), // Progress to stderr
		)
	} else {
		// Indeterminate progress (spinner)
		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription(cfg.Description),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(40),
			progressbar.OptionThrottle(100*time.Millisecond),
			progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionSpinnerType(14),
		)
	}

	return &Reader{reader: r, bar: bar}
}

// Read implements io.Reader and updates the progress bar
func (pr *Reader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if pr.bar != nil && n > 0 {
		pr.bar.Add(n)
	}
	return n, err
}

// Finish ensures the progress bar completes (useful for indeterminate progress)
func (pr *Reader) Finish() {
	if pr.bar != nil {
		pr.bar.Finish()
	}
}

// Writer wraps an io.Writer with optional progress tracking
type Writer struct {
	writer io.Writer
	bar    *progressbar.ProgressBar
}

// NewWriter creates a new progress-tracking writer
func NewWriter(w io.Writer, cfg Config) *Writer {
	if !cfg.Enabled {
		return &Writer{writer: w}
	}

	var bar *progressbar.ProgressBar
	if cfg.TotalBytes > 0 {
		bar = progressbar.NewOptions64(
			cfg.TotalBytes,
			progressbar.OptionSetDescription(cfg.Description),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(40),
			progressbar.OptionThrottle(100*time.Millisecond),
			progressbar.OptionShowCount(),
			progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
			progressbar.OptionSetWriter(os.Stderr),
		)
	} else {
		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription(cfg.Description),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(40),
			progressbar.OptionThrottle(100*time.Millisecond),
			progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionSpinnerType(14),
		)
	}

	return &Writer{writer: w, bar: bar}
}

// Write implements io.Writer and updates the progress bar
func (pw *Writer) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if pw.bar != nil && n > 0 {
		pw.bar.Add(n)
	}
	return n, err
}

// Finish ensures the progress bar completes
func (pw *Writer) Finish() {
	if pw.bar != nil {
		pw.bar.Finish()
	}
}
