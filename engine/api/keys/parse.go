package keys

import (
	"io/ioutil"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

type DecryptFunc func(gorp.SqlExecutor, int64, string) (string, error)

// Parse and decrypts an exported key
func Parse(db gorp.SqlExecutor, projID int64, kname string, kval exportentities.VariableValue, decryptFunc DecryptFunc) (*sdk.Key, error) {
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
		case sdk.KeyTypePgp:
			pgpEntity, errPGPEntity := GetOpenPGPEntity(strings.NewReader(k.Private))
			if errPGPEntity != nil {
				return nil, sdk.WrapError(errPGPEntity, "keys.Parse> Unable to read PGP Entity from private key")
			}
			pubReader, errPub := GeneratePGPPublicKey(pgpEntity)
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
		case sdk.KeyTypeSsh:
			privKey, errPrivKey := GetSSHPrivateKey(strings.NewReader(privateKey))
			if errPrivKey != nil {
				return nil, sdk.WrapError(errPrivKey, "keys.Parse> Unable to read RSA private key")
			}
			pubReader, errPub := GetSSHPublicKey(kname, privKey)
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
	} else {
		switch k.Type {
		//Compute PGP Keys
		case sdk.KeyTypePgp:
			id, pubR, privR, err := GeneratePGPKeyPair(kname)
			if err != nil {
				return nil, sdk.WrapError(err, "keys.Parse> Unable to generate PGP key pair")
			}
			pub, errPub := ioutil.ReadAll(pubR)
			if errPub != nil {
				return nil, sdk.WrapError(errPub, "keys.Parse> Unable to read public key")
			}

			priv, errPriv := ioutil.ReadAll(privR)
			if errPriv != nil {
				return nil, sdk.WrapError(errPriv, "keys.Parse>t>  Unable to read private key")
			}
			k.KeyID = id
			k.Private = string(priv)
			k.Public = string(pub)
		//Compute SSH Keys
		case sdk.KeyTypeSsh:
			pubR, privR, err := GenerateSSHKeyPair(kname)
			if err != nil {
				return nil, sdk.WrapError(err, "keys.Parse> Unable to generate SSH key pair")
			}
			pub, errPub := ioutil.ReadAll(pubR)
			if errPub != nil {
				return nil, sdk.WrapError(errPub, "keys.Parse> Unable to read public key")
			}

			priv, errPriv := ioutil.ReadAll(privR)
			if errPriv != nil {
				return nil, sdk.WrapError(errPriv, "keys.Parse>t>  Unable to read private key")
			}
			k.Private = string(priv)
			k.Public = string(pub)
		default:
			return nil, sdk.ErrUnknownKeyType
		}
	}
	return k, nil
}
