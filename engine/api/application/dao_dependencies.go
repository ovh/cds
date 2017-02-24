package application

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

var (
	loadDefaultDependencies = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		if err := loadVariables(db, app, u); err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadDefaultDependencies", app.Name)
		}
		if err := loadTriggers(db, app, u); err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadDefaultDependencies", app.Name)
		}
		if err := loadRepositoryManager(db, app, u); err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadDefaultDependencies", app.Name)
		}
		return nil
	}

	loadVariables = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		variables, err := GetAllVariableByID(db, app.ID)
		if err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadVariables> Unable to load variables for application %d", app.ID)
		}
		app.Variable = variables
		return nil
	}

	loadVariablesWithClearPassword = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		variables, err := GetAllVariableByID(db, app.ID, WithClearPassword())
		if err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadVariablesWithClearPassword> Unable to load variables for application %d", app.ID)
		}
		app.Variable = variables
		return nil
	}

	loadPipelines = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		pipelines, err := GetAllPipelinesByID(db, app.ID)
		if err != nil && err != sdk.ErrNoAttachedPipeline {
			return sdk.WrapError(err, "application.loadPipelines> Unable to load pipelines for application %d", app.ID)
		}
		app.Pipelines = pipelines
		return nil
	}

	loadTriggers = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		if app.Pipelines == nil {
			if err := loadPipelines(db, app, u); err != nil {
				return sdk.WrapError(err, "application.loadTriggers")
			}
		}
		for i := range app.Pipelines {
			appPip := &app.Pipelines[i]
			var err error
			appPip.Triggers, err = trigger.LoadTriggersByAppAndPipeline(db, app.ID, appPip.Pipeline.ID)
			if err != nil && err != sql.ErrNoRows {
				return sdk.WrapError(err, "application.loadTriggers> Unable to load trigger for application %d, pipeline %s(%d)", app.ID, appPip.Pipeline.Name, appPip.Pipeline.ID)
			}
		}
		return nil
	}

	loadRepositoryManager = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		if app.RepositoryFullname != "" {
			id, err := db.SelectNullInt("select repositories_manager_id from application where id = $1", app.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil
				}
				return sdk.WrapError(err, "application.loadPipelines> Unable to load repositories manager for application %d", app.ID)
			}
			if id.Valid {
				rm, err := repositoriesmanager.LoadByID(db, id.Int64)
				if err != nil {
					return sdk.WrapError(err, "application.loadPipelines> Unable to load repositories manager (%d) for application %d", id.Int64, app.ID)
				}
				app.RepositoriesManager = rm
			}
		}
		return nil
	}

	loadGroups = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		if err := LoadGroupByApplication(db, app); err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadGroups> Unable to load group permission for application %d", app.ID)
		}
		return nil
	}

	loadPermission = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		app.Permission = permission.ApplicationPermission(app.ID, u)
		return nil
	}

	loadHooks = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		h, err := hook.LoadApplicationHooks(db, app.ID)
		if err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadHooks> Unable to load hooks for application %d", app.ID)
		}
		app.Hooks = h
		return nil
	}

	loadNotifs = func(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
		var err error
		app.Notifications, err = notification.LoadAllUserNotificationSettings(db, app.ID)
		if err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadNotifs> Unable to load notifications for application %d", app.ID)
		}
		return nil
	}
)
