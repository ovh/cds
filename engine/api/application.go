package api

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowv0"
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

		var u = getUser(ctx)
		requestedUserName := r.Header.Get("X-Cds-Username")

		//A provider can make a call for a specific user
		if getProvider(ctx) != nil && requestedUserName != "" {
			var err error
			//Load the specific user
			u, err = user.LoadUserWithoutAuth(api.mustDB(), requestedUserName)
			if err != nil {
				if err == sql.ErrNoRows {
					return sdk.ErrUserNotFound
				}
				return sdk.WrapError(err, "getApplicationsHandler> unable to load user '%s'", requestedUserName)
			}
			if err := loadUserPermissions(api.mustDB(), api.Cache, u); err != nil {
				return sdk.WrapError(err, "getApplicationsHandler> unable to load user '%s' permissions", requestedUserName)
			}
		}
		loadOpts := []application.LoadOptionFunc{}
		if withIcon {
			loadOpts = append(loadOpts, application.LoadOptions.WithIcon)
		}
		applications, err := application.LoadAll(api.mustDB(), api.Cache, projectKey, u, loadOpts...)
		if err != nil {
			return sdk.WrapError(err, "getApplicationsHandler> Cannot load applications from db")
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

func (api *API) getApplicationTreeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		tree, err := workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), "", "", 0)
		if err != nil {
			return sdk.WrapError(err, "getApplicationTreeHandler> Cannot load CD Tree for applications %s", applicationName)
		}

		return service.WriteJSON(w, tree, http.StatusOK)
	}
}

func (api *API) getPipelineBuildBranchHistoryHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]

		err := r.ParseForm()
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "getPipelineBranchHistoryHandler> Cannot parse form: %s", err)
		}

		pageString := r.Form.Get("page")
		nbPerPageString := r.Form.Get("perPage")

		var nbPerPage int
		if nbPerPageString != "" {
			nbPerPage, err = strconv.Atoi(nbPerPageString)
			if err != nil {
				return err
			}
		} else {
			nbPerPage = 20
		}

		var page int
		if pageString != "" {
			page, err = strconv.Atoi(pageString)
			if err != nil {
				return err
			}
		} else {
			nbPerPage = 0
		}

		pbs, err := pipeline.GetBranchHistory(api.mustDB(), projectKey, appName, page, nbPerPage)
		if err != nil {
			errL := fmt.Errorf("Cannot load pipeline branch history: %s", err)
			return sdk.WrapError(errL, "getPipelineBranchHistoryHandler> Cannot get history by branch")
		}

		return service.WriteJSON(w, pbs, http.StatusOK)
	}
}

func (api *API) getApplicationDeployHistoryHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]

		pbs, err := pipeline.GetDeploymentHistory(api.mustDB(), projectKey, appName)
		if err != nil {
			errL := fmt.Errorf("Cannot load pipeline deployment history: %s", err)
			return sdk.WrapError(errL, "getPipelineDeployHistoryHandler> Cannot get history by env")
		}

		return service.WriteJSON(w, pbs, http.StatusOK)
	}
}

func (api *API) getApplicationBranchVersionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]
		branch := r.FormValue("branch")
		remote := r.FormValue("remote")

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), application.LoadOptions.WithTriggers)
		if err != nil {
			return sdk.WrapError(err, "getApplicationBranchVersionHandler: Cannot load application %s for project %s from db", applicationName, projectKey)
		}

		versions, err := pipeline.GetVersions(api.mustDB(), app, branch, remote)
		if err != nil {
			return sdk.WrapError(err, "getApplicationBranchVersionHandler: Cannot load version for application %s on branch %s with remote %s", applicationName, branch, remote)
		}

		return service.WriteJSON(w, versions, http.StatusOK)
	}
}

