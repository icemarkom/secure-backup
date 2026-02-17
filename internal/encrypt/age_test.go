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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAgeKeyPair holds generated age keys for testing
type testAgeKeyPair struct {
	identity  *age.X25519Identity
	recipient string
	filePath  string // path to identity file
}

// generateTestAgeKeys creates a temporary age key pair for testing
func generateTestAgeKeys(t *testing.T) *testAgeKeyPair {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	recipientStr := identity.Recipient().String()

	// Write identity file
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "key.txt")
	err = WriteIdentityFile(keyFile, identity.String(), recipientStr)
	require.NoError(t, err)

	return &testAgeKeyPair{
		identity:  identity,
		recipient: recipientStr,
		filePath:  keyFile,
	}
}

func TestNewAgeEncryptor(t *testing.T) {
	cfg := Config{
		Method:     AGE,
		PublicKey:  "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p",
		PrivateKey: "/tmp/key.txt",
	}

	encryptor, err := NewAgeEncryptor(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, encryptor)
	assert.Equal(t, AGE, encryptor.Type())
}

func TestAgeEncryptor_EmptyPublicKey(t *testing.T) {
	encryptor, err := NewAgeEncryptor(Config{Method: AGE})
	require.NoError(t, err)

	plaintext := bytes.NewReader([]byte("test"))
	_, err = encryptor.Encrypt(plaintext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "public key not configured")
}

func TestAgeEncryptor_EmptyPrivateKeyPath(t *testing.T) {
	encryptor, err := NewAgeEncryptor(Config{Method: AGE})
	require.NoError(t, err)

	ciphertext := bytes.NewReader([]byte("test"))
	_, err = encryptor.Decrypt(ciphertext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private key path not configured")
}

func TestAgeEncryptor_InvalidPublicKey(t *testing.T) {
	encryptor, err := NewAgeEncryptor(Config{
		Method:    AGE,
		PublicKey: "not-a-valid-age-key",
	})
	require.NoError(t, err)

	plaintext := bytes.NewReader([]byte("test"))
	_, err = encryptor.Encrypt(plaintext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse age recipient")
}

func TestAgeEncryptor_InvalidPrivateKeyPath(t *testing.T) {
	encryptor, err := NewAgeEncryptor(Config{
		Method:     AGE,
		PrivateKey: "/nonexistent/key.txt",
	})
	require.NoError(t, err)

	ciphertext := bytes.NewReader([]byte("test"))
	_, err = encryptor.Decrypt(ciphertext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open age identity file")
}

func TestAgeEncryptor_RealEncryptDecrypt(t *testing.T) {
	keys := generateTestAgeKeys(t)

	originalData := []byte("This is secret test data for age encryption testing!")

	// Encrypt
	encryptor, err := NewAgeEncryptor(Config{
		Method:    AGE,
		PublicKey: keys.recipient,
	})
	require.NoError(t, err)

	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)
	require.NotEmpty(t, encryptedData)

	// Encrypted data should differ from original
	assert.NotEqual(t, originalData, encryptedData)

	// Decrypt
	decryptor, err := NewAgeEncryptor(Config{
		Method:     AGE,
		PrivateKey: keys.filePath,
	})
	require.NoError(t, err)

	decryptedReader, err := decryptor.Decrypt(bytes.NewReader(encryptedData))
	require.NoError(t, err)

	decryptedData, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)

	assert.Equal(t, originalData, decryptedData)
}

func TestAgeEncryptor_LargeData(t *testing.T) {
	keys := generateTestAgeKeys(t)

	// Create 1MB of test data
	originalData := bytes.Repeat([]byte("ABCDEFGH"), 128*1024) // 1MB

	// Encrypt
	encryptor, err := NewAgeEncryptor(Config{
		Method:    AGE,
		PublicKey: keys.recipient,
	})
	require.NoError(t, err)

	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Decrypt
	decryptor, err := NewAgeEncryptor(Config{
		Method:     AGE,
		PrivateKey: keys.filePath,
	})
	require.NoError(t, err)

	decryptedReader, err := decryptor.Decrypt(bytes.NewReader(encryptedData))
	require.NoError(t, err)

	decryptedData, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)

	assert.Equal(t, len(originalData), len(decryptedData))
	assert.Equal(t, originalData, decryptedData)
}

func TestAgeEncryptor_WrongKey(t *testing.T) {
	keys := generateTestAgeKeys(t)
	wrongKeys := generateTestAgeKeys(t) // Different key pair

	// Encrypt with one key
	encryptor, err := NewAgeEncryptor(Config{
		Method:    AGE,
		PublicKey: keys.recipient,
	})
	require.NoError(t, err)

	encryptedReader, err := encryptor.Encrypt(bytes.NewReader([]byte("secret")))
	require.NoError(t, err)

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Decrypt with wrong key
	decryptor, err := NewAgeEncryptor(Config{
		Method:     AGE,
		PrivateKey: wrongKeys.filePath,
	})
	require.NoError(t, err)

	_, err = decryptor.Decrypt(bytes.NewReader(encryptedData))
	assert.Error(t, err)
}

func TestAgeEncryptor_DecryptInvalidData(t *testing.T) {
	keys := generateTestAgeKeys(t)

	decryptor, err := NewAgeEncryptor(Config{
		Method:     AGE,
		PrivateKey: keys.filePath,
	})
	require.NoError(t, err)

	_, err = decryptor.Decrypt(bytes.NewReader([]byte("this is not valid age data")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt age message")
}

func TestAgeEncryptor_EmptyIdentityFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	err := os.WriteFile(emptyFile, []byte(""), 0600)
	require.NoError(t, err)

	decryptor, err := NewAgeEncryptor(Config{
		Method:     AGE,
		PrivateKey: emptyFile,
	})
	require.NoError(t, err)

	_, err = decryptor.Decrypt(bytes.NewReader([]byte("test")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no identities found")
}

func TestGenerateX25519Identity(t *testing.T) {
	identityStr, recipientStr, err := GenerateX25519Identity()
	require.NoError(t, err)
	assert.True(t, len(identityStr) > 0)
	assert.True(t, len(recipientStr) > 0)
	assert.Contains(t, identityStr, "AGE-SECRET-KEY-")
	assert.Contains(t, recipientStr, "age1")
}

func TestWriteIdentityFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "key.txt")

	err := WriteIdentityFile(path, "AGE-SECRET-KEY-1TEST", "age1test")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# public key: age1test")
	assert.Contains(t, string(data), "AGE-SECRET-KEY-1TEST")

	// Verify file permissions
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestParseRecipientFromIdentityFile(t *testing.T) {
	keys := generateTestAgeKeys(t)

	got, err := ParseRecipientFromIdentityFile(keys.filePath)
	require.NoError(t, err)
	assert.Equal(t, keys.recipient, got)
}

func TestNewEncryptor_Age(t *testing.T) {
	cfg := Config{
		Method:    AGE,
		PublicKey: "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p",
	}

	encryptor, err := NewEncryptor(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, encryptor)
	assert.Equal(t, AGE, encryptor.Type())
}
