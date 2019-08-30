package gorpmapping

import (
	"encoding/json"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/configstore"
	"github.com/ovh/symmecrypt/keyloader"
)

func ConfigureKeys(signatureKeys, encryptionKeys *[]keyloader.KeyConfig) error {
	// Marshal the keys
	var marshalledKeys [][]byte
	for _, k := range *signatureKeys {
		btes, err := json.Marshal(k)
		if err != nil {
			return sdk.WithStack(err)
		}
		marshalledKeys = append(marshalledKeys, btes)
	}
	for _, k := range *encryptionKeys {
		btes, err := json.Marshal(k)
		if err != nil {
			return sdk.WithStack(err)
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

	return nil
}
