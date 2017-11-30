package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postApplicationImportHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		force := FormBool(r, "force")

		body, errr := ioutil.ReadAll(r.Body)
		if errr != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errr)
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}

		var eapp = new(exportentities.Application)
		var errapp error
		switch contentType {
		case "application/json":
			errapp = json.Unmarshal(body, eapp)
		case "application/x-yaml", "text/x-yam":
			errapp = yaml.Unmarshal(body, eapp)
		default:
			return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("unsupported content-type: %s", contentType))
		}

		if errapp != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errapp)
		}

		log.Info("postApplicationImportHandler> Import application %s in project %s (force=%v)", eapp.Name, key, force)
		log.Debug("postApplicationImportHandler> App: %+v", eapp)

		//Check valid application name
		rx := regexp.MustCompile(sdk.NamePattern)
		if !rx.MatchString(eapp.Name) {
			return sdk.WrapError(sdk.ErrInvalidApplicationPattern, "postApplicationImportHandler> Application name %s do not respect pattern %s", eapp.Name, sdk.NamePattern)
		}

		//Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithGroups)
		if errp != nil {
			return sdk.WrapError(errp, "postApplicationImportHandler> Unable load project")
		}

		//Check if app exist
		oldApp, errl := application.LoadByName(api.mustDB(), api.Cache, key, eapp.Name, nil, application.LoadOptions.WithVariablesWithClearPassword, application.LoadOptions.WithKeys)
		if errl != nil && sdk.ErrorIs(errl, sdk.ErrApplicationNotFound) {
			return sdk.WrapError(errl, "postApplicationImportHandler> Unable to load application")
		}

		//If the application exist and we don't want to force, raise an error
		if oldApp != nil && !force {
			return sdk.ErrApplicationExist
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
				secret, err := project.DecryptWithBuiltinKey(api.mustDB(), proj.ID, v.Value)
				if err != nil {
					return sdk.WrapError(err, "postApplicationImportHandler> Unable to decrypt secret variable")
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
				privateKey, err := project.DecryptWithBuiltinKey(api.mustDB(), proj.ID, kval.Value)
				if err != nil {
					return sdk.WrapError(err, "postApplicationImportHandler> Unable to decrypt secret")
				}
				k.Private = privateKey

				switch k.Type {

				//Compute PGP Keys
				case sdk.KeyTypePgp:
					pgpEntity, errPGPEntity := keys.GetOpenPGPEntity(strings.NewReader(k.Private))
					if errPGPEntity != nil {
						return sdk.WrapError(errPGPEntity, "postApplicationImportHandler> Unable to read PGP Entity from private key")
					}
					pubReader, errPub := keys.GeneratePGPPublicKey(pgpEntity)
					if errPub != nil {
						return sdk.WrapError(errPub, "postApplicationImportHandler> Unable to generate pgp public key")
					}
					pubBytes, errReadPub := ioutil.ReadAll(pubReader)
					if errReadPub != nil {
						return sdk.WrapError(errReadPub, "postApplicationImportHandler> Unable to read pgp public key")
					}
					k.Public = string(pubBytes)

				//Compute SSH Keys
				case sdk.KeyTypeSsh:
					privKey, errPrivKey := keys.GetSSHPrivateKey(strings.NewReader(privateKey))
					if errPrivKey != nil {
						return sdk.WrapError(errPrivKey, "postApplicationImportHandler> Unable to read RSA private key")
					}
					pubReader, errPub := keys.GetSSHPublicKey(kname, privKey)
					if errPub != nil {
						return sdk.WrapError(errPub, "postApplicationImportHandler> Unable to generate ssh public key")
					}
					pubBytes, errReadPub := ioutil.ReadAll(pubReader)
					if errReadPub != nil {
						return sdk.WrapError(errReadPub, "postApplicationImportHandler> Unable to read ssh public key")
					}
					k.Public = string(pubBytes)
				default:
					return sdk.ErrUnknownKeyType
				}
			} else {
				switch k.Type {
				//Compute PGP Keys
				case sdk.KeyTypePgp:
					_, pub, priv, err := keys.GeneratePGPKeyPair(kname)
					if err != nil {
						return sdk.WrapError(err, "postApplicationImportHandler> Unable to generate PGP key pair")
					}
					k.Private = priv
					k.Public = pub
				//Compute SSH Keys
				case sdk.KeyTypeSsh:
					pub, priv, err := keys.GenerateSSHKeyPair(kname)
					if err != nil {
						return sdk.WrapError(err, "postApplicationImportHandler> Unable to generate SSH key pair")
					}
					k.Private = priv
					k.Public = pub
				default:
					return sdk.ErrUnknownKeyType
				}
			}
			app.Keys = append(app.Keys, k)

		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "postApplicationImportHandler> Unable to start transaction")

		}
		defer tx.Rollback()

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

		globalError := application.Import(tx, api.Cache, proj, app, eapp.VCSServer, getUser(ctx), msgChan)
		close(msgChan)
		done.Wait()
		msgListString := translate(r, msgList)

		if globalError != nil {
			myError, ok := globalError.(sdk.Error)
			if ok {
				return WriteJSON(w, r, msgListString, myError.Status)
			}
			return sdk.WrapError(globalError, "postApplicationImportHandler> Unable import application %s", eapp.Name)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj, sdk.ProjectPipelineLastModificationType); err != nil {
			return sdk.WrapError(err, "postApplicationImportHandler> Unable to update project")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postApplicationImportHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, msgListString, http.StatusCreated)
	}
}
