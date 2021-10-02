package kcrypto

import (
	"fmt"
	"testing"
)

func TestCipher(t *testing.T) {
	var k Key
	for i := range k {
		k[i] = byte(i)
	}

	str := "0123456789ABCDEF"

	c := NewCipherWithKey(k)
	fmt.Println(c.EncryptTCP([]byte(str)))
	t.Fail()
}
