package kcrypto

import (
	"crypto/aes"
	"crypto/rand"
	"errors"
	"sync"

	"github.com/jakubDoka/keeper/util/uuid"
)

var (
	ErrInvalidPadding = errors.New("invalid padding")
	ErrPacketLost     = errors.New("packet lost")
)

const KeySize = 32

var (
	m    sync.Mutex
	temp Key
)

type Key [KeySize + aes.BlockSize*2]byte

func NewKey() Key {
	// prevent allocation due to escape analysis
	m.Lock()
	n, err := rand.Reader.Read(temp[:])
	res := temp
	m.Unlock()

	if err != nil {
		panic(err)
	}
	if n != len(res) {
		panic("unable to generate random key")
	}

	return res
}

func (k Key) IV() (udp, tcp uuid.UUID) {
	copy(udp[:], k[KeySize:])
	copy(tcp[:], k[KeySize+aes.BlockSize:])
	return
}

// Cipher is small abstraction over cipher package to make encryption
// and decryption nicer. It can be used concurrently. Uses aes.
type Cipher struct {
	key           Key
	udp           CBCUDP
	tcp           CBCTCP
	isInitialized bool
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

	tcp, udp := key.IV()

	return Cipher{key, NewCBCUDP(block, tcp), NewCBCTCP(block, udp), true}
}

func (c *Cipher) EncryptTCP(plaintext []byte) []byte {
	plaintext = AddPadding(plaintext)

	c.tcp.Encrypt(plaintext, plaintext)

	return plaintext
}

func (c *Cipher) DecryptTCP(ciphertext []byte) ([]byte, error) {
	c.tcp.Decrypt(ciphertext, ciphertext)

	return RemovePadding(ciphertext)
}

func (c *Cipher) EncryptUDP(plaintext []byte) ([]byte, uint32) {
	plaintext = AddPadding(plaintext)

	gen := c.udp.Encrypt(plaintext, plaintext)

	return plaintext, gen
}

func (c *Cipher) DecryptUDP(ciphertext []byte, gen uint32) ([]byte, error) {
	ok := c.udp.Decrypt(ciphertext, ciphertext, gen)
	if !ok {
		return nil, ErrPacketLost
	}

	return RemovePadding(ciphertext)
}

// Key returns cipher key.
func (c *Cipher) Key() Key {
	return c.key
}

func (c *Cipher) IsNil() bool {
	return !c.isInitialized
}

func AddPadding(plaintext []byte) []byte {
	padding := aes.BlockSize - len(plaintext)&(aes.BlockSize-1)
	var pad [16]byte
	for i := 0; i < padding; i++ {
		pad[i] = byte(padding)
	}
	return append(plaintext, pad[:padding]...)
}

func RemovePadding(plaintext []byte) ([]byte, error) {
	padding := int(plaintext[len(plaintext)-1])
	if padding > aes.BlockSize {
		return nil, ErrInvalidPadding
	}

	return plaintext[:len(plaintext)-padding], nil
}
