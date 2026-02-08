package encrypt

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptor(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid gpg config",
			config: Config{
				Method:     "gpg",
				PublicKey:  "/tmp/test.asc",
				PrivateKey: "/tmp/test.asc",
			},
			expectError: false,
		},
		{
			name: "default method (gpg)",
			config: Config{
				Method:     "",
				PublicKey:  "/tmp/test.asc",
				PrivateKey: "/tmp/test.asc",
			},
			expectError: false,
		},
		{
			name: "age not implemented",
			config: Config{
				Method: "age",
			},
			expectError: true,
			errorMsg:    "not yet implemented",
		},
		{
			name: "unknown method",
			config: Config{
				Method: "unknown",
			},
			expectError: true,
			errorMsg:    "unknown encryption method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := NewEncryptor(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, encryptor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encryptor)
			}
		})
	}
}

func TestGPGEncryptor_Type(t *testing.T) {
	cfg := Config{
		Method:     "gpg",
		PublicKey:  "/tmp/test.asc",
		PrivateKey: "/tmp/test.asc",
	}

	encryptor, err := NewGPGEncryptor(cfg)
	require.NoError(t, err)
	assert.Equal(t, "gpg", encryptor.Type())
}

func TestGPGEncryptor_InvalidPublicKey(t *testing.T) {
	cfg := Config{
		Method:    "gpg",
		PublicKey: "/nonexistent/public.asc",
	}

	encryptor, err := NewGPGEncryptor(cfg)
	require.NoError(t, err) // Constructor doesn't fail

	// Encrypt should fail when loading keys
	plaintext := bytes.NewReader([]byte("test"))
	_, err = encryptor.Encrypt(plaintext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load public keys")
}

func TestGPGEncryptor_InvalidPrivateKey(t *testing.T) {
	cfg := Config{
		Method:     "gpg",
		PrivateKey: "/nonexistent/private.asc",
	}

	encryptor, err := NewGPGEncryptor(cfg)
	require.NoError(t, err)

	// Decrypt should fail when loading private key
	ciphertext := bytes.NewReader([]byte("test"))
	_, err = encryptor.Decrypt(ciphertext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load private keys")
}

func TestGPGEncryptor_EmptyPublicKeyPath(t *testing.T) {
	cfg := Config{
		Method:    "gpg",
		PublicKey: "",
	}

	encryptor, err := NewGPGEncryptor(cfg)
	require.NoError(t, err)

	plaintext := bytes.NewReader([]byte("test"))
	_, err = encryptor.Encrypt(plaintext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "public key path not configured")
}

func TestGPGEncryptor_EmptyPrivateKeyPath(t *testing.T) {
	cfg := Config{
		Method:     "gpg",
		PrivateKey: "",
	}

	encryptor, err := NewGPGEncryptor(cfg)
	require.NoError(t, err)

	ciphertext := bytes.NewReader([]byte("test"))
	_, err = encryptor.Decrypt(ciphertext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private key path not configured")
}

func TestNewGPGEncryptor(t *testing.T) {
	cfg := Config{
		Method:     "gpg",
		PublicKey:  "/tmp/pub.asc",
		PrivateKey: "/tmp/priv.asc",
		Recipient:  "test@example.com",
		Passphrase: "secret",
	}

	encryptor, err := NewGPGEncryptor(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, encryptor)
	assert.Equal(t, "gpg", encryptor.Type())
}

// Integration tests with real GPG operations
// Uses test keys checked into test_data/

func getTestKeyPaths(t *testing.T) (publicKey, privateKey string) {
	t.Helper()

	// Find test_data directory (relative to test file)
	testDataDir := filepath.Join("..", "..", "test_data")

	publicKey = filepath.Join(testDataDir, "test-public.asc")
	privateKey = filepath.Join(testDataDir, "test-private.asc")

	// Verify test keys exist
	if _, err := os.Stat(publicKey); os.IsNotExist(err) {
		t.Skip("Test keys not found. Run: cd test_data && ./generate_test_keys.sh")
	}

	return publicKey, privateKey
}

func TestGPGEncryptor_RealEncryptDecrypt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GPG integration test in short mode")
	}

	publicKey, privateKey := getTestKeyPaths(t)

	// Test data
	originalData := []byte("This is secret test data for GPG encryption testing!")

	// Step 1: Encrypt
	encryptor, err := NewGPGEncryptor(Config{
		Method:    "gpg",
		PublicKey: publicKey,
	})
	require.NoError(t, err)

	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)
	require.NotEmpty(t, encryptedData)

	// Encrypted data should be different from original
	assert.NotEqual(t, originalData, encryptedData)

	// Step 2: Decrypt
	decryptor, err := NewGPGEncryptor(Config{
		Method:     "gpg",
		PrivateKey: privateKey,
		Passphrase: "", // Test key has no passphrase
	})
	require.NoError(t, err)

	decryptedReader, err := decryptor.Decrypt(bytes.NewReader(encryptedData))
	require.NoError(t, err)

	decryptedData, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)

	// Decrypted data should match original
	assert.Equal(t, originalData, decryptedData)
}

func TestGPGEncryptor_LargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GPG integration test in short mode")
	}

	publicKey, privateKey := getTestKeyPaths(t)

	// Create 1MB of test data
	originalData := bytes.Repeat([]byte("ABCDEFGH"), 128*1024) // 1MB

	// Encrypt
	encryptor, err := NewGPGEncryptor(Config{
		Method:    "gpg",
		PublicKey: publicKey,
	})
	require.NoError(t, err)

	encryptedReader, err := encryptor.Encrypt(bytes.NewReader(originalData))
	require.NoError(t, err)

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Decrypt
	decryptor, err := NewGPGEncryptor(Config{
		Method:     "gpg",
		PrivateKey: privateKey,
	})
	require.NoError(t, err)

	decryptedReader, err := decryptor.Decrypt(bytes.NewReader(encryptedData))
	require.NoError(t, err)

	decryptedData, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, len(originalData), len(decryptedData))
	assert.Equal(t, originalData, decryptedData)
}

func TestGPGEncryptor_WrongKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GPG integration test in short mode")
	}

	publicKey, _ := getTestKeyPaths(t)

	// Encrypt with test key
	encryptor, err := NewGPGEncryptor(Config{
		Method:    "gpg",
		PublicKey: publicKey,
	})
	require.NoError(t, err)

	encryptedReader, err := encryptor.Encrypt(bytes.NewReader([]byte("secret")))
	require.NoError(t, err)

	encryptedData, err := io.ReadAll(encryptedReader)
	require.NoError(t, err)

	// Try to decrypt with wrong key (nonexistent)
	decryptor, err := NewGPGEncryptor(Config{
		Method:     "gpg",
		PrivateKey: "/nonexistent/wrong.asc",
	})
	require.NoError(t, err)

	_, err = decryptor.Decrypt(bytes.NewReader(encryptedData))
	assert.Error(t, err)
}
