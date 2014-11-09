package netx

type Cipher struct {
	decrypt bool
}

func NewCipher(decrypt bool) *Cipher {
	return &Cipher{decrypt}
}
func (c *Cipher) XORKeyStream(dst, src []byte) {
	for i := 0; i < len(src); i++ {
		if c.decrypt {
			dst[i] = src[i] - 1
		} else {
			dst[i] = src[i] + 1
		}
	}
}
