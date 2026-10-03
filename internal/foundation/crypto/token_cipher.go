// Package crypto provides field-level encryption-at-rest for sensitive
// values (OAuth tokens, API secrets) stored in the database, such as
// modules/marketplace's MarketplaceConnection access/refresh tokens.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// encPrefix marks a value as AES-GCM ciphertext produced by Encrypt, so
// Decrypt can tell it apart from legacy plaintext rows written before this
// package existed (or written while ENCRYPTION_KEY was unset) and return
// those unchanged instead of failing to decrypt them.
const encPrefix = "enc:v1:"

var (
	keyOnce sync.Once
	gcm     cipher.AEAD
	warnLog sync.Once
)

// loadKey reads ENCRYPTION_KEY (a base64-encoded 32-byte AES-256 key) from
// the environment. A missing key is not a fatal error here: it means this
// deployment hasn't opted in to field-level encryption yet, so Encrypt/
// Decrypt fall back to a no-op (plaintext) rather than crashing the server,
// unlike JWT_SECRET's fail-closed behavior - marketplace tokens already
// work without this key, so requiring it retroactively would break existing
// deployments on upgrade. Once an operator sets ENCRYPTION_KEY, new writes
// are encrypted; old plaintext rows keep working via the encPrefix check
// and get re-encrypted the next time they're saved (e.g. token refresh).
func loadKey() cipher.AEAD {
	keyOnce.Do(func() {
		raw := os.Getenv("ENCRYPTION_KEY")
		if raw == "" {
			return
		}
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != 32 {
			log.Printf("crypto: ENCRYPTION_KEY is set but is not valid base64-encoded 32 bytes; field-level encryption is disabled")
			return
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			log.Printf("crypto: failed to initialize AES cipher from ENCRYPTION_KEY: %v", err)
			return
		}
		g, err := cipher.NewGCM(block)
		if err != nil {
			log.Printf("crypto: failed to initialize AES-GCM from ENCRYPTION_KEY: %v", err)
			return
		}
		gcm = g
	})
	return gcm
}

// Encrypt encrypts plaintext for storage. If value is empty, or no
// ENCRYPTION_KEY is configured, it is returned unchanged.
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	g := loadKey()
	if g == nil {
		warnLog.Do(func() {
			log.Printf("crypto: ENCRYPTION_KEY is not set - sensitive fields (e.g. marketplace OAuth tokens) are being stored in plaintext. Set ENCRYPTION_KEY (base64, 32 bytes) to enable encryption at rest.")
		})
		return plaintext, nil
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := g.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. A value with no encPrefix is assumed to be
// legacy plaintext (written before encryption was enabled) and is returned
// as-is rather than treated as an error.
func Decrypt(value string) (string, error) {
	if value == "" || !strings.HasPrefix(value, encPrefix) {
		return value, nil
	}
	g := loadKey()
	if g == nil {
		return "", errors.New("crypto: value is encrypted but ENCRYPTION_KEY is not set - cannot decrypt")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encPrefix))
	if err != nil {
		return "", err
	}
	nonceSize := g.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("crypto: ciphertext too short")
	}
	nonce, sealed := raw[:nonceSize], raw[nonceSize:]
	plain, err := g.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
