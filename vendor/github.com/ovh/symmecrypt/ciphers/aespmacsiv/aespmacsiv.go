package aespmacsiv

import (
	"crypto/cipher"

	miscreant "github.com/miscreant/miscreant-go"
	"github.com/ovh/symmecrypt"
	"github.com/ovh/symmecrypt/symutils"
)

const (
	// KeyLen is the number of raw bytes for a key: 512bits split in 2
	KeyLen = 64

	// CipherName is the name of the cipher as registered on symmecrypt
	CipherName = "aes-pmac-siv"

	nonceSize = 32
)

func init() {
	symmecrypt.RegisterCipher(CipherName, symutils.NewFactoryAEADMutex(KeyLen, newAEAD))
	// use mutex factory, miscreant implementation does not seem concurrent safe (internal state)
}

func newAEAD(b []byte) (cipher.AEAD, error) {
	return miscreant.NewAEAD("AES-PMAC-SIV", b, nonceSize)
}
