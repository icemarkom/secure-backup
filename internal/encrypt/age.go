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
	"os"
	"strings"

	"filippo.io/age"

	"github.com/icemarkom/secure-backup/internal/common"
)

// AgeEncryptor implements the Encryptor interface using age encryption
type AgeEncryptor struct {
	publicKey      string // age recipient string (age1...) or empty
	privateKeyPath string // path to age identity file
}

// NewAgeEncryptor creates a new age encryptor from the provided config.
// PublicKey is interpreted as a direct age recipient string (age1...).
// PrivateKey is interpreted as a file path to an age identity file.
func NewAgeEncryptor(cfg Config) (*AgeEncryptor, error) {
	return &AgeEncryptor{
		publicKey:      cfg.PublicKey,
		privateKeyPath: cfg.PrivateKey,
	}, nil
}

// Encrypt encrypts the plaintext stream using age encryption
func (e *AgeEncryptor) Encrypt(plaintext io.Reader) (io.Reader, error) {
	if e.publicKey == "" {
		return nil, fmt.Errorf("public key not configured")
	}

	// Parse the recipient string
	recipient, err := age.ParseX25519Recipient(e.publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse age recipient: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		// Create encrypted writer
		encWriter, err := age.Encrypt(pw, recipient)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to create age encrypted writer: %w", err))
			return
		}

		// Copy plaintext to encrypted writer
		if _, err := io.CopyBuffer(encWriter, plaintext, common.NewBuffer()); err != nil {
			pw.CloseWithError(fmt.Errorf("age encryption failed: %w", err))
			return
		}

		// Close the encrypted writer to finalize (flush last chunk)
		if err := encWriter.Close(); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to close age encrypted writer: %w", err))
			return
		}
	}()

	return pr, nil
}

// Decrypt decrypts the ciphertext stream using age encryption
func (e *AgeEncryptor) Decrypt(ciphertext io.Reader) (io.Reader, error) {
	if e.privateKeyPath == "" {
		return nil, fmt.Errorf("private key path not configured")
	}

	// Load identities from file
	identities, err := e.loadIdentities()
	if err != nil {
		return nil, fmt.Errorf("failed to load age identities: %w", err)
	}

	// Decrypt
	reader, err := age.Decrypt(ciphertext, identities...)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt age message: %w", err)
	}

	return reader, nil
}

// Type returns the encryption type
func (e *AgeEncryptor) Type() Method {
	return AGE
}

// loadIdentities loads age identities from the configured file path.
// The file format is one identity per line, with comments starting with "#".
func (e *AgeEncryptor) loadIdentities() ([]age.Identity, error) {
	f, err := os.Open(e.privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open age identity file %s: %w", e.privateKeyPath, err)
	}
	defer f.Close()

	identities, err := age.ParseIdentities(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse age identities from %s: %w", e.privateKeyPath, err)
	}

	if len(identities) == 0 {
		return nil, fmt.Errorf("no age identities found in %s", e.privateKeyPath)
	}

	return identities, nil
}

// GenerateX25519Identity generates a new age X25519 identity (key pair).
// Returns the identity string (AGE-SECRET-KEY-1...) and recipient string (age1...).
// This is primarily useful for testing.
func GenerateX25519Identity() (identityStr, recipientStr string, err error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate age identity: %w", err)
	}
	return identity.String(), identity.Recipient().String(), nil
}

// WriteIdentityFile writes an age identity to a file in the standard format.
// This is primarily useful for testing.
func WriteIdentityFile(path, identityStr, recipientStr string) error {
	content := fmt.Sprintf("# public key: %s\n%s\n", recipientStr, identityStr)
	return os.WriteFile(path, []byte(content), 0600)
}

// ParseRecipientFromIdentityFile reads an identity file and extracts the
// public key (recipient) from the comment line. Returns empty string if not found.
func ParseRecipientFromIdentityFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# public key: ") {
			return strings.TrimPrefix(line, "# public key: "), nil
		}
	}
	return "", fmt.Errorf("no public key comment found in %s", path)
}
