package vault

import (
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// vault initialization status
const (
	StatusNotInitialized = "Not initialized"
	StatusFakeClient     = "Fake client"
	StatusOK             = "OK"
	StatusKO             = "KO"
)

// Status of vault client
var Status string

func init() {
	Status = StatusNotInitialized
}

//Client is the Vault client interface
type Client interface {
	GetSecrets() (map[string]string, error)
}

//DefaultClient to Vault API
type DefaultClient struct {
	APIURL         string
	ApplicationKey string
	PlatformOTP    string
}

//GetSecrets returns all the secrets as a key/value map for the application namespace
func (c *DefaultClient) GetSecrets() (map[string]string, error) {
	return GetSecrets(c.APIURL, c.ApplicationKey, c.PlatformOTP)
}

//Ping checks canary
func (c *DefaultClient) Ping() error {
	key, value, err := GetSecret(c.APIURL, c.ApplicationKey, c.PlatformOTP, "cds/canary")
	if err != nil {
		Status = StatusKO
		return err
	}
	if key == "" || value == "" {
		log.Warning("vault.Ping> Unable to get cds/canary secret")
		Status = StatusKO
		return sdk.ErrSecretStoreUnreachable
	}

	Status = StatusOK
	return nil
}