func (api *API) getApplicationTreeStatusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]
		branchName := r.FormValue("branchName")
		remote := r.FormValue("remote")
		versionString := r.FormValue("version")

		var version int64
		var errV error
		if versionString != "" {
			version, errV = strconv.ParseInt(versionString, 10, 64)
			if errV != nil {
				return sdk.WrapError(errV, "getApplicationTreeStatusHandler>Cannot cast version %s into int", versionString)
			}
		}

		app, errApp := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx))
		if errApp != nil {
			return sdk.WrapError(errApp, "getApplicationTreeStatusHandler>Cannot get application")
		}

		pbs, schedulers, hooks, errPB := workflowv0.GetWorkflowStatus(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), branchName, remote, version)
		if errPB != nil {
			return sdk.WrapError(errPB, "getApplicationHandler> Cannot load CD Tree status %s", app.Name)
		}

		response := struct {
			Builds     []sdk.PipelineBuild     `json:"builds"`
			Schedulers []sdk.PipelineScheduler `json:"schedulers"`
			Pollers    []struct{}              `json:"pollers"`
			Hooks      []sdk.Hook              `json:"hooks"`
		}{
			pbs,
			schedulers,
			nil,
			hooks,
		}

		return service.WriteJSON(w, response, http.StatusOK)
	}
}

func (api *API) getApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		applicationStatus := FormBool(r, "applicationStatus")
		withHooks := FormBool(r, "withHooks")
		withNotifs := FormBool(r, "withNotifs")
		withWorkflow := FormBool(r, "withWorkflow")
		withTriggers := FormBool(r, "withTriggers")
		withSchedulers := FormBool(r, "withSchedulers")
		withKeys := FormBool(r, "withKeys")
		withUsage := FormBool(r, "withUsage")
		withIcon := FormBool(r, "withIcon")
		withDeploymentStrategies := FormBool(r, "withDeploymentStrategies")
		withVulnerabilities := FormBool(r, "withVulnerabilities")
		branchName := r.FormValue("branchName")
		remote := r.FormValue("remote")
		versionString := r.FormValue("version")

		loadOptions := []application.LoadOptionFunc{
			application.LoadOptions.WithVariables,
			application.LoadOptions.WithPipelines,
		}
		if withHooks {
			loadOptions = append(loadOptions, application.LoadOptions.WithHooks)
		}
		if withTriggers {
			loadOptions = append(loadOptions, application.LoadOptions.WithTriggers)
		}
		if withNotifs {
			loadOptions = append(loadOptions, application.LoadOptions.WithNotifs)
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

		app, errApp := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), loadOptions...)
		if errApp != nil {
			return sdk.WrapError(errApp, "getApplicationHandler: Cannot load application %s for project %s from db", applicationName, projectKey)
		}

		if err := application.LoadGroupByApplication(api.mustDB(), app); err != nil {
			return sdk.WrapError(err, "getApplicationHandler> Unable to load groups by application")
		}

		if withSchedulers {
			var errScheduler error
			app.Schedulers, errScheduler = scheduler.GetByApplication(api.mustDB(), app)
			if errScheduler != nil {
				return sdk.WrapError(errScheduler, "getApplicationHandler> Cannot load schedulers for application %s", applicationName)
			}
		}

		if withWorkflow {
			brName := branchName
			if brName == "" {
				brName = "master"
			}
			var errWorflow error
			app.Workflows, errWorflow = workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), brName, remote, 0)
			if errWorflow != nil {
				return sdk.WrapError(errWorflow, "getApplicationHandler> Cannot load CD Tree for applications %s", app.Name)
			}
		}

		if applicationStatus {
			version := 0
			if versionString != "" {
				var errStatus error
				version, errStatus = strconv.Atoi(versionString)
				if errStatus != nil {
					return sdk.WrapError(errStatus, "getApplicationHandler> Version %s is not an integer", versionString)
				}
			}

			if version != 0 && branchName == "" {
				return sdk.WrapError(sdk.ErrBranchNameNotProvided, "getApplicationHandler: branchName must be provided with version param")
			}

			pipelineBuilds, errPipBuilds := pipeline.GetAllLastBuildByApplication(api.mustDB(), app.ID, remote, branchName, version)
			if errPipBuilds != nil {
				return sdk.WrapError(errPipBuilds, "getApplicationHandler> Cannot load app status by version")
			}
			al := r.Header.Get("Accept-Language")
			for _, p := range pipelineBuilds {
				p.Translate(al)
			}
			app.PipelinesBuild = pipelineBuilds
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

	envs, errEnv := environment.LoadByApplicationName(db, projKey, appName)
	if errEnv != nil {
		return usage, sdk.WrapError(errEnv, "loadApplicationUsage> Cannot load environments linked to application %s in project %s", appName, projKey)
	}
	usage.Environments = envs

	pips, errPips := pipeline.LoadByApplicationName(db, projKey, appName)
	if errPips != nil {
		return usage, sdk.WrapError(errPips, "loadApplicationUsage> Cannot load pipelines linked to application %s in project %s", appName, projKey)
	}
	usage.Pipelines = pips

	return usage, nil
}

