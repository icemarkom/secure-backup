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

package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *UserError
		want string
	}{
		{
			name: "with hint",
			err: &UserError{
				Message: "Something went wrong",
				Hint:    "Try doing this instead",
			},
			want: "Something went wrong\nHint: Try doing this instead",
		},
		{
			name: "without hint",
			err: &UserError{
				Message: "Something went wrong",
			},
			want: "Something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUserError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &UserError{
		Message: "User-friendly message",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	assert.Equal(t, cause, unwrapped)
}

func TestUserError_Unwrap_NoCause(t *testing.T) {
	err := &UserError{
		Message: "User-friendly message",
	}

	unwrapped := err.Unwrap()
	assert.Nil(t, unwrapped)
}

func TestNew(t *testing.T) {
	err := New("test message", "test hint")
	require.NotNil(t, err)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, "test hint", err.Hint)
	assert.Nil(t, err.Cause)
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	err := Wrap(cause, "user message", "helpful hint")

	require.NotNil(t, err)
	assert.Equal(t, "user message", err.Message)
	assert.Equal(t, "helpful hint", err.Hint)
	assert.Equal(t, cause, err.Cause)
}

func TestMissingFile(t *testing.T) {
	err := MissingFile("/path/to/file.txt", "Create the file first")

	require.NotNil(t, err)
	assert.Equal(t, "File not found: /path/to/file.txt", err.Message)
	assert.Equal(t, "Create the file first", err.Hint)
}

func TestInvalidConfig(t *testing.T) {
	err := InvalidConfig("timeout", "must be positive", "Use a value greater than 0")

	require.NotNil(t, err)
	assert.Equal(t, "Invalid configuration for timeout: must be positive", err.Message)
	assert.Equal(t, "Use a value greater than 0", err.Hint)
}

func TestMissingRequired(t *testing.T) {
	err := MissingRequired("--api-key", "Provide your API key with --api-key")

	require.NotNil(t, err)
	assert.Equal(t, "Required parameter missing: --api-key", err.Message)
	assert.Equal(t, "Provide your API key with --api-key", err.Hint)
}

func TestUserError_ErrorChain(t *testing.T) {
	// Test that UserError works with errors.Is and errors.As
	originalErr := errors.New("original")
	wrappedErr := Wrap(originalErr, "wrapped", "hint")

	// Test errors.Is
	assert.True(t, errors.Is(wrappedErr, originalErr))

	// Test errors.As
	var userErr *UserError
	assert.True(t, errors.As(wrappedErr, &userErr))
	assert.Equal(t, "wrapped", userErr.Message)
}
