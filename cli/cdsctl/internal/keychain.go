// +build !nokeychain

package internal

import (
	"fmt"

	keychain "github.com/keybase/go-keychain"
)

var keychainEnabled = true

//storeToken store a context into keychain
func storeToken(contextName, token string) error {
	// delete existing value if present
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(contextName)
	item.SetAccount(contextName)
	keychain.DeleteItem(item) // nolint: errcheck

	// populate the rest of the object and save
	label := fmt.Sprintf("CDS-cdsctl/%s", contextName)
	item.SetLabel(label)
	item.SetData([]byte(token))
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleAccessibleAlwaysThisDeviceOnly)

	if err := keychain.AddItem(item); err != nil {
		return fmt.Errorf("error while add item '%s' into keychain: %v", label, err)
	}
	return nil
}

//getToken rerieves a CDS Context from keychain
// return true as it use the OS Keychain.
func (c CDSContext) getToken(contextName string) (string, error) {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(contextName)
	item.SetAccount(contextName)
	item.SetMatchLimit(keychain.MatchLimitOne)
	item.SetReturnData(true)
	results, err := keychain.QueryItem(item)
	if err != nil {
		return "", fmt.Errorf("error while retreiving context")
	}
	if len(results) != 1 {
		return "", fmt.Errorf("context not found in keychain")
	}

	return string(results[0].Data), nil
}