func (api *API) getApplicationBranchHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]
		remote := r.FormValue("remote")

		proj, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getApplicationBranchHandler> Cannot load project %s from db", projectKey)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "getApplicationBranchHandler> Cannot load application %s for project %s from db", applicationName, projectKey)
		}

		var branches []sdk.VCSBranch
		if app.RepositoryFullname != "" && app.VCSServer != "" {
			vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
			client, erra := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, vcsServer)
			if erra != nil {
				return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getApplicationBranchHandler> Cannot get client got %s %s : %s", projectKey, app.VCSServer, erra)
			}
			if remote != "" && remote != app.RepositoryFullname {
				brs, errB := client.Branches(ctx, remote)
				if errB != nil {
					return sdk.WrapError(errB, "getApplicationBranchHandler> Cannot get branches from repository %s", remote)
				}
				for _, br := range brs {
					branches = append(branches, br)
				}
			} else {
				var errb error
				branches, errb = client.Branches(ctx, app.RepositoryFullname)
				if errb != nil {
					return sdk.WrapError(errb, "getApplicationBranchHandler> Cannot get branches from repository %s: %s", app.RepositoryFullname, errb)
				}
			}
		} else {
			var errg error
			branches, errg = pipeline.GetBranches(api.mustDB(), app, remote)
			if errg != nil {
				return sdk.WrapError(errg, "getApplicationBranchHandler> Cannot get branches from builds")
			}
		}

		//Yo analyze branch and delete pipeline_build for old branches...

		return service.WriteJSON(w, branches, http.StatusOK)
	}
}

func (api *API) getApplicationVCSInfosHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]
		remote := r.FormValue("remote")

		proj, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getApplicationVCSInfosHandler> Cannot load project %s from db", projectKey)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "getApplicationVCSInfosHandler> Cannot load application %s for project %s from db", applicationName, projectKey)
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

func (api *API) getApplicationRemoteHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		proj, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getApplicationRemoteHandler> Cannot load project %s", projectKey)
		}

		app, errL := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), application.LoadOptions.Default)
		if errL != nil {
			return sdk.WrapError(errL, "getApplicationRemoteHandler: Cannot load application %s for project %s", applicationName, projectKey)
		}

		remotes := []sdk.VCSRemote{}
		var prs []sdk.VCSPullRequest
		if app.RepositoryFullname != "" && app.VCSServer != "" {
			vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
			client, erra := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, vcsServer)
			if erra != nil {
				return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getApplicationRemoteHandler> Cannot get client got %s %s : %s", projectKey, app.VCSServer, erra)
			}
			var errb error
			prs, errb = client.PullRequests(ctx, app.RepositoryFullname)
			if errb != nil {
				return sdk.WrapError(errb, "getApplicationRemoteHandler> Cannot get branches from repository %s: %s", app.RepositoryFullname, errb)
			}

			found := map[string]bool{app.RepositoryFullname: true}
			remotes = append(remotes, sdk.VCSRemote{Name: app.RepositoryFullname})
			for _, pr := range prs {
				if _, exist := found[pr.Head.Repo]; !exist {
					remotes = append(remotes, sdk.VCSRemote{URL: pr.Head.CloneURL, Name: pr.Head.Repo})
				}
				found[pr.Head.Repo] = true
			}
		}

		oldRemotes, errg := pipeline.GetRemotes(api.mustDB(), app)
		if errg != nil {
			return sdk.WrapError(errg, "getApplicationRemoteHandler> Cannot get remotes from builds")
		}
		for _, oldRemote := range oldRemotes {
			exist := false
			for _, remote := range remotes {
				if remote.Name == oldRemote.Name {
					exist = true
				}
			}
			if !exist {
				remotes = append(remotes, oldRemote)
			}
		}

		return service.WriteJSON(w, remotes, http.StatusOK)
	}
}

