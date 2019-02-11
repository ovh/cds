package application

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
)

//Import is able to create a new application and all its components
func Import(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, app *sdk.Application, repomanager string, u *sdk.User, msgChan chan<- sdk.Message) error {
	doUpdate, erre := Exists(db, proj.Key, app.Name)
	if erre != nil {
		return sdk.WrapError(erre, "application.Import> Unable to check if application exists")
	}

	if doUpdate {
		oldApp, errlo := LoadByName(db, store, proj.Key, app.Name, LoadOptions.WithKeys, LoadOptions.WithVariablesWithClearPassword, LoadOptions.WithClearDeploymentStrategies)
		if errlo != nil {
			return sdk.WrapError(errlo, "application.Import> Unable to load application by name: %s", app.Name)
		}
		//Delete all Variables
		if err := DeleteAllVariable(db, oldApp.ID); err != nil {
			return sdk.WrapError(err, "Cannot delete application variable")
		}

		///Delete all Keys
		if err := DeleteAllApplicationKeys(db, oldApp.ID); err != nil {
			return sdk.WrapError(err, "application.Import")
		}

		app.ProjectID = oldApp.ProjectID
		app.ID = oldApp.ID

		//Save app in database
		if err := Update(db, store, app); err != nil {
			return sdk.WrapError(err, "Unable to update application")
		}

		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgAppUpdated, app.Name)
		}

	} else {
		//Save application in database
		if err := Insert(db, store, proj, app, u); err != nil {
			return sdk.WrapError(err, "application.Import")
		}

		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgAppCreated, app.Name)
		}
	}

	if err := importVariables(db, store, proj, app, u, msgChan); err != nil {
		return err
	}

	//Set repositories manager
	app.VCSServer = repomanager
	if app.VCSServer != "" && app.RepositoryFullname != "" {
		if repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer) == nil {
			return sdk.ErrNoReposManager
		}

		if err := repositoriesmanager.InsertForApplication(db, app, proj.Key); err != nil {
			return err
		}
	}

	//Manage keys
	for _, k := range app.Keys {
		k.ApplicationID = app.ID
		if err := InsertKey(db, &k); err != nil {
			return sdk.WrapError(err, "Unable to insert key %s", k.Name)
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgAppKeyCreated, strings.ToUpper(k.Type), k.Name, app.Name)
		}
	}

	//Set deployment strategies
	if err := DeleteAllDeploymentStrategies(db, app.ID); err != nil {
		return sdk.WrapError(err, "Unable to delete deployment strategies")
	}

	for pfName, pfConfig := range app.DeploymentStrategies {
		pf, has := proj.GetIntegration(pfName)
		if !has {
			return sdk.WrapError(sdk.NewError(sdk.ErrNotFound, fmt.Errorf("integration %s not found", pfName)), "application.Import")
		}
		if err := SetDeploymentStrategy(db, proj.ID, app.ID, pf.IntegrationModelID, pfName, pfConfig); err != nil {
			return sdk.WrapError(err, "unable to set deployment strategy %s", pfName)
		}
	}

	return nil
}

//importVariables is able to create variable on an existing application
func importVariables(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, app *sdk.Application, u *sdk.User, msgChan chan<- sdk.Message) error {
	for _, newVar := range app.Variable {
		var errCreate error
		switch newVar.Type {
		case sdk.KeyVariable:
			errCreate = AddKeyPairToApplication(db, store, app, newVar.Name, u)
			break
		default:
			errCreate = InsertVariable(db, store, app, newVar, u)
			break
		}
		if errCreate != nil {
			return sdk.WrapError(errCreate, "importVariables> Cannot add variable %s in application %s:  %s", newVar.Name, app.Name, errCreate)
		}
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgAppVariablesCreated, app.Name)
	}

	return nil
}
