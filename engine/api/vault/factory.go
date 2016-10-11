package vault

import (
	"os"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//GetClient is a Vault client factory
func GetClient(vaultAPI, vaultKey, vaultPassword, keysDirectory, secretsDir, httpTokenHeader string) (Client, error) {
	Status = StatusKO
	tokenHeader = httpTokenHeader
	if vaultPassword == "" || vaultPassword == "wrong" {
		//last chance !
		vaultPassword = os.Getenv("VAULT_PLATFORM_OTP")
	}
	if vaultKey == "" {
		return nil, sdk.ErrSecretStoreUnreachable
	}

	var vaultClient Client
	if vaultAPI == "local-insecure" {
		log.Warning("vault.GetClient> Connecting to fake Vault. /!\\ This is not secure\n")
		//In local-insecure mode, we will load secrets keys from a plain old json file
		localClient := &LocalInsecureClient{}
		if err := localClient.Load(secretsDir); err != nil {
			log.Warning("vault.GetClient>> Unable to load fake Vault secrets %s\n", err)
			return nil, sdk.ErrSecretStoreUnreachable
		}
		vaultClient = localClient
		Status = StatusFakeClient
	} else {
		log.Notice("vault.GetClient> Connecting to Vault %s\n", vaultAPI)
		vaultDefaultClient := &DefaultClient{}
		vaultDefaultClient.APIURL = vaultAPI
		vaultDefaultClient.ApplicationKey = vaultKey
		vaultDefaultClient.PlatformOTP = vaultPassword
		vaultClient = vaultDefaultClient
		//Check the canary value
		if err := vaultDefaultClient.Ping(); err != nil {
			return nil, err
		}
	}
	return vaultClient, nil
}
