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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMethod_String(t *testing.T) {
	tests := []struct {
		name   string
		method Method
		want   string
	}{
		{"Gzip", Gzip, "gzip"},
		{"unknown", Method(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.method.String())
		})
	}
}

func TestParseMethod(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Method
		wantErr bool
	}{
		{"gzip lowercase", "gzip", Gzip, false},
		{"GZIP uppercase", "GZIP", Gzip, false},
		{"Gzip mixed case", "Gzip", Gzip, false},
		{"unknown method", "bzip2", Method(0), true},
		{"empty string", "", Method(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMethod(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unknown compression method")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestValidMethods(t *testing.T) {
	methods := ValidMethods()
	assert.Len(t, methods, 1)
	assert.Contains(t, methods, Gzip)
}

func TestValidMethodNames(t *testing.T) {
	names := ValidMethodNames()
	assert.Contains(t, names, MethodGzip)
	assert.Equal(t, "gzip", names)
}

func TestGzipCompressor_Type(t *testing.T) {
	compressor, err := NewGzipCompressor(6)
	require.NoError(t, err)
	assert.Equal(t, Gzip, compressor.Type())
}

func TestNewCompressor_UnknownMethod(t *testing.T) {
	_, err := NewCompressor(Config{Method: Method(99)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown compression method")
}
