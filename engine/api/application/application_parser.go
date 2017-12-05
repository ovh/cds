package application

import (
	"io/ioutil"
	"regexp"
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

// ParseAndImport parse an exportentities.Application and insert or update the application in database
func ParseAndImport(db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, eapp *exportentities.Application, force bool, decryptFunc decryptFunc, u *sdk.User) ([]sdk.Message, error) {
	log.Info("ParseAndImport>> Import application %s in project %s (force=%v)", eapp.Name, proj.Key, force)
	log.Debug("ParseAndImport>> App: %+v", eapp)

	//Check valid application name
	rx := regexp.MustCompile(sdk.NamePattern)
	if !rx.MatchString(eapp.Name) {
		return nil, sdk.WrapError(sdk.ErrInvalidApplicationPattern, "ParseAndImport>> Application name %s do not respect pattern %s", eapp.Name, sdk.NamePattern)
	}

	//Check if app exist
	oldApp, errl := LoadByName(db, cache, proj.Key, eapp.Name, nil, LoadOptions.WithVariablesWithClearPassword, LoadOptions.WithKeys)
	if errl != nil && sdk.ErrorIs(errl, sdk.ErrApplicationNotFound) {
		return nil, sdk.WrapError(errl, "ParseAndImport>> Unable to load application")
	}

	//If the application exist and we don't want to force, raise an error
	if oldApp != nil && !force {
		return nil, sdk.ErrApplicationExist
	}

	//Craft the application
	app := new(sdk.Application)
	app.Name = eapp.Name
	app.VCSServer = eapp.VCSServer
	app.RepositoryFullname = eapp.RepositoryName

	//Inherit permissions from project
	if len(eapp.Permissions) == 0 {
		eapp.Permissions = make(map[string]int)
		for _, p := range proj.ProjectGroups {
			eapp.Permissions[p.Group.Name] = p.Permission
		}
	}

	//Compute permissions
	for g, p := range eapp.Permissions {
		perm := sdk.GroupPermission{Group: sdk.Group{Name: g}, Permission: p}
		app.ApplicationGroups = append(app.ApplicationGroups, perm)
	}

	//Compute variables
	for p, v := range eapp.Variables {
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
		app.Variable = append(app.Variable, vv)
	}

	//Compute keys
	for kname, kval := range eapp.Keys {
		k := sdk.ApplicationKey{
			Key: sdk.Key{
				Name: kname,
				Type: kval.Type,
			},
			ApplicationID: app.ID,
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
		app.Keys = append(app.Keys, k)

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

	globalError := Import(db, cache, proj, app, eapp.VCSServer, u, msgChan)
	close(msgChan)
	done.Wait()

	return msgList, globalError
}
