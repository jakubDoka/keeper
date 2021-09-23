package kcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"sync"

	"github.com/jakubDoka/keeper/util/uuid"
	"github.com/jakubDoka/keeper/util/xor"
)

type CBC struct {
	b       cipher.Block
	iv      uuid.UUID
	ivMutex sync.Mutex
}

func NewCBC(b cipher.Block, iv uuid.UUID) *CBC {
	if b.BlockSize() != aes.BlockSize {
		panic("expected cipher with block size 16")
	}

	return &CBC{b: b, iv: iv}
}

func (c *CBC) Encrypt(dst, src []byte) {
	if len(src)&(aes.BlockSize-1) != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}

	c.ivMutex.Lock()
	iv := c.iv
	c.ivMutex.Unlock()

	for len(src) > 0 {
		// Write the xor to dst, then encrypt in place.
		xor.Bytes(dst[:aes.BlockSize], src[:aes.BlockSize], iv[:])
		c.b.Encrypt(dst[:aes.BlockSize], dst[:aes.BlockSize])

		// Move to the next block with this block as the next iv.
		copy(iv[:], dst[:aes.BlockSize])
		src = src[aes.BlockSize:]
		dst = dst[aes.BlockSize:]
	}
}

func (c *CBC) Decrypt(dst, src []byte) {
	if len(src)&(aes.BlockSize-1) != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	if len(src) == 0 {
		return
	}

	end := len(src)
	start := end - aes.BlockSize
	prev := start - aes.BlockSize

	for start > 0 {
		c.b.Decrypt(dst[start:end], src[start:end])
		xor.Bytes(dst[start:end], dst[start:end], src[prev:start])

		end = start
		start = prev
		prev -= aes.BlockSize
	}

	iv := c.GetIV()

	c.b.Decrypt(dst[start:end], src[start:end])
	xor.Bytes(dst[start:end], dst[start:end], iv[:])
}

func (c *CBC) RefreshIV() {
	c.ivMutex.Lock()
	c.b.Encrypt(c.iv[:], c.iv[:])
	c.ivMutex.Unlock()
}

func (c *CBC) GetIV() uuid.UUID {
	c.ivMutex.Lock()
	iv := c.iv
	c.ivMutex.Unlock()
	return iv
}
