// Package vpn: ChaCha20-Poly1305 cho Data tunnel (key từ password + salt).
package vpn

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"golang.org/x/crypto/chacha20poly1305"
)

var errDecrypt = errors.New("decrypt failed")

const (
	SaltSize  = 16
	NonceSize = 12
	KeySize   = 32
)

// DeriveKey tạo key 32 byte từ salt + password (SHA256).
func DeriveKey(salt []byte, password string) []byte {
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	return h.Sum(nil)[:KeySize]
}

// EncryptData encrypt plaintext với AEAD; nonce = nonceBuf (12 bytes, ghi seq vào đây); trả về nonce||ciphertext.
func EncryptData(aead cipherAEAD, plaintext []byte, seq uint64) (nonceAndCipher []byte, err error) {
	nonce := make([]byte, NonceSize)
	binary.BigEndian.PutUint64(nonce[0:8], seq)
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// DecryptData decrypt nonceAndCipher (12 byte nonce + ciphertext); trả về plaintext.
func DecryptData(aead cipherAEAD, nonceAndCipher []byte) (plaintext []byte, err error) {
	if len(nonceAndCipher) < NonceSize {
		return nil, errDecrypt
	}
	nonce := nonceAndCipher[:NonceSize]
	ciphertext := nonceAndCipher[NonceSize:]
	return aead.Open(nil, nonce, ciphertext, nil)
}

type cipherAEAD interface {
	Seal(dst, nonce, plaintext, additionalData []byte) []byte
	Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error)
}

// newAEAD tạo ChaCha20-Poly1305 AEAD từ key 32 byte.
func newAEAD(key []byte) (cipherAEAD, error) {
	return chacha20poly1305.New(key)
}

// ClientCrypto dùng cho client: encrypt/decrypt với sendSeq.
type ClientCrypto struct {
	aead    cipherAEAD
	sendSeq uint64
}

// NewClientCrypto tạo crypto từ salt (16 byte) + password (sau auth response).
func NewClientCrypto(salt []byte, password string) (*ClientCrypto, error) {
	key := DeriveKey(salt, password)
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	return &ClientCrypto{aead: aead, sendSeq: 0}, nil
}

// Encrypt plaintext, tăng sendSeq; trả về nonce||ciphertext.
func (c *ClientCrypto) Encrypt(plaintext []byte) ([]byte, error) {
	out, err := EncryptData(c.aead, plaintext, c.sendSeq)
	if err != nil {
		return nil, err
	}
	c.sendSeq++
	return out, nil
}

// Decrypt nonceAndCipher (12 + ciphertext), trả về plaintext.
func (c *ClientCrypto) Decrypt(nonceAndCipher []byte) ([]byte, error) {
	return DecryptData(c.aead, nonceAndCipher)
}
