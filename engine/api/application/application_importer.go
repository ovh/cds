package application

import (
	"context"
	"strings"

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// Import is able to create a new application and all its components
func Import(ctx context.Context, db gorpmapper.SqlExecutorWithTx, proj sdk.Project, app *sdk.Application, repomanager string, u sdk.Identifiable, msgChan chan<- sdk.Message) error {
	doUpdate, erre := Exists(db, proj.Key, app.Name)
	if erre != nil {
		return sdk.WrapError(erre, "application.Import> Unable to check if application exists")
	}

	if app.RepositoryStrategy.ConnectionType == "ssh" {
		app.RepositoryStrategy.User = ""
		app.RepositoryStrategy.Password = ""
	} else {
		app.RepositoryStrategy.SSHKey = ""
		app.RepositoryStrategy.SSHKeyContent = ""
	}

	if doUpdate {
		oldApp, err := LoadByName(ctx, db, proj.Key, app.Name, LoadOptions.WithKeys, LoadOptions.WithVariablesWithClearPassword, LoadOptions.WithClearDeploymentStrategies)
		if err != nil {
			return sdk.WrapError(err, "unable to load application by name: %s", app.Name)
		}
		//Delete all Variables
		if err := DeleteAllVariables(db, oldApp.ID); err != nil {
			return sdk.WrapError(err, "cannot delete application variable")
		}

		///Delete all Keys
		if err := DeleteAllApplicationKeys(db, oldApp.ID); err != nil {
			return sdk.WrapError(err, "application.Import")
		}

		app.ProjectID = oldApp.ProjectID
		app.ID = oldApp.ID

		//Save app in database
		if err := Update(ctx, db, app); err != nil {
			return sdk.WrapError(err, "Unable to update application")
		}

		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgAppUpdated, app.Name)
		}
	} else {
		//Save application in database
		if err := Insert(db, proj, app); err != nil {
			return err
		}

		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgAppCreated, app.Name)
		}
	}

	if err := importVariables(db, app, u, msgChan); err != nil {
		return err
	}

	//Set repositories manager
	app.VCSServer = repomanager
	if app.VCSServer != "" && app.RepositoryFullname != "" {
		if _, err := vcs.LoadVCSByProject(ctx, db, proj.Key, app.VCSServer); err != nil {
			return err
		}

		if err := repositoriesmanager.InsertForApplication(db, app); err != nil {
			return err
		}
	}

	//Manage keys
	for _, k := range app.Keys {
		k.ApplicationID = app.ID
		if err := InsertKey(db, &k); err != nil {
			return sdk.WrapError(err, "unable to insert key %s", k.Name)
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgAppKeyCreated, strings.ToUpper(k.Type.String()), k.Name, app.Name)
		}
	}

	//Set deployment strategies
	if err := DeleteAllDeploymentStrategies(db, app.ID); err != nil {
		return sdk.WrapError(err, "unable to delete deployment strategies")
	}

	for pfName, pfConfig := range app.DeploymentStrategies {
		pf, has := proj.GetIntegration(pfName)
		if !has {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "integration %s not found", pfName)
		}
		if err := SetDeploymentStrategy(db, proj.ID, app.ID, pf.IntegrationModelID, pfName, pfConfig); err != nil {
			return sdk.WrapError(err, "unable to set deployment strategy %s", pfName)
		}
	}

	return nil
}

// importVariables is able to create variable on an existing application
func importVariables(db gorpmapper.SqlExecutorWithTx, app *sdk.Application, u sdk.Identifiable, msgChan chan<- sdk.Message) error {
	for i := range app.Variables {
		newVar := &app.Variables[i]
		if !sdk.IsInArray(newVar.Type, sdk.AvailableVariableType) {
			return sdk.WithStack(sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid variable type %s", newVar.Type))
		}

		if err := InsertVariable(db, app.ID, newVar, u); err != nil {
			return sdk.WrapError(err, "importVariables> Cannot add variable %s in application %s", newVar.Name, app.Name)
		}
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgAppVariablesCreated, app.Name)
	}

	return nil
}
