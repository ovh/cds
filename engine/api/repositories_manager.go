package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getRepositoriesManagerForProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		proj, errproj := project.Load(ctx, api.mustDB(), key)
		if errproj != nil {
			return errproj
		}

		return service.WriteJSON(w, proj.VCSServers, http.StatusOK)
	}
}

func (api *API) getReposFromRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		vcsServerName := vars["name"]
		sync := service.FormBool(r, "synchronize")

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(ctx, tx, projectKey)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth,
				"cannot get client got %s %s", projectKey, vcsServerName))
		}

		repos, err := repositoriesmanager.GetReposForProjectVCSServer(ctx, tx, api.Cache, *proj, vcsServerName, repositoriesmanager.Options{
			Sync: sync,
		})
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, repos, http.StatusOK)
	}
}

func (api *API) getRepoFromRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		rmName := vars["name"]
		repoName := r.FormValue("repo")

		if repoName == "" {
			return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Missing repository name 'repo' as a query parameter"))
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, projectKey, rmName)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth,
				"cannot get client got %s %s", projectKey, rmName))
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		log.Info(ctx, "getRepoFromRepositoriesManagerHandler> Loading repository on %s", rmName)

		repo, err := client.RepoByFullname(ctx, repoName)
		if err != nil {
			return sdk.WrapError(err, "cannot get repos")
		}

		return service.WriteJSON(w, repo, http.StatusOK)
	}
}

func (api *API) attachRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		appName := vars["applicationName"]
		rmName := vars["name"]
		fullname := r.FormValue("fullname")

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		app, err := application.LoadByName(ctx, tx, projectKey, appName)
		if err != nil {
			return sdk.WrapError(err, "cannot load application %s", appName)
		}

		//Get an authorized Client
		client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, projectKey, rmName)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "cannot get client got %s %s : %s", projectKey, rmName, err)
		}

		if _, err := client.RepoByFullname(ctx, fullname); err != nil {
			return sdk.WrapError(sdk.ErrRepoNotFound, "cannot get repo %s: %s", fullname, err)
		}

		app.VCSServer = rmName
		app.RepositoryFullname = fullname

		if err := repositoriesmanager.InsertForApplication(tx, app); err != nil {
			return sdk.WrapError(err, "cannot insert for application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		db := api.mustDB()

		usage, errU := loadApplicationUsage(ctx, db, projectKey, appName)
		if errU != nil {
			return sdk.WrapError(errU, "attachRepositoriesManager> Cannot load application usage")
		}

		// Update default payload of linked workflow root
		if len(usage.Workflows) > 0 {
			proj, errP := project.Load(ctx, db, projectKey, project.LoadOptions.WithIntegrations)
			if errP != nil {
				return sdk.WrapError(errP, "attachRepositoriesManager> Cannot load project")
			}

			for _, wf := range usage.Workflows {
				wfDB, err := workflow.LoadByID(ctx, db, api.Cache, *proj, wf.ID, workflow.LoadOptions{})
				if err != nil {
					return err
				}

				// second load for publish the event below
				wfOld, err := workflow.LoadByID(ctx, db, api.Cache, *proj, wf.ID, workflow.LoadOptions{})
				if err != nil {
					return err
				}

				if wfDB.WorkflowData.Node.Context == nil {
					wfDB.WorkflowData.Node.Context = &sdk.NodeContext{}
				}
				if wfDB.WorkflowData.Node.Context.ApplicationID != app.ID {
					continue
				}

				payload, err := wfDB.WorkflowData.Node.Context.DefaultPayloadToMap()
				if err != nil {
					return sdk.WithStack(err)
				}

				if _, ok := payload["git.branch"]; ok && payload["git.repository"] == app.RepositoryFullname {
					continue
				}

				tx, err := api.mustDB().Begin()
				if err != nil {
					return sdk.WithStack(err)
				}
				defer tx.Rollback() // nolint

				defaultPayload, err := workflow.DefaultPayload(ctx, tx, api.Cache, *proj, wfDB)
				if err != nil {
					return sdk.WithStack(err)
				}

				wfDB.WorkflowData.Node.Context.DefaultPayload = defaultPayload

				if err := workflow.Update(ctx, tx, api.Cache, *proj, wfDB, workflow.UpdateOptions{DisableHookManagement: true}); err != nil {
					return sdk.WrapError(err, "cannot update node context %d", wfDB.WorkflowData.Node.Context.ID)
				}

				if err := tx.Commit(); err != nil {
					return sdk.WithStack(err)
				}

				event.PublishWorkflowUpdate(ctx, proj.Key, *wfDB, *wfOld, getUserConsumer(ctx))
			}
		}

		event.PublishApplicationRepositoryAdd(ctx, projectKey, *app, getUserConsumer(ctx))

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) detachRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		appName := vars["applicationName"]
		db := api.mustDB()
		u := getUserConsumer(ctx)

		app, err := application.LoadByName(ctx, db, projectKey, appName)
		if err != nil {
			return err
		}

		// Check if there is hooks on this application
		repositoryWebHooksCount, err := workflow.CountRepositoryWebHooksByApplication(db, app.ID)
		if err != nil {
			return err
		}
		if repositoryWebHooksCount > 0 {
			return sdk.WithStack(sdk.ErrRepositoryUsedByHook)
		}

		// Remove all the things in a transaction
		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := repositoriesmanager.DeleteForApplication(tx, app); err != nil {
			return sdk.WrapError(err, "cannot delete for application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishApplicationRepositoryDelete(ctx, projectKey, appName, app.VCSServer, app.RepositoryFullname, u)

		return service.WriteJSON(w, app, http.StatusOK)
	}
}