func (api *API) addApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		proj, errl := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errl != nil {
			return sdk.WrapError(errl, "addApplicationHandler> Cannot load %s", key)
		}

		var app sdk.Application
		if err := UnmarshalBody(r, &app); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(app.Name) {
			return sdk.WrapError(sdk.ErrInvalidApplicationPattern, "addApplicationHandler: Application name %s do not respect pattern %s", app.Name, sdk.NamePattern)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addApplicationHandler> Cannot start transaction")
		}

		defer tx.Rollback()

		if err := application.Insert(tx, api.Cache, proj, &app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addApplicationHandler> Cannot insert pipeline")
		}

		if err := group.LoadGroupByProject(tx, proj); err != nil {
			return sdk.WrapError(err, "addApplicationHandler> Cannot load group from project")
		}

		if err := application.AddGroup(tx, api.Cache, proj, &app, getUser(ctx), proj.ProjectGroups...); err != nil {
			return sdk.WrapError(err, "addApplicationHandler> Cannot add groups on application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addApplicationHandler> Cannot commit transaction")
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

		proj, errP := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "deleteApplicationHandler> Cannot laod project")
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx))
		if err != nil {
			if err != sdk.ErrApplicationNotFound {
				log.Warning("deleteApplicationHandler> Cannot load application %s: %s\n", applicationName, err)
			}
			return err
		}

		nb, errNb := pipeline.CountBuildingPipelineByApplication(api.mustDB(), app.ID)
		if errNb != nil {
			return sdk.WrapError(errNb, "deleteApplicationHandler> Cannot count pipeline build for application %d", app.ID)
		}

		if nb > 0 {
			return sdk.WrapError(sdk.ErrAppBuildingPipelines, "deleteApplicationHandler> Cannot delete application [%d], there are building pipelines: %d", app.ID, nb)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteApplicationHandler> Cannot begin transaction")
		}
		defer tx.Rollback()

		err = application.DeleteApplication(tx, app.ID)
		if err != nil {
			return sdk.WrapError(err, "deleteApplicationHandler> Cannot delete application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteApplicationHandler> Cannot commit transaction")
		}

		event.PublishDeleteApplication(proj.Key, *app, getUser(ctx))

		return nil
	}
}

func (api *API) cloneApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		proj, errProj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errProj != nil {
			return sdk.WrapError(sdk.ErrNoProject, "cloneApplicationHandler> Cannot load %s", projectKey)
		}

		envs, errE := environment.LoadEnvironments(api.mustDB(), projectKey, true, getUser(ctx))
		if errE != nil {
			return sdk.WrapError(errE, "cloneApplicationHandler> Cannot load Environments %s", projectKey)

		}
		proj.Environments = envs

		var newApp sdk.Application
		if err := UnmarshalBody(r, &newApp); err != nil {
			return err
		}

		appToClone, errApp := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), application.LoadOptions.Default, application.LoadOptions.WithGroups)
		if errApp != nil {
			return sdk.WrapError(errApp, "cloneApplicationHandler> Cannot load application %s", applicationName)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "cloneApplicationHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := cloneApplication(tx, api.Cache, proj, &newApp, appToClone, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "cloneApplicationHandler> Cannot insert new application %s", newApp.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cloneApplicationHandler> Cannot commit transaction")
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

		p, errload := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.Default)
		if errload != nil {
			return sdk.WrapError(errload, "updateApplicationHandler> Cannot load project %s", projectKey)
		}

		app, errloadbyname := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), application.LoadOptions.Default)
		if errloadbyname != nil {
			return sdk.WrapError(errloadbyname, "updateApplicationHandler> Cannot load application %s", applicationName)
		}

		var appPost sdk.Application
		if err := UnmarshalBody(r, &appPost); err != nil {
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
			return sdk.WrapError(err, "updateApplicationHandler> Cannot start transaction")
		}
		defer tx.Rollback()
		if err := application.Update(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateApplicationHandler> Cannot delete application %s", applicationName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateApplicationHandler> Cannot commit transaction")
		}

		event.PublishUpdateApplication(p.Key, *app, old, getUser(ctx))

		return service.WriteJSON(w, app, http.StatusOK)

	}
}

func (api *API) postApplicationMetadataHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx))
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
			return sdk.WrapError(err, "postApplicationMetadataHandler> unable to start tx")
		}
		defer tx.Rollback() // nolint

		if err := application.Update(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "postApplicationMetadataHandler> unable to update application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postApplicationMetadataHandler> unable to commit tx")
		}

		event.PublishUpdateApplication(projectKey, *app, oldApp, getUser(ctx))

		return nil
	}
}
