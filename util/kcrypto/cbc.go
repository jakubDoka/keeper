package kcrypto

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/jakubDoka/keeper/util/uuid"
	"github.com/jakubDoka/keeper/util/xor"
)

type CBCTCP struct {
	b cipher.Block

	encIV, decIV uuid.UUID
}

func NewCBCTCP(b cipher.Block, iv uuid.UUID) CBCTCP {
	if b.BlockSize() != aes.BlockSize {
		panic("expected cipher with block size 16")
	}

	return CBCTCP{b: b, encIV: iv, decIV: iv}
}

func (c *CBCTCP) Encrypt(dst, src []byte) {
	EncryptCBC(c.b, dst, src, c.encIV)
	c.b.Encrypt(c.encIV[:], c.encIV[:])
}

func (c *CBCTCP) Decrypt(dst, src []byte) {
	DecryptCBC(c.b, dst, src, c.decIV)
	c.b.Encrypt(c.decIV[:], c.decIV[:])
}

const IVCap = 30

type CBCUDP struct {
	b cipher.Block

	encIV  uuid.UUID
	encGen uint32

	decIV  []uuid.UUID
	decGen uint32
}

func NewCBCUDP(b cipher.Block, iv uuid.UUID) CBCUDP {
	if b.BlockSize() != aes.BlockSize {
		panic("expected cipher with block size 16")
	}

	decIV := make([]uuid.UUID, 1, IVCap)
	decIV[0] = iv

	return CBCUDP{b: b, encIV: iv, decIV: decIV}
}

func (c *CBCUDP) Encrypt(dst, src []byte) uint32 {
	EncryptCBC(c.b, dst, src, c.encIV)

	c.b.Encrypt(c.encIV[:], c.encIV[:])
	c.encGen++

	return c.encGen - 1
}

func (c *CBCUDP) Decrypt(dst, src []byte, gen uint32) bool {
	dif := int(c.decGen) - int(gen)
	if dif >= len(c.decIV) {
		return false
	}
	for dif < 0 {
		c.RefreshDecIV()
		dif++
	}
	DecryptCBC(c.b, dst, src, c.decIV[len(c.decIV)-dif-1])
	return true
}

func (c *CBCUDP) RefreshDecIV() {
	last := len(c.decIV) - 1
	var newIV uuid.UUID
	c.b.Encrypt(newIV[:], c.decIV[last][:])
	if len(c.decIV) >= IVCap {
		copy(c.decIV, c.decIV[1:])
		c.decIV[last] = newIV
	} else {
		c.decIV = append(c.decIV, newIV)
	}
	c.decGen++
}

func EncryptCBC(block cipher.Block, dst, src []byte, iv uuid.UUID) {
	if len(src)&(aes.BlockSize-1) != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}

	for len(src) > 0 {
		xor.Bytes(dst[:aes.BlockSize], src[:aes.BlockSize], iv[:])
		block.Encrypt(dst[:aes.BlockSize], dst[:aes.BlockSize])

		copy(iv[:], dst)
		src = src[aes.BlockSize:]
		dst = dst[aes.BlockSize:]
	}
}

func DecryptCBC(block cipher.Block, dst, src []byte, iv uuid.UUID) {
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
		block.Decrypt(dst[start:end], src[start:end])
		xor.Bytes(dst[start:end], dst[start:end], src[prev:start])

		end = start
		start = prev
		prev -= aes.BlockSize
	}

	block.Decrypt(dst[start:end], src[start:end])
	xor.Bytes(dst[start:end], dst[start:end], iv[:])
}
