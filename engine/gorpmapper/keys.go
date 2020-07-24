package gorpmapper

import (
	"encoding/json"

	// Import all symmecrypt ciphers
	_ "github.com/ovh/symmecrypt/ciphers/aesgcm"
	_ "github.com/ovh/symmecrypt/ciphers/aespmacsiv"
	_ "github.com/ovh/symmecrypt/ciphers/chacha20poly1305"
	_ "github.com/ovh/symmecrypt/ciphers/hmac"
	_ "github.com/ovh/symmecrypt/ciphers/xchacha20poly1305"
	"github.com/ovh/symmecrypt/keyloader"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/configstore"
)

func (m *Mapper) ConfigureKeys(signatureKeys, encryptionKeys *[]keyloader.KeyConfig) error {
	var globalErr error
	m.once.Do(func() {
		// Marshal the keys
		var marshalledKeys [][]byte
		for _, k := range *signatureKeys {
			btes, err := json.Marshal(k)
			if err != nil {
				globalErr = sdk.WithStack(err)
				return
			}
			marshalledKeys = append(marshalledKeys, btes)
		}
		for _, k := range *encryptionKeys {
			btes, err := json.Marshal(k)
			if err != nil {
				globalErr = sdk.WithStack(err)
			}
			marshalledKeys = append(marshalledKeys, btes)
		}

		store := configstore.NewStore()

		var provider configstore.Provider
		provider = func() (configstore.ItemList, error) {
			list := configstore.ItemList{}
			for _, btes := range marshalledKeys {
				list.Items = append(list.Items, configstore.NewItem(keyloader.EncryptionKeyConfigName, string(btes), 99))
			}
			return list, nil
		}
		store.RegisterProvider("fakeConfigstoreProvider", provider)

		var err error
		m.signatureKey, err = keyloader.WatchKeyFromStore(KeySignIdentifier, store)
		if err != nil {
			globalErr = sdk.WithStack(err)
		}

		m.encryptionKey, err = keyloader.WatchKeyFromStore(KeyEcnryptionIdentifier, store)
		if err != nil {
			globalErr = sdk.WithStack(err)
		}
	})

	return globalErr
}
