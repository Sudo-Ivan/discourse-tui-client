// Copyright (c) 2025 Sudo-Ivan
// MIT License

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"syscall"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

const (
	KeyLength  = 32
	SaltLength = 16
	Iterations = 100000
)

// DeriveKey derives a key from password and salt using PBKDF2
func DeriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, Iterations, KeyLength, sha256.New)
}

// EncryptData encrypts data using AES-GCM
func EncryptData(data []byte, password string) ([]byte, error) {
	// Generate salt
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key
	key := DeriveKey(password, salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Prepend salt to ciphertext
	encrypted := append(salt, ciphertext...)
	return encrypted, nil
}

// DecryptData decrypts data using AES-GCM
func DecryptData(encrypted []byte, password string) ([]byte, error) {
	if len(encrypted) < SaltLength {
		return nil, fmt.Errorf("encrypted data too short")
	}

	// Extract salt
	salt := encrypted[:SaltLength]
	ciphertext := encrypted[SaltLength:]

	// Derive key
	key := DeriveKey(password, salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// PromptPassword securely prompts for a password
func PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Print newline after password input
	return string(password), err
}
