//go:build !nokeychain
// +build !nokeychain

package internal

import (
	"encoding/json"
	"fmt"

	keychain "github.com/keybase/go-keychain"
	"github.com/ovh/cds/cli"
)

var keychainEnabled = true

// storeTokens store tokens into keychain
func storeTokens(contextName string, tokens ContextTokens) error {
	// delete existing value if present
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(contextName)
	item.SetAccount(contextName)
	keychain.DeleteItem(item) // nolint: errcheck

	// populate the rest of the object and save
	label := fmt.Sprintf("CDS-cdsctl/%s", contextName)
	item.SetLabel(label)

	b, err := json.Marshal(tokens)
	if err != nil {
		return cli.WrapError(err, "error while encoding tokens into keychain")
	}

	item.SetData(b)
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleAccessibleAlwaysThisDeviceOnly)

	if err := keychain.AddItem(item); err != nil {
		return cli.WrapError(err, "error while add item '%s' into keychain", label)
	}
	return nil
}

// getTokens rerieves a CDS Context from keychain
func (c CDSContext) getTokens(contextName string) (*ContextTokens, error) {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(contextName)
	item.SetAccount(contextName)
	item.SetMatchLimit(keychain.MatchLimitOne)
	item.SetReturnData(true)
	results, err := keychain.QueryItem(item)
	if err != nil {
		return nil, cli.WrapError(err, "error while retrieving context")
	}
	if len(results) != 1 {
		return nil, cli.NewError("context not found in keychain")
	}

	tokens := &ContextTokens{}
	if err := json.Unmarshal(results[0].Data, &tokens); err != nil {
		return nil, cli.WrapError(err, "error while unmarshal tokens from keychain")
	}
	return tokens, nil
}
