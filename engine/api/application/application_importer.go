package application

import (
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Import is able to create a new application and all its components
func Import(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, app *sdk.Application, repomanager string, u *sdk.User, msgChan chan<- sdk.Message) error {
	doUpdate, erre := Exists(db, proj.Key, app.Name)
	if erre != nil {
		return sdk.WrapError(erre, "application.Import> Unable to check if application exists")
	}

	if doUpdate {
		oldApp, errlo := LoadByName(db, store, proj.Key, app.Name, u, LoadOptions.WithGroups, LoadOptions.WithKeys, LoadOptions.WithVariablesWithClearPassword)
		if errlo != nil {
			return sdk.WrapError(errlo, "application.Import> Unable to check if application exists")
		}
		//Delete all Variables
		if err := DeleteAllVariable(db, oldApp.ID); err != nil {
			return sdk.WrapError(err, "application.Import> Cannot delete application variable")
		}

		///Delete all Keys
		if err := DeleteAllApplicationKeys(db, oldApp.ID); err != nil {
			return sdk.WrapError(err, "application.Import")
		}

		//Delete groups
		if err := group.DeleteAllGroupFromApplication(db, oldApp.ID); err != nil {
			return sdk.WrapError(err, "application.Import> Unable to delete group")
		}

		app.ProjectID = oldApp.ProjectID
		app.ID = oldApp.ID

		//Save app in database
		if err := Update(db, store, app, u); err != nil {
			return sdk.WrapError(err, "application.Import> Unable to update application")
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

	//Inherit project groups if not provided
	if app.ApplicationGroups == nil {
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgAppGroupInheritPermission, app.Name)
		}
		app.ApplicationGroups = proj.ProjectGroups
	}

	if err := importVariables(db, store, proj, app, u, msgChan); err != nil {
		return err
	}

	//Insert group permission on application
	for i := range app.ApplicationGroups {
		//Load the group by name
		g, err := group.LoadGroup(db, app.ApplicationGroups[i].Group.Name)
		if err != nil {
			return err
		}
		log.Debug("application.Import> Insert group %d in application", g.ID)
		if err := AddGroup(db, store, proj, app, u, app.ApplicationGroups[i]); err != nil {
			return err
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgAppGroupSetPermission, g.Name, app.Name)
		}
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

		if len(app.Pipelines) > 0 {
			//Manage hook
			if _, err := hook.CreateHook(db, store, proj, repomanager, app.RepositoryFullname, app, &app.Pipelines[0].Pipeline); err != nil {
				return err
			}
			if msgChan != nil {
				msgChan <- sdk.NewMessage(sdk.MsgHookCreated, app.RepositoryFullname, app.Pipelines[0].Pipeline.Name)
			}
		}
	}

	//Manage keys
	for _, k := range app.Keys {
		k.ApplicationID = app.ID
		if err := InsertKey(db, &k); err != nil {
			return sdk.WrapError(err, "application.Import> Unable to insert key %s", k.Name)
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgAppKeyCreated, strings.ToUpper(k.Type), k.Name, app.Name)
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
