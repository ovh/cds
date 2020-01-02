package gorpmapping

import (
	"encoding/json"
	"sync"

	"github.com/ovh/symmecrypt"

	"github.com/ovh/configstore"
	// Import all symmecrypt ciphers
	_ "github.com/ovh/symmecrypt/ciphers/aesgcm"
	_ "github.com/ovh/symmecrypt/ciphers/aespmacsiv"
	_ "github.com/ovh/symmecrypt/ciphers/chacha20poly1305"
	_ "github.com/ovh/symmecrypt/ciphers/hmac"
	_ "github.com/ovh/symmecrypt/ciphers/xchacha20poly1305"
	"github.com/ovh/symmecrypt/keyloader"

	"github.com/ovh/cds/sdk"
)

var (
	once          sync.Once
	signatureKey  symmecrypt.Key
	encryptionKey symmecrypt.Key
)

func ConfigureKeys(signatureKeys, encryptionKeys *[]keyloader.KeyConfig) error {
	var globalErr error
	once.Do(func() {
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

		// Push the keys in the keyloader
		// TODO: this should be updated to whole configuration management with configstore
		var provider configstore.Provider
		provider = func() (configstore.ItemList, error) {
			list := configstore.ItemList{}
			for _, btes := range marshalledKeys {
				list.Items = append(list.Items, configstore.NewItem(keyloader.EncryptionKeyConfigName, string(btes), 99))
			}

			return list, nil
		}
		configstore.RegisterProvider("fakeConfigstoreProvider", provider)

		var err error
		signatureKey, err = keyloader.WatchKey(KeySignIdentifier)
		if err != nil {
			globalErr = sdk.WithStack(err)
		}

		encryptionKey, err = keyloader.WatchKey(KeyEcnryptionIdentifier)
		if err != nil {
			globalErr = sdk.WithStack(err)
		}
	})

	return globalErr
}
