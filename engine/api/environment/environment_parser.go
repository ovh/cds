package environment

import (
	"io/ioutil"
	"strings"
	"sync"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

type decryptFunc func(gorp.SqlExecutor, int64, string) (string, error)

// ParseAndImport parse an exportentities.Environment and insert or update the environment in database
func ParseAndImport(db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, eenv *exportentities.Environment, force bool, decryptFunc decryptFunc, u *sdk.User) ([]sdk.Message, error) {
	log.Info("ParseAndImport>> Import environment %s in project %s (force=%v)", eenv.Name, proj.Key, force)
	log.Debug("ParseAndImport>> App: %+v", eenv)

	//Check valid application name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(eenv.Name) {
		return nil, sdk.WrapError(sdk.ErrInvalidApplicationPattern, "ParseAndImport>> Application name %s do not respect pattern %s", eenv.Name, sdk.NamePattern)
	}

	//Check if app exist
	oldEnv, errl := LoadEnvironmentByName(db, proj.Key, eenv.Name)
	if errl != nil && sdk.ErrorIs(errl, sdk.ErrNoEnvironment) {
		return nil, sdk.WrapError(errl, "ParseAndImport>> Unable to load application")
	}

	//If the environment exist and we don't want to force, raise an error
	var exist bool
	if oldEnv != nil && !force {
		return nil, sdk.ErrEnvironmentExist
	}
	if oldEnv != nil {
		exist = true
	}

	//Inherit permissions from project
	if len(eenv.Permissions) == 0 {
		eenv.Permissions = make(map[string]int)
		for _, p := range proj.ProjectGroups {
			eenv.Permissions[p.Group.Name] = p.Permission
		}
	}

	env := new(sdk.Environment)

	//Compute permissions
	for g, p := range eenv.Permissions {
		perm := sdk.GroupPermission{Group: sdk.Group{Name: g}, Permission: p}
		env.EnvironmentGroups = append(env.EnvironmentGroups, perm)
	}

	//Compute variables
	for p, v := range eenv.Values {
		switch v.Type {
		case "":
			v.Type = sdk.StringVariable
		case sdk.SecretVariable:
			secret, err := decryptFunc(db, proj.ID, v.Value)
			if err != nil {
				return nil, sdk.WrapError(err, "ParseAndImport>> Unable to decrypt secret variable")
			}
			v.Value = secret
		}

		vv := sdk.Variable{Name: p, Type: v.Type, Value: v.Value}
		env.Variable = append(env.Variable, vv)
	}

	//Compute keys
	for kname, kval := range eenv.Keys {
		k := sdk.EnvironmentKey{
			Key: sdk.Key{
				Name: kname,
				Type: kval.Type,
			},
			EnvironmentID: env.ID,
		}

		if kval.Value != "" {
			privateKey, err := decryptFunc(db, proj.ID, kval.Value)
			if err != nil {
				return nil, sdk.WrapError(err, "ParseAndImport>> Unable to decrypt secret")
			}
			k.Private = privateKey

			switch k.Type {
			//Compute PGP Keys
			case sdk.KeyTypePgp:
				pgpEntity, errPGPEntity := keys.GetOpenPGPEntity(strings.NewReader(k.Private))
				if errPGPEntity != nil {
					return nil, sdk.WrapError(errPGPEntity, "ParseAndImport>> Unable to read PGP Entity from private key")
				}
				pubReader, errPub := keys.GeneratePGPPublicKey(pgpEntity)
				if errPub != nil {
					return nil, sdk.WrapError(errPub, "ParseAndImport>> Unable to generate pgp public key")
				}
				pubBytes, errReadPub := ioutil.ReadAll(pubReader)
				if errReadPub != nil {
					return nil, sdk.WrapError(errReadPub, "ParseAndImport>> Unable to read pgp public key")
				}
				k.Public = string(pubBytes)
				k.KeyID = pgpEntity.PrimaryKey.KeyIdShortString()
			//Compute SSH Keys
			case sdk.KeyTypeSsh:
				privKey, errPrivKey := keys.GetSSHPrivateKey(strings.NewReader(privateKey))
				if errPrivKey != nil {
					return nil, sdk.WrapError(errPrivKey, "ParseAndImport>> Unable to read RSA private key")
				}
				pubReader, errPub := keys.GetSSHPublicKey(kname, privKey)
				if errPub != nil {
					return nil, sdk.WrapError(errPub, "ParseAndImport>> Unable to generate ssh public key")
				}
				pubBytes, errReadPub := ioutil.ReadAll(pubReader)
				if errReadPub != nil {
					return nil, sdk.WrapError(errReadPub, "ParseAndImport>> Unable to read ssh public key")
				}
				k.Public = string(pubBytes)
			default:
				return nil, sdk.ErrUnknownKeyType
			}
		} else {
			switch k.Type {
			//Compute PGP Keys
			case sdk.KeyTypePgp:
				id, pubR, privR, err := keys.GeneratePGPKeyPair(kname)
				if err != nil {
					return nil, sdk.WrapError(err, "ParseAndImport>> Unable to generate PGP key pair")
				}
				pub, errPub := ioutil.ReadAll(pubR)
				if errPub != nil {
					return nil, sdk.WrapError(errPub, "ParseAndImport>> Unable to read public key")
				}

				priv, errPriv := ioutil.ReadAll(privR)
				if errPriv != nil {
					return nil, sdk.WrapError(errPriv, "ParseAndImport>t>  Unable to read private key")
				}
				k.KeyID = id
				k.Private = string(priv)
				k.Public = string(pub)
			//Compute SSH Keys
			case sdk.KeyTypeSsh:
				pubR, privR, err := keys.GenerateSSHKeyPair(kname)
				if err != nil {
					return nil, sdk.WrapError(err, "ParseAndImport>> Unable to generate SSH key pair")
				}
				pub, errPub := ioutil.ReadAll(pubR)
				if errPub != nil {
					return nil, sdk.WrapError(errPub, "ParseAndImport>> Unable to read public key")
				}

				priv, errPriv := ioutil.ReadAll(privR)
				if errPriv != nil {
					return nil, sdk.WrapError(errPriv, "ParseAndImport>t>  Unable to read private key")
				}
				k.Private = string(priv)
				k.Public = string(pub)
			default:
				return nil, sdk.ErrUnknownKeyType
			}
		}
		env.Keys = append(env.Keys, k)
	}

	done := new(sync.WaitGroup)
	done.Add(1)
	msgChan := make(chan sdk.Message)
	msgList := []sdk.Message{}
	go func(array *[]sdk.Message) {
		defer done.Done()
		for {
			m, more := <-msgChan
			if !more {
				return
			}
			*array = append(*array, m)
		}
	}(&msgList)

	var globalError error

	if exist {
		globalError = ImportInto(db, proj, env, oldEnv, msgChan, u)
	} else {
		globalError = Import(db, proj, env, msgChan, u)
	}

	close(msgChan)
	done.Wait()

	return msgList, globalError
}
