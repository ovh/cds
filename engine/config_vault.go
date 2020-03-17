package main

import (
	"context"
	"fmt"

	vault "github.com/hashicorp/vault/api"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type VaultSecret struct {
	Token  string
	Client *vault.Client
}

// Create new secret client
func VaultNewSecret(token, addr string) (*VaultSecret, error) {
	client, err := vault.NewClient(vault.DefaultConfig())
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	client.SetToken(token)
	client.SetAddress(addr)
	return &VaultSecret{
		Client: client,
		Token:  token,
	}, nil
}

// GetFromVault Get secret from vault
func (secret *VaultSecret) GetFromVault(s string) (string, error) {
	conf, err := secret.Client.Logical().Read(s)
	if err != nil {
		return "", sdk.WithStack(err)
	} else if conf == nil {
		log.Warning(context.Background(), "vault> no value found at %q", s)
		return "", nil
	}

	value, exists := conf.Data["data"]
	if !exists {
		log.Warning(context.Background(), "vault> no 'data' field found for %q (you must add a field with a key named data)", s)
		return "", nil
	}

	return fmt.Sprintf("%v", value), nil
}
