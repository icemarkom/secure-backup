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
	"fmt"
	"io"
	"strings"
)

// Method represents a supported encryption method.
type Method int

const (
	// GPG is the GNU Privacy Guard encryption method.
	GPG Method = iota
	// AGE is the AGE encryption method (filippo.io/age).
	AGE
)

// String names for encryption methods, used in CLI flags, file extensions,
// and user-facing output.
const (
	MethodGPG = "gpg"
	MethodAGE = "age"
)

// String returns the lowercase name of the encryption method.
func (m Method) String() string {
	switch m {
	case GPG:
		return MethodGPG
	case AGE:
		return MethodAGE
	default:
		return fmt.Sprintf("unknown(%d)", int(m))
	}
}

// Extension returns the file extension for the encryption method (without dot).
func (m Method) Extension() string {
	return m.String()
}

// ValidMethods returns all supported encryption methods.
func ValidMethods() []Method {
	return []Method{GPG, AGE}
}

// ValidMethodNames returns a comma-separated string of valid method names.
// Useful for CLI help text and error messages.
func ValidMethodNames() string {
	methods := ValidMethods()
	names := make([]string, len(methods))
	for i, m := range methods {
		names[i] = m.String()
	}
	return strings.Join(names, ", ")
}

// ParseMethod converts a string to a Method. Returns an error for unknown methods.
func ParseMethod(s string) (Method, error) {
	switch strings.ToLower(s) {
	case MethodGPG:
		return GPG, nil
	case MethodAGE:
		return AGE, nil
	default:
		return 0, fmt.Errorf("unknown encryption method: %s", s)
	}
}

// Encryptor defines the interface for encryption/decryption operations
type Encryptor interface {
	// Encrypt encrypts the input stream and returns the encrypted output
	Encrypt(plaintext io.Reader) (io.Reader, error)

	// Decrypt decrypts the input stream and returns the plaintext output
	Decrypt(ciphertext io.Reader) (io.Reader, error)

	// Type returns the encryption method type
	Type() Method
}

// Config holds encryption configuration
type Config struct {
	Method     Method // GPG or AGE
	PublicKey  string // Path to public key or key data
	PrivateKey string // Path to private key or key data
	Recipient  string // GPG recipient email (GPG only)
	Passphrase string // Key passphrase (optional)
}

// NewEncryptor creates an encryptor based on config
func NewEncryptor(cfg Config) (Encryptor, error) {
	switch cfg.Method {
	case GPG:
		return NewGPGEncryptor(cfg)
	case AGE:
		return NewAgeEncryptor(cfg)
	default:
		return nil, fmt.Errorf("unknown encryption method: %s", cfg.Method)
	}
}

// ResolveMethod returns the encryption method to use. If explicit is non-empty,
// it is returned as-is. Otherwise the method is auto-detected from the file extension.
func ResolveMethod(explicit, filename string) (Method, error) {
	if explicit != "" {
		return ParseMethod(explicit)
	}

	switch {
	case strings.HasSuffix(filename, "."+MethodGPG):
		return GPG, nil
	case strings.HasSuffix(filename, "."+MethodAGE):
		return AGE, nil
	default:
		return 0, fmt.Errorf("cannot detect encryption method from file extension: %s (use --encryption to specify)", filename)
	}
}
