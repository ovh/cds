package gpg

import (
	"github.com/ovh/configstore"

	"github.com/pkg/errors"
)

func NewPrivateKeyFromStore(keyAlias string, passphraseAlias string) (*PrivateKey, error) {
	config := configstore.Filter().Squash()
	key, err := config.MustGetItem(keyAlias).Value()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load GPG key from Vault")
	}

	passphrase, err := config.MustGetItem(passphraseAlias).Value()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load GPG passphrase from Vault")
	}

	return NewPrivateKeyFromPem(key, passphrase)
}

func NewPublicKeyFromStore(keyAlias string) (*PublicKey, error) {
	key, err := configstore.Filter().Squash().MustGetItem(keyAlias).Value()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load GPG key from Vault")
	}

	return NewPublicKeyFromPem(key)
}
