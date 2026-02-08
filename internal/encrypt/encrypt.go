package encrypt

import (
	"fmt"
	"io"
)

// Encryptor defines the interface for encryption/decryption operations
type Encryptor interface {
	// Encrypt encrypts the input stream and returns the encrypted output
	Encrypt(plaintext io.Reader) (io.Reader, error)

	// Decrypt decrypts the input stream and returns the plaintext output
	Decrypt(ciphertext io.Reader) (io.Reader, error)

	// Type returns the encryption method type ("gpg", "age")
	Type() string
}

// Config holds encryption configuration
type Config struct {
	Method     string // "gpg" or "age"
	PublicKey  string // Path to public key or key data
	PrivateKey string // Path to private key or key data
	Recipient  string // GPG recipient email (GPG only)
	Passphrase string // Key passphrase (optional)
}

// NewEncryptor creates an encryptor based on config
func NewEncryptor(cfg Config) (Encryptor, error) {
	// Default to GPG if method not specified
	if cfg.Method == "" {
		cfg.Method = "gpg"
	}

	switch cfg.Method {
	case "gpg":
		return NewGPGEncryptor(cfg)
	case "age":
		// Future implementation
		return nil, fmt.Errorf("age encryption not yet implemented")
	default:
		return nil, fmt.Errorf("unknown encryption method: %s", cfg.Method)
	}
}
