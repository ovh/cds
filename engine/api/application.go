package api

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getApplicationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		withPermissions := r.FormValue("permission")
		withUsage := FormBool(r, "withUsage")
		withIcon := FormBool(r, "withIcon")

		var u = deprecatedGetUser(ctx)
		requestedUserName := r.Header.Get("X-Cds-Username")

		//A provider can make a call for a specific user
		if getProvider(ctx) != nil && requestedUserName != "" {
			var err error
			//Load the specific user
			u, err = user.LoadUserWithoutAuth(api.mustDB(), requestedUserName)
			if err != nil {
				if sdk.Cause(err) == sql.ErrNoRows {
					return sdk.ErrUserNotFound
				}
				return sdk.WrapError(err, "unable to load user '%s'", requestedUserName)
			}
			if err := loadUserPermissions(api.mustDB(), api.Cache, u); err != nil {
				return sdk.WrapError(err, "unable to load user '%s' permissions", requestedUserName)
			}
		}
		loadOpts := []application.LoadOptionFunc{}
		if withIcon {
			loadOpts = append(loadOpts, application.LoadOptions.WithIcon)
		}
		applications, err := application.LoadAll(api.mustDB(), api.Cache, projectKey, u, loadOpts...)
		if err != nil {
			return sdk.WrapError(err, "Cannot load applications from db")
		}

		if strings.ToUpper(withPermissions) == "W" {
			res := make([]sdk.Application, 0, len(applications))
			for _, a := range applications {
				if a.Permission >= permission.PermissionReadWriteExecute {
					res = append(res, a)
				}
			}
			applications = res
		}

		if withUsage {
			for i := range applications {
				usage, errU := loadApplicationUsage(api.mustDB(), projectKey, applications[i].Name)
				if errU != nil {
					return sdk.WrapError(errU, "getApplicationHandler> Cannot load application usage")
				}
				applications[i].Usage = &usage
			}
		}

		return service.WriteJSON(w, applications, http.StatusOK)
	}
}

func (api *API) getApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		withKeys := FormBool(r, "withKeys")
		withUsage := FormBool(r, "withUsage")
		withIcon := FormBool(r, "withIcon")
		withDeploymentStrategies := FormBool(r, "withDeploymentStrategies")
		withVulnerabilities := FormBool(r, "withVulnerabilities")

		loadOptions := []application.LoadOptionFunc{
			application.LoadOptions.WithVariables,
		}
		if withKeys {
			loadOptions = append(loadOptions, application.LoadOptions.WithKeys)
		}
		if withDeploymentStrategies {
			loadOptions = append(loadOptions, application.LoadOptions.WithDeploymentStrategies)
		}
		if withVulnerabilities {
			loadOptions = append(loadOptions, application.LoadOptions.WithVulnerabilities)
		}
		if withIcon {
			loadOptions = append(loadOptions, application.LoadOptions.WithIcon)
		}

		app, errApp := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, deprecatedGetUser(ctx), loadOptions...)
		if errApp != nil {
			return sdk.WrapError(errApp, "getApplicationHandler: Cannot load application %s for project %s from db", applicationName, projectKey)
		}

		if err := application.LoadGroupByApplication(api.mustDB(), app); err != nil {
			return sdk.WrapError(err, "Unable to load groups by application")
		}

		if withUsage {
			usage, errU := loadApplicationUsage(api.mustDB(), projectKey, applicationName)
			if errU != nil {
				return sdk.WrapError(errU, "getApplicationHandler> Cannot load application usage")
			}
			app.Usage = &usage
		}

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

// loadApplicationUsage return usage of application
func loadApplicationUsage(db gorp.SqlExecutor, projKey, appName string) (sdk.Usage, error) {
	usage := sdk.Usage{}

	wf, errW := workflow.LoadByApplicationName(db, projKey, appName)
	if errW != nil {
		return usage, sdk.WrapError(errW, "loadApplicationUsage> Cannot load workflows linked to application %s in project %s", appName, projKey)
	}
	usage.Workflows = wf

	// TODO: add usage of pipelines and environments

	return usage, nil
}

func (api *API) getApplicationVCSInfosHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]
		remote := r.FormValue("remote")

		proj, err := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load project %s from db", projectKey)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, deprecatedGetUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s for project %s from db", applicationName, projectKey)
		}

		resp := struct {
			Branches []sdk.VCSBranch `json:"branches,omitempty"`
			Remotes  []sdk.VCSRepo   `json:"remotes,omitempty"`
			Tags     []sdk.VCSTag    `json:"tags,omitempty"`
		}{}

		if app.RepositoryFullname == "" || app.VCSServer == "" {
			return service.WriteJSON(w, resp, http.StatusOK)
		}

		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
		client, erra := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, vcsServer)
		if erra != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getApplicationVCSInfosHandler> Cannot get client got %s %s : %s", projectKey, app.VCSServer, erra)
		}

		repositoryFullname := app.RepositoryFullname
		if remote != "" && remote != app.RepositoryFullname {
			repositoryFullname = remote
		}
		branches, errb := client.Branches(ctx, repositoryFullname)
		if errb != nil {
			return sdk.WrapError(errb, "getApplicationVCSInfosHandler> Cannot get branches from repository %s", repositoryFullname)
		}
		resp.Branches = branches

		tags, errt := client.Tags(ctx, repositoryFullname)
		if errt != nil {
			return sdk.WrapError(errt, "getApplicationVCSInfosHandler> Cannot get tags from repository %s", repositoryFullname)
		}
		resp.Tags = tags

		remotes, errR := client.ListForks(ctx, repositoryFullname)
		if errR != nil {
			return sdk.WrapError(errR, "getApplicationVCSInfosHandler> Cannot get remotes from repository %s", repositoryFullname)
		}
		resp.Remotes = remotes

		return service.WriteJSON(w, resp, http.StatusOK)
	}
}

func (api *API) addApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		proj, errl := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx))
		if errl != nil {
			return sdk.WrapError(errl, "addApplicationHandler> Cannot load %s", key)
		}

		var app sdk.Application
		if err := service.UnmarshalBody(r, &app); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(app.Name) {
			return sdk.WrapError(sdk.ErrInvalidApplicationPattern, "addApplicationHandler: Application name %s do not respect pattern %s", app.Name, sdk.NamePattern)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}

		defer tx.Rollback()

		if err := application.Insert(tx, api.Cache, proj, &app, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot insert pipeline")
		}

		if err := group.LoadGroupByProject(tx, proj); err != nil {
			return sdk.WrapError(err, "Cannot load group from project")
		}

		if err := application.AddGroup(tx, api.Cache, proj, &app, deprecatedGetUser(ctx), proj.ProjectGroups...); err != nil {
			return sdk.WrapError(err, "Cannot add groups on application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) deleteApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		proj, errP := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "deleteApplicationHandler> Cannot laod project")
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, deprecatedGetUser(ctx))
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrApplicationNotFound) {
				log.Warning("deleteApplicationHandler> Cannot load application %s: %s\n", applicationName, err)
			}
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot begin transaction")
		}
		defer tx.Rollback()

		err = application.DeleteApplication(tx, app.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot delete application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishDeleteApplication(proj.Key, *app, deprecatedGetUser(ctx))

		return nil
	}
}

func (api *API) cloneApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		proj, errProj := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
		if errProj != nil {
			return sdk.WrapError(sdk.ErrNoProject, "cloneApplicationHandler> Cannot load %s", projectKey)
		}

		envs, errE := environment.LoadEnvironments(api.mustDB(), projectKey, true, deprecatedGetUser(ctx))
		if errE != nil {
			return sdk.WrapError(errE, "cloneApplicationHandler> Cannot load Environments %s", projectKey)

		}
		proj.Environments = envs

		var newApp sdk.Application
		if err := service.UnmarshalBody(r, &newApp); err != nil {
			return err
		}

		appToClone, errApp := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, deprecatedGetUser(ctx), application.LoadOptions.Default, application.LoadOptions.WithGroups)
		if errApp != nil {
			return sdk.WrapError(errApp, "cloneApplicationHandler> Cannot load application %s", applicationName)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "cloneApplicationHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := cloneApplication(tx, api.Cache, proj, &newApp, appToClone, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot insert new application %s", newApp.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, newApp, http.StatusOK)
	}
}

