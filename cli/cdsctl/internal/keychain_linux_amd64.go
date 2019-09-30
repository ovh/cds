// +build !nokeychain

package internal

import (
	"encoding/json"
	"fmt"

	"github.com/keybase/go-keychain/secretservice"
	dbus "github.com/keybase/go.dbus"
)

var keychainEnabled = true

//storeTokens store tokens into keychain
func storeTokens(contextName string, tokens ContextTokens) error {
	srv, err := secretservice.NewService()
	if err != nil {
		return fmt.Errorf("error while getting secret service: %v", err)
	}

	session, err := srv.OpenSession(secretservice.AuthenticationDHAES)
	if err != nil {
		return fmt.Errorf("error while opening session to secret service: %v", err)
	}

	collection := secretservice.DefaultCollection

	b, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("error while encoding tokens into keychain: %v", err)
	}

	secret, err := session.NewSecret(b)
	if err != nil {
		return fmt.Errorf("failed to prepare secret: %v", err)
	}

	if err = srv.Unlock([]dbus.ObjectPath{collection}); err != nil {
		return fmt.Errorf("failed to unlock secret service")
	}

	_, err = srv.CreateItem(collection, secretservice.NewSecretProperties(fmt.Sprintf("CDS-cdsctl/%s", contextName), map[string]string{"context-name": contextName}), secret, secretservice.ReplaceBehaviorReplace)
	if err != nil {
		return fmt.Errorf("failed to store new secret: %v", err)
	}
	return nil
}

//getTokens rerieves Context tokens from keychain
// return true as it use the OS Keychain.
func (c CDSContext) getTokens(contextName string) (*ContextTokens, error) {
	srv, err := secretservice.NewService()
	if err != nil {
		return nil, fmt.Errorf("error while getting secret service: %v", err)
	}

	session, err := srv.OpenSession(secretservice.AuthenticationDHAES)
	if err != nil {
		return nil, fmt.Errorf("error while opening session to secret service: %v", err)
	}

	collection := secretservice.DefaultCollection

	items, err := srv.SearchCollection(collection, map[string]string{"context-name": contextName})
	if err != nil {
		return nil, fmt.Errorf("failed to search secret from secret service: %v", err)
	}

	if len(items) != 1 {
		return nil, fmt.Errorf("context not found in keychain")
	}

	gotItem := items[0]
	content, err := srv.GetSecret(gotItem, *session)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret from secret service: %v", err)
	}

	tokens := &ContextTokens{}
	if err := json.Unmarshal(content, &tokens); err != nil {
		return nil, fmt.Errorf("error while unmarshal tokens from keychain: %v", err)
	}

	return tokens, nil
}
