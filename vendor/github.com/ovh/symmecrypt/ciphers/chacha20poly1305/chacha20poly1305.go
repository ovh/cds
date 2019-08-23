package chacha20poly1305

import (
	"github.com/ovh/symmecrypt"
	"github.com/ovh/symmecrypt/symutils"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	// CipherName is the name of the cipher as registered on symmecrypt
	CipherName = "chacha20-poly1305"
)

func init() {
	symmecrypt.RegisterCipher(CipherName, symutils.NewFactoryAEAD(chacha20poly1305.KeySize, chacha20poly1305.New))
}
