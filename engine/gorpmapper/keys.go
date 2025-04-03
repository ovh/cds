package gorpmapper

import (
	"encoding/json"
	"sort"

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

func (m *Mapper) ConfigureKeys(signatureKeys, encryptionKeys []keyloader.KeyConfig) error {
	var globalErr error
	m.once.Do(func() {
		// Marshal the keys
		var marshalledKeys [][]byte

		sort.Slice(signatureKeys, func(i, j int) bool { return signatureKeys[i].Timestamp > signatureKeys[j].Timestamp })
		m.signatureKeyTimestamp = make([]int64, len(signatureKeys))
		for i, k := range signatureKeys {
			m.signatureKeyTimestamp[i] = k.Timestamp
			btes, err := json.Marshal(k)
			if err != nil {
				globalErr = sdk.WithStack(err)
				return
			}
			marshalledKeys = append(marshalledKeys, btes)
		}

		sort.Slice(encryptionKeys, func(i, j int) bool { return encryptionKeys[i].Timestamp > encryptionKeys[j].Timestamp })
		m.encryptionKeyTimestamp = make([]int64, len(encryptionKeys))
		for i, k := range encryptionKeys {
			m.encryptionKeyTimestamp[i] = k.Timestamp
			btes, err := json.Marshal(k)
			if err != nil {
				globalErr = sdk.WithStack(err)
			}
			marshalledKeys = append(marshalledKeys, btes)
		}

		store := configstore.NewStore()
		store.RegisterProvider("fakeConfigstoreProvider", func() (configstore.ItemList, error) {
			list := configstore.ItemList{}
			for _, btes := range marshalledKeys {
				list.Items = append(list.Items, configstore.NewItem(keyloader.EncryptionKeyConfigName, string(btes), 99))
			}
			return list, nil
		})

		var err error
		m.signatureKey, err = keyloader.LoadKeyFromStore(KeySignIdentifier, store)
		if err != nil {
			globalErr = sdk.WithStack(err)
		}

		m.encryptionKey, err = keyloader.LoadKeyFromStore(KeyEncryptionIdentifier, store)
		if err != nil {
			globalErr = sdk.WithStack(err)
		}
	})

	return globalErr
}
