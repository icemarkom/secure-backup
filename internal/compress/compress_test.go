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
		{"Zstd", Zstd, "zstd"},
		{"None", None, "none"},
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
		{"zstd lowercase", "zstd", Zstd, false},
		{"none lowercase", "none", None, false},
		{"GZIP uppercase", "GZIP", Gzip, false},
		{"ZSTD uppercase", "ZSTD", Zstd, false},
		{"NONE uppercase", "NONE", None, false},
		{"Gzip mixed case", "Gzip", Gzip, false},
		{"Zstd mixed case", "Zstd", Zstd, false},
		{"None mixed case", "None", None, false},
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
	assert.Len(t, methods, 3)
	assert.Contains(t, methods, Gzip)
	assert.Contains(t, methods, Zstd)
	assert.Contains(t, methods, None)
}

func TestValidMethodNames(t *testing.T) {
	names := ValidMethodNames()
	assert.Contains(t, names, MethodGzip)
	assert.Contains(t, names, MethodZstd)
	assert.Contains(t, names, MethodNone)
	assert.Equal(t, "gzip, zstd, none", names)
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

func TestResolveMethod(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     Method
		wantErr  bool
	}{
		{"gzip gpg", "backup_test_20260101_120000.tar.gz.gpg", Gzip, false},
		{"gzip age", "backup_test_20260101_120000.tar.gz.age", Gzip, false},
		{"zstd gpg", "backup_test_20260101_120000.tar.zst.gpg", Zstd, false},
		{"zstd age", "backup_test_20260101_120000.tar.zst.age", Zstd, false},
		{"none gpg", "backup_test_20260101_120000.tar.gpg", None, false},
		{"none age", "backup_test_20260101_120000.tar.age", None, false},
		{"full path gzip", "/backups/daily/backup_data.tar.gz.gpg", Gzip, false},
		{"full path zstd", "/backups/daily/backup_data.tar.zst.gpg", Zstd, false},
		{"full path none", "/backups/daily/backup_data.tar.gpg", None, false},
		{"unknown extension", "backup.zip", Method(0), true},
		{"no extension", "backup", Method(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveMethod(tt.filename)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cannot detect compression method")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
