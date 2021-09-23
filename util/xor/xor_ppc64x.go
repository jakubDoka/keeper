// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ppc64 || ppc64le
// +build ppc64 ppc64le

package xor

// Bytes xors the bytes in a and b. The destination should have enough
// space, otherwise xorBytes will panic. Returns the number of bytes xor'd.
func Bytes(dst, a, b []byte) int {
	n := len(a)
	if n != len(b) || n != len(dst) {
		panic("dst, a and b has to have same size")
	}
	xorBytesVSX(&dst[0], &a[0], &b[0], n)
	return n
}

//go:noescape
func xorBytesVSX(dst, a, b *byte, n int)
