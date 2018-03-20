package keys

import (
	"io/ioutil"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

type DecryptFunc func(gorp.SqlExecutor, int64, string) (string, error)

// Parse and decrypts an exported key
func Parse(db gorp.SqlExecutor, projID int64, kname string, kval exportentities.KeyValue, decryptFunc DecryptFunc) (*sdk.Key, error) {
	k := new(sdk.Key)
	k.Type = kval.Type
	k.Name = kname
	if kval.Value != "" {
		privateKey, err := decryptFunc(db, projID, kval.Value)
		if err != nil {
			return nil, sdk.WrapError(err, "keys.Parse> Unable to decrypt secret")
		}
		k.Private = privateKey

		switch k.Type {
		//Compute PGP Keys
		case sdk.KeyTypePGP:
			pgpEntity, errPGPEntity := GetOpenPGPEntity(strings.NewReader(k.Private))
			if errPGPEntity != nil {
				return nil, sdk.WrapError(errPGPEntity, "keys.Parse> Unable to read PGP Entity from private key")
			}
			pubReader, errPub := generatePGPPublicKey(pgpEntity)
			if errPub != nil {
				return nil, sdk.WrapError(errPub, "keys.Parse> Unable to generate pgp public key")
			}
			pubBytes, errReadPub := ioutil.ReadAll(pubReader)
			if errReadPub != nil {
				return nil, sdk.WrapError(errReadPub, "keys.Parse> Unable to read pgp public key")
			}
			k.Public = string(pubBytes)
			k.KeyID = pgpEntity.PrimaryKey.KeyIdShortString()
		//Compute SSH Keys
		case sdk.KeyTypeSSH:
			privKey, errPrivKey := getSSHPrivateKey(strings.NewReader(privateKey))
			if errPrivKey != nil {
				return nil, sdk.WrapError(errPrivKey, "keys.Parse> Unable to read RSA private key")
			}
			pubReader, errPub := getSSHPublicKey(kname, privKey)
			if errPub != nil {
				return nil, sdk.WrapError(errPub, "keys.Parse> Unable to generate ssh public key")
			}
			pubBytes, errReadPub := ioutil.ReadAll(pubReader)
			if errReadPub != nil {
				return nil, sdk.WrapError(errReadPub, "keys.Parse> Unable to read ssh public key")
			}
			k.Public = string(pubBytes)
		default:
			return nil, sdk.ErrUnknownKeyType
		}
	} else if kval.Regen == nil || *kval.Regen == true {
		switch k.Type {
		//Compute PGP Keys
		case sdk.KeyTypePGP:
			ktemp, err := GeneratePGPKeyPair(kname)
			if err != nil {
				return nil, sdk.WrapError(err, "keys.Parse> Unable to generate PGP key pair")
			}
			k = &ktemp
		//Compute SSH Keys
		case sdk.KeyTypeSSH:
			ktemp, err := GenerateSSHKey(kname)
			if err != nil {
				return nil, sdk.WrapError(err, "keys.Parse> Unable to generate SSH key pair")
			}
			k = &ktemp
		default:
			return nil, sdk.ErrUnknownKeyType
		}
	} else {
		log.Debug("keys.Parse> Skip key regeneration")
	}
	return k, nil
}
