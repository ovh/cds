package api

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) getApplicationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		withUsage := service.FormBool(r, "withUsage")
		withIcon := service.FormBool(r, "withIcon")
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

		applications, err := application.LoadAll(ctx, api.mustDB(), projectKey, loadOpts...)
		if err != nil {
			return sdk.WrapError(err, "cannot load applications from db")
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

func (api *API) getApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]

		withKeys := service.FormBool(r, "withKeys")
		withUsage := service.FormBool(r, "withUsage")
		withIcon := service.FormBool(r, "withIcon")
		withDeploymentStrategies := service.FormBool(r, "withDeploymentStrategies")
		withVulnerabilities := service.FormBool(r, "withVulnerabilities")

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

		app, err := application.LoadByName(ctx, api.mustDB(), projectKey, applicationName, loadOptions...)
		if err != nil {
			return sdk.WrapError(err, "cannot load application %s for project %s from db", applicationName, projectKey)
		}

		if withUsage {
			usage, err := loadApplicationUsage(ctx, api.mustDB(), projectKey, applicationName)
			if err != nil {
				return sdk.WrapError(err, "cannot load application usage")
			}
			app.Usage = &usage
		}

		if app.FromRepository != "" {
			proj, err := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.WithIntegrations)
			if err != nil {
				return err
			}

			wkAscodeHolder, err := workflow.LoadByRepo(ctx, api.mustDB(), *proj, app.FromRepository, workflow.LoadOptions{
				WithTemplate: true,
			})
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.NewErrorFrom(err, "cannot found workflow holder of the application")
			}
			app.WorkflowAscodeHolder = wkAscodeHolder

			// FIXME from_repository should never be set if the workflow holder was deleted
			if app.WorkflowAscodeHolder == nil {
				app.FromRepository = ""
			}
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

	return usage, nil
}

func (api *API) getApplicationVCSInfosHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]
		remote := r.FormValue("remote")

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		app, err := application.LoadByName(ctx, tx, projectKey, applicationName, application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "cannot load application %s for project %s from db", applicationName, projectKey)
		}

		resp := struct {
			Branches []sdk.VCSBranch `json:"branches,omitempty"`
			Remotes  []sdk.VCSRepo   `json:"remotes,omitempty"`
			Tags     []sdk.VCSTag    `json:"tags,omitempty"`
		}{}

		if app.RepositoryFullname == "" || app.VCSServer == "" {
			return service.WriteJSON(w, resp, http.StatusOK)
		}

		client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, projectKey, app.VCSServer)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth, "cannot get vcs server %s for project %s", app.VCSServer, projectKey))
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		repositoryFullname := app.RepositoryFullname
		if remote != "" && remote != app.RepositoryFullname {
			repositoryFullname = remote
		}
		branches, err := client.Branches(ctx, repositoryFullname, sdk.VCSBranchesFilter{Limit: 50})
		if err != nil {
			return err
		}
		resp.Branches = branches

		tags, err := client.Tags(ctx, repositoryFullname)
		if err != nil {
			return sdk.WrapError(err, "cannot get tags from repository %s", repositoryFullname)
		}
		resp.Tags = tags

		remotes, err := client.ListForks(ctx, repositoryFullname)
		if err != nil {
			return sdk.WrapError(err, "cannot get remotes from repository %s", repositoryFullname)
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

		proj, errl := project.Load(ctx, api.mustDB(), key)
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
			return sdk.WrapError(err, "cannot start transaction")
		}

		defer tx.Rollback() // nolint

		if err := application.Insert(tx, *proj, &app); err != nil {
			return err
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

		app, err := application.LoadByName(ctx, api.mustDB(), projectKey, applicationName)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot begin transaction")
		}
		defer tx.Rollback() // nolint

		err = application.DeleteApplication(tx, app.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot delete application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishDeleteApplication(ctx, projectKey, *app, getAPIConsumer(ctx))

		return nil
	}
}

func (api *API) cloneApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]

		proj, err := project.Load(ctx, api.mustDB(), projectKey)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNoProject, "cannot load %s", projectKey))
		}

		envs, err := environment.LoadEnvironments(api.mustDB(), projectKey)
		if err != nil {
			return sdk.WrapError(err, "cannot load environments for project %s", projectKey)
		}
		proj.Environments = envs

		var newApp sdk.Application
		if err := service.UnmarshalBody(r, &newApp); err != nil {
			return err
		}

		appToClone, err := application.LoadByName(ctx, api.mustDB(), projectKey, applicationName, application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "cannot load application %s", applicationName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := cloneApplication(ctx, tx, api.Cache, *proj, &newApp, appToClone); err != nil {
			return sdk.WrapError(err, "cannot insert new application %s", newApp.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, newApp, http.StatusOK)
	}
}