// cloneApplication Clone an application with all her dependencies: pipelines, permissions, triggers
func cloneApplication(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, newApp *sdk.Application, appToClone *sdk.Application, u *sdk.User) error {
	newApp.ApplicationGroups = appToClone.ApplicationGroups

	// Create Application
	if err := application.Insert(db, store, proj, newApp, u); err != nil {
		return err
	}

	var variablesToDelete []string
	for _, v := range newApp.Variable {
		if v.Type == sdk.KeyVariable {
			variablesToDelete = append(variablesToDelete, fmt.Sprintf("%s.pub", v.Name))
		}
	}

	for _, vToDelete := range variablesToDelete {
		for i := range newApp.Variable {
			if vToDelete == newApp.Variable[i].Name {
				newApp.Variable = append(newApp.Variable[:i], newApp.Variable[i+1:]...)
				break
			}
		}
	}

	// Insert variable
	for _, v := range newApp.Variable {
		var errVar error
		// If variable is a key variable, generate a new one for this application
		if v.Type == sdk.KeyVariable {
			errVar = application.AddKeyPairToApplication(db, store, newApp, v.Name, u)
		} else {
			errVar = application.InsertVariable(db, store, newApp, v, u)
		}
		if errVar != nil {
			return errVar
		}
	}

	// Insert Permission
	return application.AddGroup(db, store, proj, newApp, u, newApp.ApplicationGroups...)
}

func (api *API) updateApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		p, errload := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx), project.LoadOptions.Default)
		if errload != nil {
			return sdk.WrapError(errload, "updateApplicationHandler> Cannot load project %s", projectKey)
		}

		app, errloadbyname := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, deprecatedGetUser(ctx), application.LoadOptions.Default)
		if errloadbyname != nil {
			return sdk.WrapError(errloadbyname, "updateApplicationHandler> Cannot load application %s", applicationName)
		}

		var appPost sdk.Application
		if err := service.UnmarshalBody(r, &appPost); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(appPost.Name) {
			return sdk.WrapError(sdk.ErrInvalidApplicationPattern, "updateApplicationHandler> Application name %s do not respect pattern %s", appPost.Name, sdk.NamePattern)
		}

		if appPost.RepositoryStrategy.Password != sdk.PasswordPlaceholder && appPost.RepositoryStrategy.Password != "" {
			if errP := application.EncryptVCSStrategyPassword(&appPost); errP != nil {
				return sdk.WrapError(errP, "updateApplicationHandler> Cannot encrypt password")
			}
		}
		if appPost.RepositoryStrategy.Password == sdk.PasswordPlaceholder {
			appPost.RepositoryStrategy.Password = app.RepositoryStrategy.Password
		}

		old := *app

		//Update name and Metadata
		app.Name = appPost.Name
		app.Description = appPost.Description
		if appPost.Icon != "" {
			app.Icon = appPost.Icon
		}
		app.Metadata = appPost.Metadata
		app.RepositoryStrategy = appPost.RepositoryStrategy
		app.RepositoryStrategy.SSHKeyContent = ""

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback()
		if err := application.Update(tx, api.Cache, app, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot delete application %s", applicationName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishUpdateApplication(p.Key, *app, old, deprecatedGetUser(ctx))

		return service.WriteJSON(w, app, http.StatusOK)

	}
}

func (api *API) postApplicationMetadataHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, deprecatedGetUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "postApplicationMetadataHandler")
		}
		oldApp := *app

		m := vars["metadata"]
		v, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "postApplicationMetadataHandler")
		}
		defer r.Body.Close()

		app.Metadata[m] = string(v)
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start tx")
		}
		defer tx.Rollback() // nolint

		if err := application.Update(tx, api.Cache, app, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "unable to update application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit tx")
		}

		event.PublishUpdateApplication(projectKey, *app, oldApp, deprecatedGetUser(ctx))

		return nil
	}
}
