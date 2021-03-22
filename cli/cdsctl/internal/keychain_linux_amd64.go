// +build !nokeychain

package internal

import (
	"encoding/json"
	"fmt"

	"github.com/keybase/go-keychain/secretservice"
	dbus "github.com/keybase/go.dbus"
	"github.com/ovh/cds/cli"
)

var keychainEnabled = true

//storeTokens store tokens into keychain
func storeTokens(contextName string, tokens ContextTokens) error {
	srv, err := secretservice.NewService()
	if err != nil {
		return cli.WrapError(err, "error while getting secret service")
	}

	session, err := srv.OpenSession(secretservice.AuthenticationDHAES)
	if err != nil {
		return cli.WrapError(err, "error while opening session to secret service")
	}

	collection := secretservice.DefaultCollection

	b, err := json.Marshal(tokens)
	if err != nil {
		return cli.WrapError(err, "error while encoding tokens into keychain")
	}

	secret, err := session.NewSecret(b)
	if err != nil {
		return cli.WrapError(err, "failed to prepare secret")
	}

	if err = srv.Unlock([]dbus.ObjectPath{collection}); err != nil {
		return cli.NewError("failed to unlock secret service")
	}

	_, err = srv.CreateItem(collection, secretservice.NewSecretProperties(fmt.Sprintf("CDS-cdsctl/%s", contextName), map[string]string{"context-name": contextName}), secret, secretservice.ReplaceBehaviorReplace)
	if err != nil {
		return cli.WrapError(err, "failed to store new secret")
	}
	return nil
}

//getTokens rerieves Context tokens from keychain
// return true as it use the OS Keychain.
func (c CDSContext) getTokens(contextName string) (*ContextTokens, error) {
	srv, err := secretservice.NewService()
	if err != nil {
		return nil, cli.WrapError(err, "error while getting secret service")
	}

	session, err := srv.OpenSession(secretservice.AuthenticationDHAES)
	if err != nil {
		return nil, cli.WrapError(err, "error while opening session to secret service")
	}

	collection := secretservice.DefaultCollection

	items, err := srv.SearchCollection(collection, map[string]string{"context-name": contextName})
	if err != nil {
		return nil, cli.WrapError(err, "failed to search secret from secret service")
	}

	if len(items) != 1 {
		return nil, cli.NewError("context not found in keychain")
	}

	gotItem := items[0]
	content, err := srv.GetSecret(gotItem, *session)
	if err != nil {
		return nil, cli.WrapError(err, "failed to get secret from secret service")
	}

	tokens := &ContextTokens{}
	if err := json.Unmarshal(content, &tokens); err != nil {
		return nil, cli.WrapError(err, "error while unmarshal tokens from keychain")
	}

	return tokens, nil
}
