package gorpmapping

import (
	"github.com/ovh/symmecrypt/keyloader"
)

func ConfigureKeys(signatureKeys, encryptionKeys *[]keyloader.KeyConfig) error {
	return Mapper.ConfigureKeys(signatureKeys, encryptionKeys)
}
