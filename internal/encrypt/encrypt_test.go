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

package encrypt

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
		{"GPG", GPG, "gpg"},
		{"AGE", AGE, "age"},
		{"unknown", Method(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.method.String())
		})
	}
}

func TestMethod_Extension(t *testing.T) {
	tests := []struct {
		name   string
		method Method
		want   string
	}{
		{"GPG", GPG, "gpg"},
		{"AGE", AGE, "age"},
		{"unknown", Method(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.method.Extension())
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
		{"gpg lowercase", "gpg", GPG, false},
		{"age lowercase", "age", AGE, false},
		{"GPG uppercase", "GPG", GPG, false},
		{"AGE uppercase", "AGE", AGE, false},
		{"Gpg mixed case", "Gpg", GPG, false},
		{"Age mixed case", "Age", AGE, false},
		{"unknown method", "aes", Method(0), true},
		{"empty string", "", Method(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMethod(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unknown encryption method")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestValidMethods(t *testing.T) {
	methods := ValidMethods()
	assert.Len(t, methods, 2)
	assert.Contains(t, methods, GPG)
	assert.Contains(t, methods, AGE)
}

func TestValidMethodNames(t *testing.T) {
	names := ValidMethodNames()
	assert.Contains(t, names, MethodGPG)
	assert.Contains(t, names, MethodAGE)
	assert.Equal(t, "gpg, age", names)
}

func TestResolveMethod(t *testing.T) {
	tests := []struct {
		name     string
		explicit string
		filename string
		want     Method
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "explicit gpg",
			explicit: "gpg",
			filename: "backup.tar.gz.age",
			want:     GPG,
		},
		{
			name:     "explicit age",
			explicit: "age",
			filename: "backup.tar.gz.gpg",
			want:     AGE,
		},
		{
			name:     "explicit uppercase",
			explicit: "GPG",
			filename: "",
			want:     GPG,
		},
		{
			name:     "auto-detect .gpg",
			explicit: "",
			filename: "backup.tar.gz.gpg",
			want:     GPG,
		},
		{
			name:     "auto-detect .age",
			explicit: "",
			filename: "backup.tar.gz.age",
			want:     AGE,
		},
		{
			name:     "explicit unknown",
			explicit: "aes",
			filename: "backup.tar.gz.gpg",
			wantErr:  true,
			errMsg:   "unknown encryption method",
		},
		{
			name:     "auto-detect unknown extension",
			explicit: "",
			filename: "backup.tar.gz.enc",
			wantErr:  true,
			errMsg:   "cannot detect encryption method",
		},
		{
			name:     "auto-detect no extension",
			explicit: "",
			filename: "backup",
			wantErr:  true,
			errMsg:   "cannot detect encryption method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveMethod(tt.explicit, tt.filename)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewEncryptor_UnknownMethod(t *testing.T) {
	_, err := NewEncryptor(Config{Method: Method(99)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown encryption method")
}
