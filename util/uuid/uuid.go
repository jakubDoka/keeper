// uuid provides uuid generator that. It offers
// minimalistic support for the most common use cases.
package uuid

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

var (
	ErrInvalidLength       = errors.New("invalid length, expected 32")
	ErrInvalidLengthHyphen = errors.New("invalid length, expected 36")
)

const Length = 16

type UUID [Length]byte

var (
	m         sync.Mutex
	temp, Nil UUID
)

// New generates uuid with crypto/rand.Reader
func New() UUID {
	// this gets rid of allocation due to escape analysis
	m.Lock()
	n, err := rand.Reader.Read(temp[:])
	res := temp
	m.Unlock()

	if err != nil {
		panic(err)
	}
	if n != Length {
		panic("unexpected read length")
	}

	return res
}

// MustParse is like Parse, but panics on error.
func MustParse(str string) UUID {
	uuid, err := Parse(str)
	if err != nil {
		panic(err)
	}
	return uuid
}

// Parse parses a string of length 32 into a UUID. If length of the
// string does not match or id does not contain only hexadecimal digits,
// it returns an error.
func Parse(str string) (UUID, error) {
	if len(str) != Length*2 {
		return Nil, ErrInvalidLength
	}

	var data [Length * 2]byte
	copy(data[:], str)

	var res UUID
	_, err := hex.Decode(res[:], data[:])
	if err != nil {
		return Nil, err
	}

	return res, nil
}

// StringWithHyphens calls ParseWithHyphens and panics on error.
func MustParseWithHyphens(str string) UUID {
	uuid, err := ParseWithHyphens(str)
	if err != nil {
		panic(err)
	}
	return uuid
}

// ParseWithHyphen parses a string of length 36 into a UUID that contains hypens at indexes
// 8, 13, 18 and 23.
func ParseWithHyphens(str string) (UUID, error) {
	if len(str) != Length*2+4 {
		return Nil, ErrInvalidLengthHyphen
	}

	// get rid of hiphens
	var data [Length * 2]byte
	copy(data[:8], str[:8])
	copy(data[8:12], str[9:13])
	copy(data[12:16], str[14:18])
	copy(data[16:20], str[19:23])
	copy(data[20:], str[24:])

	var res UUID
	_, err := hex.Decode(res[:], data[:])
	if err != nil {
		return Nil, err
	}

	return res, nil
}

// Encodes uuid into string of length 32. (hexadecimal representation)
func (uuid UUID) String() string {
	var data [Length * 2]byte
	hex.Encode(data[:], uuid[:])
	return string(data[:])
}

// Encodes uuid into string of length 36. (hexadecimal representation with hyphens)
func (uuid UUID) StringWithHyphens() string {
	var data [Length*2 + 4]byte
	hex.Encode(data[:8], uuid[:4])
	data[8] = '-'
	hex.Encode(data[9:13], uuid[4:6])
	data[13] = '-'
	hex.Encode(data[14:18], uuid[6:8])
	data[18] = '-'
	hex.Encode(data[19:23], uuid[8:10])
	data[23] = '-'
	hex.Encode(data[24:], uuid[10:])
	return string(data[:])
}
