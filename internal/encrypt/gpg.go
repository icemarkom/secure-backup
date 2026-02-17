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

	"golang.org/x/crypto/openpgp"
)

// GPGEncryptor implements the Encryptor interface using GPG/OpenPGP
type GPGEncryptor struct {
	publicKeyPath  string
	privateKeyPath string
	recipient      string
	passphrase     []byte
}

// NewGPGEncryptor creates a new GPG encryptor from the provided config
func NewGPGEncryptor(cfg Config) (*GPGEncryptor, error) {
	return &GPGEncryptor{
		publicKeyPath:  cfg.PublicKey,
		privateKeyPath: cfg.PrivateKey,
		recipient:      cfg.Recipient,
		passphrase:     []byte(cfg.Passphrase),
	}, nil
}

// Encrypt encrypts the plaintext stream using binary GPG format
func (e *GPGEncryptor) Encrypt(plaintext io.Reader) (io.Reader, error) {
	// Load public key(s)
	keyring, err := e.loadPublicKeyring()
	if err != nil {
		return nil, fmt.Errorf("failed to load public keys: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		// Create encrypted writer (binary output — no armor)
		encWriter, err := openpgp.Encrypt(pw, keyring, nil, nil, nil)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to create encrypted writer: %w", err))
			return
		}
		defer encWriter.Close()

		// Copy plaintext to encrypted writer
		if _, err := io.Copy(encWriter, plaintext); err != nil {
			pw.CloseWithError(fmt.Errorf("encryption failed: %w", err))
			return
		}

		if err := encWriter.Close(); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to close encrypted writer: %w", err))
			return
		}
	}()

	return pr, nil
}

// Decrypt decrypts the ciphertext stream using GPG
func (e *GPGEncryptor) Decrypt(ciphertext io.Reader) (io.Reader, error) {
	// Load private keyring
	keyring, err := e.loadPrivateKeyring()
	if err != nil {
		return nil, fmt.Errorf("failed to load private keys: %w", err)
	}

	// Decrypt the passphrase-protected private keys
	for _, entity := range keyring {
		if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
			if err := entity.PrivateKey.Decrypt(e.passphrase); err != nil {
				return nil, fmt.Errorf("failed to decrypt private key: %w", err)
			}
		}
		for _, subkey := range entity.Subkeys {
			if subkey.PrivateKey != nil && subkey.PrivateKey.Encrypted {
				if err := subkey.PrivateKey.Decrypt(e.passphrase); err != nil {
					return nil, fmt.Errorf("failed to decrypt subkey: %w", err)
				}
			}
		}
	}

	// Read binary GPG message directly
	md, err := openpgp.ReadMessage(ciphertext, keyring, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted message: %w", err)
	}

	return md.UnverifiedBody, nil
}

// Type returns the encryption type
func (e *GPGEncryptor) Type() Method {
	return GPG
}

// loadPublicKeyring loads the public keyring from the configured path
func (e *GPGEncryptor) loadPublicKeyring() (openpgp.EntityList, error) {
	if e.publicKeyPath == "" {
		return nil, fmt.Errorf("public key path not configured")
	}

	keyFile, err := os.Open(e.publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open public key file %s: %w", e.publicKeyPath, err)
	}
	defer keyFile.Close()

	// Try armored format first
	keyring, err := openpgp.ReadArmoredKeyRing(keyFile)
	if err != nil {
		// Try binary format
		keyFile.Seek(0, 0)
		keyring, err = openpgp.ReadKeyRing(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read public keyring: %w", err)
		}
	}

	if len(keyring) == 0 {
		return nil, fmt.Errorf("no public keys found in %s", e.publicKeyPath)
	}

	return keyring, nil
}

// loadPrivateKeyring loads the private keyring from the configured path
func (e *GPGEncryptor) loadPrivateKeyring() (openpgp.EntityList, error) {
	if e.privateKeyPath == "" {
		return nil, fmt.Errorf("private key path not configured")
	}

	keyFile, err := os.Open(e.privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open private key file %s: %w", e.privateKeyPath, err)
	}
	defer keyFile.Close()

	// Try armored format first
	keyring, err := openpgp.ReadArmoredKeyRing(keyFile)
	if err != nil {
		// Try binary format
		keyFile.Seek(0, 0)
		keyring, err = openpgp.ReadKeyRing(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read private keyring: %w", err)
		}
	}

	if len(keyring) == 0 {
		return nil, fmt.Errorf("no private keys found in %s", e.privateKeyPath)
	}

	return keyring, nil
}
