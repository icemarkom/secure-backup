package encrypt

import (
	"bytes"
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
