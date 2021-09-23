// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xor

import (
	"bytes"
	"fmt"
	"testing"
)

func TestXOR(t *testing.T) {
	a := make([]byte, 100)
	b := make([]byte, 100)
	c := make([]byte, 100)
	for i := range b {
		b[i] = byte(i)
		c[i] = byte(i * i)
	}

	Bytes(a, b, c)

	d := make([]byte, 100)
	for i := range c {
		d[i] = b[i] ^ c[i]
	}

	if !bytes.Equal(a, d) {
		t.Errorf("\n%#v\n%#v", a, d)
	}
}

func BenchmarkXORBytes(b *testing.B) {
	dst := make([]byte, 1<<15)
	data0 := make([]byte, 1<<15)
	data1 := make([]byte, 1<<15)
	sizes := []int64{1 << 3, 1 << 7, 1 << 11, 1 << 15}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dBytes", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			for i := 0; i < b.N; i++ {
				Bytes(dst[:size], data0[:size], data1[:size])
			}
		})
	}
}
