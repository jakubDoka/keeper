package kcrypto

import (
	"crypto/aes"
	"crypto/rand"
	"errors"
	"sync"

	"github.com/jakubDoka/keeper/util/uuid"
)

var ErrInvalidPadding = errors.New("invalid padding")

const KeySize = 32

var (
	m    sync.Mutex
	temp Key
)

// Key fixed array of bytes representing both aes key and
// iv stored respectively.
type Key [KeySize + aes.BlockSize]byte

// NewKey creates random key using crypto/rand.Reader.
func NewKey() Key {
	// prevent allocation due to escape analysis
	m.Lock()
	n, err := rand.Reader.Read(temp[:])
	res := temp
	m.Unlock()

	if err != nil {
		panic(err)
	}
	if n != KeySize+aes.BlockSize {
		panic("unable to generate random key")
	}

	return res
}

func (k Key) IV() uuid.UUID {
	var iv uuid.UUID
	copy(iv[:], k[KeySize:])
	return iv
}

// Cipher is small abstraction over cipher package to make encryption
// and decryption nicer. It can be used concurrently. Uses aes.
type Cipher struct {
	key      Key
	enc, dec *CBC
}

// NewCipher creates new chipher with random key.
func NewCipher() Cipher {
	return NewCipherWithKey(NewKey())
}

// NewCipherWithKey creates new chiper with custom key.
func NewCipherWithKey(key Key) Cipher {
	block, err := aes.NewCipher(key[:KeySize])
	if err != nil {
		panic(err)
	}

	iv := key.IV()

	dec := NewCBC(block, iv)
	enc := NewCBC(block, iv)

	return Cipher{
		key: key,
		dec: dec,
		enc: enc,
	}
}

// Encrypt overwrites passed bytes and also appends the padding.
func (c *Cipher) Encrypt(plaintext []byte, refresh bool) []byte {
	// handle padding
	padding := aes.BlockSize - len(plaintext)&(aes.BlockSize-1)
	var pad [16]byte
	for i := 0; i < padding; i++ {
		pad[i] = byte(padding)
	}
	plaintext = append(plaintext, pad[:padding]...)

	c.enc.Encrypt(plaintext, plaintext)
	if refresh {
		c.dec.RefreshIV()
	}

	return plaintext
}

// Decrypt expects encoded message with padding. Error is returned only if
// padding has invalid format. That means at least last byte is holding number
// in range 1-16.
func (c *Cipher) Decrypt(ciphertext []byte, refresh bool) ([]byte, error) {
	c.dec.Decrypt(ciphertext, ciphertext)
	if refresh {
		c.dec.RefreshIV()
	}

	// handle padding
	padding := int(ciphertext[len(ciphertext)-1])
	if padding > aes.BlockSize {
		return nil, ErrInvalidPadding
	}

	return ciphertext[:len(ciphertext)-padding], nil
}

// Key returns cipher key.
func (c *Cipher) Key() Key {
	return c.key
}
