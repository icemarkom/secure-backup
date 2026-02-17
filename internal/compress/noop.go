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

import "io"

// NoopCompressor is a passthrough compressor that performs no compression.
// It implements the Compressor interface with identity transforms.
type NoopCompressor struct{}

// NewNoopCompressor creates a new passthrough compressor.
func NewNoopCompressor() *NoopCompressor {
	return &NoopCompressor{}
}

// Compress returns the input stream unchanged.
func (c *NoopCompressor) Compress(input io.Reader) (io.Reader, error) {
	return input, nil
}

// Decompress returns the input stream unchanged.
func (c *NoopCompressor) Decompress(input io.Reader) (io.Reader, error) {
	return input, nil
}

// Type returns None.
func (c *NoopCompressor) Type() Method {
	return None
}

// Extension returns an empty string since no compression suffix is needed.
func (c *NoopCompressor) Extension() string {
	return ""
}
