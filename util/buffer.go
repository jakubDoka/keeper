package util

import (
	"crypto/aes"
	"encoding/binary"

	"github.com/jakubDoka/keeper/util/kcrypto"
	"github.com/jakubDoka/keeper/util/uuid"
)

// Reader is a helper for reading data packed by Writer
type Reader struct {
	buf    []byte
	offset int
}

// NewReader constructs new reader reading from start of the byte slice
func NewReader(buf []byte) Reader {
	return Reader{buf: buf}
}

// String reads a string from buffer. Returns false if failed.
func (r *Reader) String() (string, bool) {
	result, ok := r.Bytes()
	return string(result), ok
}

// Bytes reads byte slice with variable length. Returns false fi failed.
func (r *Reader) Bytes() ([]byte, bool) {
	size, ok := r.Uint32()
	if !ok {
		return nil, false
	}

	nextoffset := r.offset + int(size)

	if nextoffset > len(r.buf) {
		return nil, false
	}

	result := r.buf[r.offset:nextoffset]
	r.offset = nextoffset
	return result, true
}

// Key reads a cipher key from buffer. Returns false if failed.
func (r *Reader) Key() (kcrypto.Key, bool) {
	var result kcrypto.Key

	nextOffset := r.offset + len(result)
	if nextOffset > len(r.buf) {
		return kcrypto.Key{}, false
	}

	copy(result[:], r.buf[r.offset:nextOffset])
	r.offset = nextOffset
	return result, true
}

// UUID reads uuid from buffer. Returns false if failed.
func (r *Reader) UUID() (uuid.UUID, bool) {
	nextOffset := r.offset + 16
	if nextOffset > len(r.buf) {
		return uuid.Nil, false
	}
	var result uuid.UUID
	copy(result[:], r.buf[r.offset:nextOffset])
	r.offset = nextOffset
	return result, true
}

// Uint32 reads Uint32 from buffer with Big endian encoding. returns false if failed.
func (r *Reader) Uint32() (uint32, bool) {
	nextoffset := r.offset + 4
	if nextoffset > len(r.buf) {
		return 0, false
	}
	result := binary.BigEndian.Uint32(r.buf[r.offset:])
	r.offset = nextoffset
	return result, true
}

// Rest returns rest of the buffer.
func (r *Reader) Rest() []byte {
	return r.buf[r.offset:]
}

// Writer can produce byte slice for Reader to read.
type Writer struct {
	buf []byte
}

// NewWriter creates Writer with cap. You can use {} notation if you need cap of 0.
func NewWriter(cap int) Writer {
	return Writer{buf: make([]byte, 0, cap)}
}

// Uint32 writes uint32 to buffer in Big endian encoding.
func (w *Writer) Uint32(value uint32) *Writer {
	var data [4]byte
	binary.BigEndian.PutUint32(data[:], value)
	w.buf = append(w.buf, data[:]...)
	return w
}

// UUID writes uuid to buffer as is.
func (w *Writer) UUID(value uuid.UUID) *Writer {
	w.buf = append(w.buf, value[:]...)
	return w
}

// Key writes Key to buffer as is.
func (w *Writer) Key(value kcrypto.Key) *Writer {
	w.buf = append(w.buf, value[:]...)
	return w
}

// Bytes writes length of the slice as uint32 and then slice.
func (w *Writer) Bytes(value []byte) *Writer {
	w.Uint32(uint32(len(value)))
	w.buf = append(w.buf, value...)
	return w
}

// String writes length of the string as uint32 and then string it self.
func (w *Writer) String(value string) *Writer {
	w.Uint32(uint32(len(value)))
	w.buf = append(w.buf, value...)
	return w
}

// Rest appens the slice to buffer with no size hint.
func (w *Writer) Rest(buf []byte) *Writer {
	w.buf = append(w.buf, buf...)
	return w
}

// Buffer returns resulting buffer.
func (w *Writer) Buffer() []byte {
	return w.buf
}

// Calculator is for readable calculation of Writer Capacity. Its meant to save
// allocations yet express intent better then some magic constants.
type Calculator struct {
	offset int
}

// Pad increments counter so it is divisible by chunk. (specific i know)
func (c *Calculator) Pad(chunk int) *Calculator {
	c.offset += chunk - (c.offset % chunk)
	return c
}

// Uint32 increments counter by size of uint32 in bytes.
func (c *Calculator) Uint32() *Calculator {
	c.offset += 4
	return c
}

// UUID increments counter by size of uuid in bytes.
func (c *Calculator) UUID() *Calculator {
	c.offset += 16
	return c
}

// Key increments counter by size of kcripto.Key in bytes.
func (c *Calculator) Key() *Calculator {
	c.offset += kcrypto.KeySize + aes.BlockSize
	return c
}

// Bytes increments counter so it captures real buffer size of slice.
func (c *Calculator) Bytes(buf []byte) *Calculator {
	c.offset += 4 + len(buf)
	return c
}

// String increments counter so it captures real buffer size of string.
func (c *Calculator) String(str string) *Calculator {
	c.offset += 4 + len(str)
	return c
}

// Rest ...
func (c *Calculator) Rest(buf []byte) *Calculator {
	c.offset += len(buf)
	return c
}

// ToWriter produces writer out of calculator.
func (c *Calculator) ToWriter() Writer {
	return NewWriter(c.offset)
}

// Value return inner value.
func (c *Calculator) Value() int {
	return c.offset
}