// cloneApplication Clone an application with all her dependencies: pipelines, permissions, triggers
func cloneApplication(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, newApp *sdk.Application, appToClone *sdk.Application) error {
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

func (api *API) updateAsCodeApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		name := vars["applicationName"]

		branch := FormString(r, "branch")
		message := FormString(r, "message")

		if branch == "" || message == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing branch or message data")
		}

		var a sdk.Application
		if err := service.UnmarshalBody(r, &a); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(a.Name) {
			return sdk.WrapError(sdk.ErrInvalidApplicationPattern, "Application name %s do not respect pattern", a.Name)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(ctx, tx, key, project.LoadOptions.WithClearKeys)
		if err != nil {
			return err
		}

		appDB, err := application.LoadByName(ctx, tx, key, name)
		if err != nil {
			return sdk.WrapError(err, "cannot load application %s", name)
		}

		if appDB.FromRepository == "" {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "current application is not ascode")
		}
		if appDB.FromRepository != a.FromRepository {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "you can't use this repository to update your application: %s", a.FromRepository)
		}

		client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, key, appDB.VCSServer)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "updateAsCodeApplicationHandler> Cannot get client got %s %s : %v", key, appDB.VCSServer, err)
		}

		b, err := client.Branch(ctx, appDB.RepositoryFullname, sdk.VCSBranchFilters{BranchName: branch})
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}

		if b != nil && b.Default {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot push the the default branch on your git repository")
		}

		wkHolder, err := workflow.LoadByRepo(ctx, tx, *proj, appDB.FromRepository, workflow.LoadOptions{
			WithTemplate: true,
		})
		if err != nil {
			return err
		}
		if wkHolder.TemplateInstance != nil {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot edit an application that was generated by a template")
		}

		var rootApp *sdk.Application
		if wkHolder.WorkflowData.Node.Context != nil && wkHolder.WorkflowData.Node.Context.ApplicationID != 0 {
			rootApp, err = application.LoadByIDWithClearVCSStrategyPassword(ctx, tx, wkHolder.WorkflowData.Node.Context.ApplicationID)
			if err != nil {
				return err
			}
		}
		if rootApp == nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot find the root application of the workflow %s that hold the pipeline", wkHolder.Name)
		}

		// create keys
		for i := range a.Keys {
			k := &a.Keys[i]
			newKey, err := keys.GenerateKey(k.Name, k.Type)
			if err != nil {
				return err
			}
			k.Public = newKey.Public
			k.Private = newKey.Private
			k.KeyID = newKey.KeyID
		}

		if a.RepositoryStrategy.ConnectionType == "https" && a.RepositoryStrategy.Password == sdk.PasswordPlaceholder {
			a.RepositoryStrategy.Password = rootApp.RepositoryStrategy.Password
		}

		u := getAPIConsumer(ctx)
		a.ProjectID = proj.ID
		app, err := application.ExportApplication(ctx, tx, a, project.EncryptWithBuiltinKey, fmt.Sprintf("app:%d:%s", appDB.ID, branch))
		if err != nil {
			return sdk.WrapError(err, "unable to export app %s", a.Name)
		}
		wp := exportentities.WorkflowComponents{
			Applications: []exportentities.Application{app},
		}

		ope, err := operation.PushOperationUpdate(ctx, tx, api.Cache, *proj, wp, rootApp.VCSServer, rootApp.RepositoryFullname, branch, message, a.RepositoryStrategy, u)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		api.GoRoutines.Exec(context.Background(), fmt.Sprintf("UpdateAsCodeApplicationHandler-%s", ope.UUID), func(ctx context.Context) {
			ed := ascode.EntityData{
				FromRepo:      appDB.FromRepository,
				Type:          ascode.ApplicationEvent,
				ID:            appDB.ID,
				Name:          appDB.Name,
				OperationUUID: ope.UUID,
			}
			ascode.UpdateAsCodeResult(ctx, api.mustDB(), api.Cache, api.GoRoutines, *proj, *wkHolder, *rootApp, ed, u)
		})

		return service.WriteJSON(w, sdk.Operation{
			UUID:   ope.UUID,
			Status: ope.Status,
		}, http.StatusOK)
	}
}

func (api *API) updateApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]

		app, err := application.LoadByNameWithClearVCSStrategyPassword(ctx, api.mustDB(), projectKey, applicationName, application.LoadOptions.Default)
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
			return sdk.WrapError(sdk.ErrInvalidApplicationPattern, "application name %s do not respect pattern %s", appPost.Name, sdk.NamePattern)
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
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := application.Update(ctx, tx, app); err != nil {
			return sdk.WrapError(err, "cannot delete application %s", applicationName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishUpdateApplication(ctx, projectKey, *app, old, getAPIConsumer(ctx))

		return service.WriteJSON(w, app, http.StatusOK)

	}
}

func (api *API) postApplicationMetadataHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		applicationName := vars["applicationName"]

		app, err := application.LoadByName(ctx, api.mustDB(), projectKey, applicationName)
		if err != nil {
			return err
		}
		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		oldApp := *app

		m := vars["metadata"]
		v, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.WithStack(err)
		}
		defer r.Body.Close()

		app.Metadata[m] = string(v)
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start tx")
		}
		defer tx.Rollback() // nolint

		if err := application.Update(ctx, tx, app); err != nil {
			return sdk.WrapError(err, "unable to update application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit tx")
		}

		event.PublishUpdateApplication(ctx, projectKey, *app, oldApp, getAPIConsumer(ctx))

		return nil
	}
}
