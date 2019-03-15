package gorpmapping

import (
	"encoding/json"
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/symmecrypt/keyloader"
)

const (
	KeyEcnryptionIdentifier = "db-crypt"
)

func Encrypt(src interface{}, dst *[]byte, extra ...[]byte) error {
	k, err := keyloader.LoadKey(KeyEcnryptionIdentifier)
	if err != nil {
		return sdk.WithStack(err)
	}

	clearContent, err := json.Marshal(src)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to marshal content: %v", err))
	}

	btes, err := k.Encrypt(clearContent, extra...)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to encrypt content: %v", err))
	}

	*dst = btes

	return nil

}

func Decrypt(src []byte, dest interface{}, extra ...[]byte) error {
	k, err := keyloader.LoadKey(KeyEcnryptionIdentifier)
	if err != nil {
		return sdk.WithStack(err)
	}

	clearContent, err := k.Decrypt(src, extra...)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to decrypt content: %v", err))
	}

	return json.Unmarshal(clearContent, dest)
}
