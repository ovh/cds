package application

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// ImportOptions are options to import application
type ImportOptions struct {
	Force          bool
	FromRepository string
}

// ParseAndImport parse an exportentities.Application and insert or update the application in database
func ParseAndImport(ctx context.Context, db gorpmapper.SqlExecutorWithTx, cache cache.Store, proj sdk.Project, eapp *exportentities.Application, opts ImportOptions, decryptFunc keys.DecryptFunc, u sdk.Identifiable) (*sdk.Application, []sdk.Variable, []sdk.Message, error) {
	log.Info(ctx, "ParseAndImport>> Import application %s in project %s (force=%v)", eapp.Name, proj.Key, opts.Force)
	msgList := []sdk.Message{}

	//Check valid application name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(eapp.Name) {
		msgList = append(msgList, sdk.NewMessage(sdk.MsgWorkflowErrorBadApplicationName, eapp.Name))
		return nil, nil, msgList, sdk.WrapError(sdk.ErrInvalidApplicationPattern, "application name %s do not respect pattern %s", eapp.Name, sdk.NamePattern)
	}

	//Check if app exist
	oldApp, err := LoadByProjectKeyAndName(ctx, db, proj.Key, eapp.Name,
		LoadOptions.WithVariablesWithClearPassword,
		LoadOptions.WithClearKeys,
		LoadOptions.WithClearDeploymentStrategies,
	)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil, nil, msgList, sdk.WrapError(err, "unable to load application")
	}

	//If the application exist and we don't want to force, raise an error
	if oldApp != nil && !opts.Force {
		return nil, nil, msgList, sdk.WithStack(sdk.ErrApplicationExist)
	}

	if oldApp != nil && oldApp.FromRepository != "" && opts.FromRepository != oldApp.FromRepository {
		return nil, nil, msgList, sdk.NewErrorFrom(sdk.ErrApplicationAsCodeOverride, "unable to update existing ascode application from %s", oldApp.FromRepository)
	}

	//Craft the application
	app := new(sdk.Application)
	app.Name = eapp.Name
	app.VCSServer = eapp.VCSServer
	app.RepositoryFullname = eapp.RepositoryName
	app.FromRepository = opts.FromRepository

	applicationSecrets := make([]sdk.Variable, 0)

	//Compute variables
	for p, v := range eapp.Variables {
		switch v.Type {
		case "":
			v.Type = sdk.StringVariable
		case sdk.SecretVariable:
			secret, err := decryptFunc(db, proj.ID, v.Value)
			if err != nil {
				return app, nil, msgList, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, err), "unable to decrypt secret variable")
			}
			v.Value = secret
		}
		vv := sdk.ApplicationVariable{Name: p, Type: v.Type, Value: v.Value}
		app.Variables = append(app.Variables, vv)

		if v.Type == sdk.SecretVariable {
			applicationSecrets = append(applicationSecrets, sdk.Variable{
				Name:  fmt.Sprintf("cds.app.%s", vv.Name),
				Type:  v.Type,
				Value: vv.Value,
			})
		}
	}

	//Compute keys
	for kname, kval := range eapp.Keys {
		if !strings.HasPrefix(kname, "app-") {
			msgList = append(msgList, sdk.NewMessage(sdk.MsgWorkflowErrorUnknownKey, kname))
			return app, nil, msgList, sdk.WrapError(sdk.ErrInvalidKeyName, "unable to parse key %s", kname)
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
			return app, nil, msgList, sdk.ErrorWithFallback(err, sdk.ErrWrongRequest, "unable to parse key %s", kname)
		}

		k := sdk.ApplicationKey{
			Name:          kk.Name,
			Public:        kk.Public,
			Private:       kk.Private,
			KeyID:         kk.KeyID,
			Type:          kk.Type,
			ApplicationID: app.ID,
		}

		if keepOldValue && oldKey != nil {
			k.Name = oldKey.Name
			k.Public = oldKey.Public
			k.Private = oldKey.Private
			k.KeyID = oldKey.KeyID
			k.Type = oldKey.Type
			k.ApplicationID = oldKey.ApplicationID
		}

		app.Keys = append(app.Keys, k)
		applicationSecrets = append(applicationSecrets, sdk.Variable{
			Name:  fmt.Sprintf("cds.key.%s.priv", k.Name),
			Type:  string(k.Type),
			Value: k.Private,
		})
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
		return app, nil, msgList, sdk.NewErrorFrom(sdk.ErrInvalidApplicationRepoStrategy, "could not import application %s with a connection type ssh without ssh key", app.Name)
	}
	if eapp.VCSPassword != "" {
		clearPWD, err := decryptFunc(db, proj.ID, eapp.VCSPassword)
		if err != nil {
			return app, nil, msgList, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, err), "unable to decrypt vcs password")
		}
		app.RepositoryStrategy.Password = clearPWD
		applicationSecrets = append(applicationSecrets, sdk.Variable{
			Name:  "git.http.password",
			Type:  sdk.SecretVariable,
			Value: clearPWD,
		})
	}

	// deployment strategies
	deploymentStrategies := make(map[string]sdk.IntegrationConfig)
	for pfName, pfConfig := range eapp.DeploymentStrategies {
		// init deployment strategy from project if default exists
		projIt, has := proj.GetIntegration(pfName)
		if !has {
			msgList = append(msgList, sdk.NewMessage(sdk.MsgWorkflowErrorBadIntegrationName, pfName))
			return app, nil, msgList, sdk.WrapError(sdk.NewErrorFrom(sdk.ErrWrongRequest, "deployment platform not found"), "deployment platform %s not found", pfName)
		}
		if projIt.Model.DeploymentDefaultConfig != nil {
			deploymentStrategies[pfName] = projIt.Model.DeploymentDefaultConfig.Clone()
		} else {
			deploymentStrategies[pfName] = make(map[string]sdk.IntegrationConfigValue)
		}

		// merge deployment strategy with old application deployment strategy if exists
		if oldApp != nil {
			if oldItConfig, has := oldApp.DeploymentStrategies[pfName]; has {
				deploymentStrategies[pfName].MergeWith(oldItConfig)
			}
		}

		// update deployment strategy with given values from request
		for k, v := range pfConfig {
			if v.Value != "" {
				if v.Type == sdk.SecretVariable {
					clearPWD, err := decryptFunc(db, proj.ID, v.Value)
					if err != nil {
						return app, nil, nil, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, err), "unable to decrypt deployment strategy password")
					}
					v.Value = clearPWD
					applicationSecrets = append(applicationSecrets, sdk.Variable{
						Name:  fmt.Sprintf("%s:cds.integration.%s", pfName, k),
						Type:  sdk.SecretVariable,
						Value: clearPWD,
					})
				}
			}
			deploymentStrategies[pfName][k] = sdk.IntegrationConfigValue{
				Type:  v.Type,
				Value: v.Value,
			}
		}
	}
	app.DeploymentStrategies = deploymentStrategies

	done := new(sync.WaitGroup)
	done.Add(1)
	msgChan := make(chan sdk.Message)
	go func(array *[]sdk.Message) {
		defer done.Done()
		for m := range msgChan {
			*array = append(*array, m)
		}
	}(&msgList)

	globalError := Import(ctx, db, proj, app, eapp.VCSServer, u, msgChan)
	close(msgChan)
	done.Wait()

	return app, applicationSecrets, msgList, globalError
}
