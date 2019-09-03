// +build !nokeychain

package internal

import (
	"fmt"

	"github.com/keybase/go-keychain/secretservice"
	dbus "github.com/keybase/go.dbus"
)

var keychainEnabled = true

//storeToken store a context into keychain
func storeToken(contextName, token string) error {
	srv, err := secretservice.NewService()
	if err != nil {
		return fmt.Errorf("error while getting secret service: %v", err)
	}

	session, err := srv.OpenSession(secretservice.AuthenticationDHAES)
	if err != nil {
		return fmt.Errorf("error while opening session to secret service: %v", err)
	}

	collection := secretservice.DefaultCollection

	secret, err := session.NewSecret([]byte(token))
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

//getToken rerieves a CDS Context from keychain
// return true as it use the OS Keychain.
func (c CDSContext) getToken(contextName string) (string, error) {
	srv, err := secretservice.NewService()
	if err != nil {
		return "", fmt.Errorf("error while getting secret service: %v", err)
	}

	session, err := srv.OpenSession(secretservice.AuthenticationDHAES)
	if err != nil {
		return "", fmt.Errorf("error while opening session to secret service: %v", err)
	}

	collection := secretservice.DefaultCollection

	items, err := srv.SearchCollection(collection, map[string]string{"context-name": contextName})
	if err != nil {
		return "", fmt.Errorf("failed to search secret from secret service: %v", err)
	}

	if len(items) != 1 {
		return "", fmt.Errorf("context not found in keychain")
	}

	gotItem := items[0]
	secretPlaintext, err := srv.GetSecret(gotItem, *session)
	if err != nil {
		return "", fmt.Errorf("failed to get secret from secret service: %v", err)
	}
	return string(secretPlaintext), nil
}
