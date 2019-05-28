package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/ovh/symmecrypt"
	"github.com/ovh/symmecrypt/symutils"
)

const (
	// KeyLen is the number of raw bytes for a key: AES256
	KeyLen = 32

	// CipherName is the name of the cipher as registered on symmecrypt
	CipherName = "aes-gcm"
)

func init() {
	symmecrypt.RegisterCipher(CipherName, symutils.NewFactoryAEAD(KeyLen, gcmCipher))
}

func gcmCipher(b []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher([]byte(b))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
