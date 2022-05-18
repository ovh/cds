package keys

import (
	"context"
	"io"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

type DecryptFunc func(context.Context, gorp.SqlExecutor, int64, string) (string, error)

// Parse and decrypts an exported key
func Parse(ctx context.Context, db gorp.SqlExecutor, projID int64, kname string, kval exportentities.KeyValue, decryptFunc DecryptFunc) (*sdk.Key, error) {
	k := new(sdk.Key)
	k.Type = sdk.KeyType(kval.Type)
	k.Name = kname
	if kval.Value != "" {
		privateKey, err := decryptFunc(ctx, db, projID, kval.Value)
		if err != nil {
			return nil, sdk.WrapError(err, "Unable to decrypt secret")
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
			pubBytes, errReadPub := io.ReadAll(pubReader)
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
			pubBytes, errReadPub := io.ReadAll(pubReader)
			if errReadPub != nil {
				return nil, sdk.WrapError(errReadPub, "keys.Parse> Unable to read ssh public key")
			}
			k.Public = string(pubBytes)
		default:
			return nil, sdk.WithStack(sdk.ErrUnknownKeyType)
		}
	} else if kval.Regen == nil || *kval.Regen == true {
		ktemp, err := GenerateKey(kname, k.Type)
		if err != nil {
			return nil, err
		}
		k = &ktemp
	} else {
		log.Debug(ctx, "keys.Parse> Skip key regeneration")
	}
	return k, nil
}
