package application

import (
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
func ParseAndImport(db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, eapp *exportentities.Application, force bool, decryptFunc keys.DecryptFunc, u *sdk.User) ([]sdk.Message, error) {
	log.Info("ParseAndImport>> Import application %s in project %s (force=%v)", eapp.Name, proj.Key, force)
	log.Debug("ParseAndImport>> App: %+v", eapp)

	//Check valid application name
	rx := sdk.NamePatternRegex
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
		if !strings.HasPrefix(kname, "app-") {
			return nil, sdk.WrapError(sdk.ErrInvalidKeyName, "ParseAndImport>> Unable to parse key %s", kname)
		}

		kk, err := keys.Parse(db, proj.ID, kname, kval, decryptFunc)
		if err != nil {
			return nil, sdk.WrapError(err, "ParseAndImport>> Unable to parse key")
		}

		k := sdk.ApplicationKey{
			Key:           *kk,
			ApplicationID: app.ID,
		}

		app.Keys = append(app.Keys, k)
	}

	// VCS Strategy
	app.RepositoryStrategy = sdk.RepositoryStrategy{
		ConnectionType: eapp.VCSConnectionType,
		User:           eapp.VCSUser,
		SSHKey:         eapp.VCSSSHKey,
		PGPKey:         eapp.VCSPGPKey,
		DefaultBranch:  eapp.VCSDefaultBranch,
		Branch:         eapp.VCSBranch,
	}
	if app.RepositoryStrategy.ConnectionType == "" {
		app.RepositoryStrategy.ConnectionType = "https"
	}
	if eapp.VCSPassword != "" {
		clearPWD, err := decryptFunc(db, proj.ID, eapp.VCSPassword)
		if err != nil {
			return nil, sdk.WrapError(err, "ParseAndImport> Unable to decrypt vcs password")
		}
		app.RepositoryStrategy.Password = clearPWD
		if errE := EncryptVCSStrategyPassword(app); errE != nil {
			return nil, sdk.WrapError(errE, "ParseAndImport> Cannot encrypt vcs password")
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

	return msgList, globalError
}
