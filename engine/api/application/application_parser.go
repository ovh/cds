package application

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// ParseAndImport parse an exportentities.Application and insert or update the application in database
func ParseAndImport(db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, eapp *exportentities.Application, force bool, decryptFunc keys.DecryptFunc, u *sdk.User) (*sdk.Application, []sdk.Message, error) {
	log.Info("ParseAndImport>> Import application %s in project %s (force=%v)", eapp.Name, proj.Key, force)

	//Check valid application name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(eapp.Name) {
		return nil, nil, sdk.WrapError(sdk.ErrInvalidApplicationPattern, "ParseAndImport>> Application name %s do not respect pattern %s", eapp.Name, sdk.NamePattern)
	}

	//Check if app exist
	oldApp, errl := LoadByName(db, cache, proj.Key, eapp.Name, nil, LoadOptions.WithVariablesWithClearPassword, LoadOptions.WithKeys, LoadOptions.WithClearDeploymentStrategies)
	if errl != nil && !sdk.ErrorIs(errl, sdk.ErrApplicationNotFound) {
		return nil, nil, sdk.WrapError(errl, "ParseAndImport>> Unable to load application")
	}

	//If the application exist and we don't want to force, raise an error
	if oldApp != nil && !force {
		return nil, nil, sdk.ErrApplicationExist
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
				return app, nil, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, err), "ParseAndImport>> Unable to decrypt secret variable")
			}
			v.Value = secret
		}

		vv := sdk.Variable{Name: p, Type: v.Type, Value: v.Value}
		app.Variable = append(app.Variable, vv)
	}

	//Compute keys
	for kname, kval := range eapp.Keys {
		if !strings.HasPrefix(kname, "app-") {
			return app, nil, sdk.WrapError(sdk.ErrInvalidKeyName, "ParseAndImport>> Unable to parse key %s", kname)
		}

		var oldKey *sdk.ApplicationKey
		var keepOldValue bool
		//If application doesn't exist, skip the regen mecanism to generate key
		if oldApp == nil {
			kval.Regen = nil
			log.Debug("ParseAndImport> Skipping regen feature")
		} else {
			//If application exist, check the key exist
			oldKey = oldApp.GetKey(kname)
			//If the key doesn't exist, skip the regen mecanism to generate key
			if oldKey == nil {
				kval.Regen = nil
				log.Debug("ParseAndImport> Skipping regen feature")
			}
		}

		if kval.Regen != nil && !*kval.Regen {
			keepOldValue = true
		}

		kk, err := keys.Parse(db, proj.ID, kname, kval, decryptFunc)
		if err != nil {
			return app, nil, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, err), "ParseAndImport>> Unable to parse key")
		}

		k := sdk.ApplicationKey{
			Key:           *kk,
			ApplicationID: app.ID,
		}

		if keepOldValue && oldKey != nil {
			k.Key = oldKey.Key
		}

		app.Keys = append(app.Keys, k)
	}

	// VCS Strategy
	app.RepositoryStrategy = sdk.RepositoryStrategy{
		ConnectionType: eapp.VCSConnectionType,
		User:           eapp.VCSUser,
		SSHKey:         eapp.VCSSSHKey,
		PGPKey:         eapp.VCSPGPKey,
	}
	if app.RepositoryStrategy.ConnectionType == "" {
		app.RepositoryStrategy.ConnectionType = "https"
	}
	if app.RepositoryStrategy.ConnectionType == "ssh" && app.RepositoryStrategy.SSHKey == "" {
		return app, nil, sdk.NewErrorFrom(sdk.ErrInvalidApplicationRepoStrategy, "Could not import application %s with a connection type ssh without ssh key", app.Name)
	}
	if eapp.VCSPassword != "" {
		clearPWD, err := decryptFunc(db, proj.ID, eapp.VCSPassword)
		if err != nil {
			return app, nil, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, err), "ParseAndImport> Unable to decrypt vcs password")
		}
		app.RepositoryStrategy.Password = clearPWD
		if errE := EncryptVCSStrategyPassword(app); errE != nil {
			return app, nil, sdk.WrapError(errE, "ParseAndImport> Cannot encrypt vcs password")
		}
	}

	//deployment strategies
	for pfName, pfConfig := range eapp.DeploymentStrategies {
		if app.DeploymentStrategies == nil {
			app.DeploymentStrategies = make(map[string]sdk.IntegrationConfig)
		}
		if app.DeploymentStrategies[pfName] == nil {
			app.DeploymentStrategies[pfName] = make(map[string]sdk.IntegrationConfigValue)
		}

		projPF, has := proj.GetIntegration(pfName)
		if !has {
			return app, nil, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("integration not found")), "ParseAndImport> Integration %s not found", pfName)
		}

		// Inherit from existing deployment strategy or from the project
		if oldApp != nil {
			oldPFConfig, has := oldApp.DeploymentStrategies[pfName]
			if has {
				app.DeploymentStrategies[pfName] = oldPFConfig.Clone()
			}
		} else {
			app.DeploymentStrategies[pfName] = projPF.Model.DeploymentDefaultConfig.Clone()
		}

		app.DeploymentStrategies[pfName].MergeWith(projPF.Model.DeploymentDefaultConfig.Clone())

		for k, v := range pfConfig {
			if v.Value != "" {
				if v.Type == sdk.SecretVariable {
					clearPWD, err := decryptFunc(db, proj.ID, v.Value)
					if err != nil {
						return app, nil, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, err), "ParseAndImport> Unable to decrypt deployment strategy password")
					}
					v.Value = clearPWD
				}
			}
			app.DeploymentStrategies[pfName][k] = sdk.IntegrationConfigValue{
				Type:  v.Type,
				Value: v.Value,
			}
		}
	}

	done := new(sync.WaitGroup)
	done.Add(1)
	msgChan := make(chan sdk.Message)
	msgList := []sdk.Message{}
	go func(array *[]sdk.Message) {
		defer done.Done()
		for m := range msgChan {
			*array = append(*array, m)
		}
	}(&msgList)

	globalError := Import(db, cache, proj, app, eapp.VCSServer, u, msgChan)
	close(msgChan)
	done.Wait()

	return app, msgList, globalError
}
