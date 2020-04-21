package api

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/user"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getApplicationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		withUsage := FormBool(r, "withUsage")
		withIcon := FormBool(r, "withIcon")
		withPermissions := r.FormValue("permission")

		loadOpts := []application.LoadOptionFunc{}
		if withIcon {
			loadOpts = append(loadOpts, application.LoadOptions.WithIcon)
		}

		requestedUserName := r.Header.Get("X-Cds-Username")
		var requestedUser *sdk.AuthentifiedUser
		if requestedUserName != "" && isMaintainer(ctx) {
			var err error
			requestedUser, err = user.LoadByUsername(ctx, api.mustDB(), requestedUserName)
			if err != nil {
				if sdk.Cause(err) == sql.ErrNoRows {
					return sdk.WithStack(sdk.ErrUserNotFound)
				}
				return err
			}

			groups, err := group.LoadAllByUserID(context.TODO(), api.mustDB(), requestedUser.ID)
			if err != nil {
				return sdk.WrapError(err, "unable to load user '%s' groups", requestedUserName)
			}
			requestedUser.Groups = groups

			projPerms, err := permission.LoadProjectMaxLevelPermission(ctx, api.mustDB(), []string{projectKey}, requestedUser.GetGroupIDs())
			if err != nil {
				return err
			}
			if projPerms.Level(projectKey) < sdk.PermissionRead {
				return nil
			}
		}

		applications, err := application.LoadAll(api.mustDB(), projectKey, loadOpts...)
		if err != nil {
			return sdk.WrapError(err, "Cannot load applications from db")
		}

		if strings.ToUpper(withPermissions) == "W" {
			var groupIDs []int64
			if requestedUser != nil {
				groupIDs = requestedUser.GetGroupIDs()
			} else {
				groupIDs = getAPIConsumer(ctx).GetGroupIDs()
			}

			projectPerms, err := permission.LoadProjectMaxLevelPermission(ctx, api.mustDB(), []string{projectKey}, groupIDs)
			if err != nil {
				return err
			}
			res := make([]sdk.Application, 0, len(applications))
			for _, a := range applications {
				if projectPerms.Permissions(projectKey).Writable {
					res = append(res, a)
				}
			}
			applications = res
		}

		if withUsage {
			for i := range applications {
				usage, errU := loadApplicationUsage(ctx, api.mustDB(), projectKey, applications[i].Name)
				if errU != nil {
					return sdk.WrapError(errU, "getApplicationHandler> Cannot load application usage")
				}
				applications[i].Usage = &usage
			}
		}

		return service.WriteJSON(w, applications, http.StatusOK)
	}
}

func (api *API) getAsCodeApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		fromRepo := FormString(r, "repo")

		apps, err := application.LoadAsCode(api.mustDB(), projectKey, fromRepo)
		if err != nil {
			return sdk.WrapError(err, "cannot load application from repo %s for project %s from db", fromRepo, projectKey)
		}
		return service.WriteJSON(w, apps, http.StatusOK)
	}
}

func (api *API) getApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]

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

		app, errApp := application.LoadByName(api.mustDB(), projectKey, applicationName, loadOptions...)
		if errApp != nil {
			return sdk.WrapError(errApp, "getApplicationHandler: Cannot load application %s for project %s from db", applicationName, projectKey)
		}

		if withUsage {
			usage, errU := loadApplicationUsage(ctx, api.mustDB(), projectKey, applicationName)
			if errU != nil {
				return sdk.WrapError(errU, "getApplicationHandler> Cannot load application usage")
			}
			app.Usage = &usage
		}

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

// loadApplicationUsage return usage of application
func loadApplicationUsage(ctx context.Context, db gorp.SqlExecutor, projKey, appName string) (sdk.Usage, error) {
	usage := sdk.Usage{}

	wf, errW := workflow.LoadByApplicationName(ctx, db, projKey, appName)
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
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]
		remote := r.FormValue("remote")

		proj, err := project.Load(api.mustDB(), projectKey)
		if err != nil {
			return sdk.WrapError(err, "Cannot load project %s from db", projectKey)
		}

		app, err := application.LoadByName(api.mustDB(), projectKey, applicationName, application.LoadOptions.Default)
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

		vcsServer := repositoriesmanager.GetProjectVCSServer(*proj, app.VCSServer)
		client, erra := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, projectKey, vcsServer)
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
		key := vars[permProjectKey]

		proj, errl := project.Load(api.mustDB(), key)
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

		defer tx.Rollback() // nolint

		if err := application.Insert(tx, *proj, &app); err != nil {
			return sdk.WrapError(err, "Cannot insert pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishAddApplication(ctx, proj.Key, app, getAPIConsumer(ctx))

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) deleteApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]

		proj, errP := project.Load(api.mustDB(), projectKey)
		if errP != nil {
			return sdk.WrapError(errP, "deleteApplicationHandler> Cannot laod project")
		}

		app, err := application.LoadByName(api.mustDB(), projectKey, applicationName)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot begin transaction")
		}
		defer tx.Rollback() // nolint

		err = application.DeleteApplication(tx, app.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot delete application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishDeleteApplication(ctx, proj.Key, *app, getAPIConsumer(ctx))

		return nil
	}
}

