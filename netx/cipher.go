package netx

import (
	"crypto/cipher"
)

type mycipher struct {
	decrypt bool
}

func NewCipher(decrypt bool) cipher.Stream {
	return &mycipher{decrypt}
}
func (c *mycipher) XORKeyStream(dst, src []byte) {
	if c.decrypt {
		decrypt(dst, src)
	} else {
		encrypt(dst, src)
	}
}

func encrypt(dst, src []byte) {
	for i := 0; i < len(src); i++ {
		dst[i] = src[i] + 1
	}
}

func decrypt(dst, src []byte) {
	for i := 0; i < len(src); i++ {
		dst[i] = src[i] - 1
	}
}
