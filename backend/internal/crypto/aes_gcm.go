package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

type AESGCM struct {
	aead cipher.AEAD
}

func NewAESGCM(encodedKey string) (*AESGCM, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 32 {
		key = []byte(encodedKey)
	}
	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes or base64-encoded 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes gcm: %w", err)
	}
	return &AESGCM{aead: aead}, nil
}

func (c *AESGCM) Encrypt(plaintext string) (ciphertext string, nonce string, err error) {
	nonceBytes := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", "", fmt.Errorf("generate nonce: %w", err)
	}

	encrypted := c.aead.Seal(nil, nonceBytes, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(encrypted),
		base64.StdEncoding.EncodeToString(nonceBytes),
		nil
}

func (c *AESGCM) Decrypt(ciphertext, nonce string) (string, error) {
	encrypted, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}

	plaintext, err := c.aead.Open(nil, nonceBytes, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt env var: %w", err)
	}
	return string(plaintext), nil
}