func (api *API) cloneApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]

		proj, errProj := project.Load(api.mustDB(), projectKey)
		if errProj != nil {
			return sdk.WrapError(sdk.ErrNoProject, "cloneApplicationHandler> Cannot load %s", projectKey)
		}

		envs, errE := environment.LoadEnvironments(api.mustDB(), projectKey)
		if errE != nil {
			return sdk.WrapError(errE, "cloneApplicationHandler> Cannot load Environments %s", projectKey)

		}
		proj.Environments = envs

		var newApp sdk.Application
		if err := service.UnmarshalBody(r, &newApp); err != nil {
			return err
		}

		appToClone, errApp := application.LoadByName(api.mustDB(), projectKey, applicationName, application.LoadOptions.Default)
		if errApp != nil {
			return sdk.WrapError(errApp, "cloneApplicationHandler> Cannot load application %s", applicationName)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "cloneApplicationHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := cloneApplication(ctx, tx, api.Cache, *proj, &newApp, appToClone); err != nil {
			return sdk.WrapError(err, "Cannot insert new application %s", newApp.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, newApp, http.StatusOK)
	}
}

// cloneApplication Clone an application with all her dependencies: pipelines, permissions, triggers
func cloneApplication(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, newApp *sdk.Application, appToClone *sdk.Application) error {
	// Create Application
	if err := application.Insert(db, proj, newApp); err != nil {
		return err
	}

	var variablesToDelete []string
	for _, v := range newApp.Variables {
		if v.Type == sdk.KeyVariable {
			variablesToDelete = append(variablesToDelete, fmt.Sprintf("%s.pub", v.Name))
		}
	}

	for _, vToDelete := range variablesToDelete {
		for i := range newApp.Variables {
			if vToDelete == newApp.Variables[i].Name {
				newApp.Variables = append(newApp.Variables[:i], newApp.Variables[i+1:]...)
				break
			}
		}
	}

	// Insert variables
	for i := range newApp.Variables {
		newVar := &newApp.Variables[i]
		if !sdk.IsInArray(newVar.Type, sdk.AvailableVariableType) {
			return sdk.WithStack(sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid variable type %s", newVar.Type))
		}

		if err := application.InsertVariable(db, newApp.ID, newVar, getAPIConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "cloneApplication> Cannot add variable %s in application %s", newVar.Name, newApp.Name)
		}
	}

	event.PublishAddApplication(ctx, proj.Key, *newApp, getAPIConsumer(ctx))

	return nil
}

func (api *API) updateApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]

		p, err := project.Load(api.mustDB(), projectKey, project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", projectKey)
		}

		app, err := application.LoadByNameWithClearVCSStrategyPassword(api.mustDB(), projectKey, applicationName, application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "cannot load application %s", applicationName)
		}

		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
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
		defer tx.Rollback() // nolint
		if err := application.Update(tx, app); err != nil {
			return sdk.WrapError(err, "Cannot delete application %s", applicationName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishUpdateApplication(ctx, p.Key, *app, old, getAPIConsumer(ctx))

		return service.WriteJSON(w, app, http.StatusOK)

	}
}

func (api *API) postApplicationMetadataHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]

		app, err := application.LoadByName(api.mustDB(), projectKey, applicationName)
		if err != nil {
			return sdk.WrapError(err, "postApplicationMetadataHandler")
		}
		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
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

		if err := application.Update(tx, app); err != nil {
			return sdk.WrapError(err, "unable to update application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit tx")
		}

		event.PublishUpdateApplication(ctx, projectKey, *app, oldApp, getAPIConsumer(ctx))

		return nil
	}
}
