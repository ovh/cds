package application

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

var (
	loadDefaultDependencies = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		if err := loadVariables(db, app, u); err != nil {
			return err
		}
		if err := loadTriggers(db, app, u); err != nil {
			return err
		}
		if err := loadRepositoryManager(db, app, u); err != nil {
			return err
		}
		return nil
	}

	loadVariables = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		variables, err := GetAllVariableByID(db, app.ID)
		if err != nil {
			return err
		}
		app.Variable = variables
		return nil
	}

	loadPipelines = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		pipelines, err := GetAllPipelinesByID(db, app.ID)
		if err != nil {
			return err
		}
		app.Pipelines = pipelines
		return nil
	}

	loadTriggers = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		if app.Pipelines == nil {
			if err := loadPipelines(db, app, u); err != nil {
				return err
			}
		}
		for i := range app.Pipelines {
			appPip := &app.Pipelines[i]
			var err error
			appPip.Triggers, err = trigger.LoadTriggersByAppAndPipeline(db, app.ID, appPip.Pipeline.ID)
			if err != nil {
				return err
			}
		}
		return nil
	}

	loadRepositoryManager = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		if app.RepositoryFullname != "" {
			id, err := db.SelectNullInt("select repositories_manager_id from application where id = $1", app.ID)
			if err != nil {
				return err
			}
			if id.Valid {
				rm, err := repositoriesmanager.LoadByID(db, id.Int64)
				if err != nil {
					return err
				}
				app.RepositoriesManager = rm
			}
		}
		return nil
	}

	loadGroups = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		return LoadGroupByApplication(db, app)
	}

	loadPermission = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		return nil
	}

	loadPollers = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		return nil
	}

	loadSchedulers = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		return nil
	}

	loadHooks = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		return nil
	}

	loadNotifs = func(gorp.SqlExecutor, *sdk.Application, *sdk.User) error {
		return nil
	}
)
