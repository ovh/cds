package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkflowHooksHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		hooks, err := workflow.LoadAllHooks(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getWorkflowHooksHandler")
		}

		return WriteJSON(w, r, hooks, http.StatusOK)
	}
}

func (api *API) getWorkflowHookModelsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]
		nodeID, errN := requestVarInt(r, "nodeID")
		if errN != nil {
			return sdk.WrapError(errN, "getWorkflowHookModelsHandler")
		}

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "getWorkflowHookModelsHandler > project.Load")
		}

		wf, errW := workflow.Load(api.mustDB(), api.Cache, key, workflowName, getUser(ctx))
		if errW != nil {
			return sdk.WrapError(errW, "getWorkflowHookModelsHandler > workflow.Load")
		}

		node := wf.GetNode(nodeID)
		if node == nil {
			return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowHookModelsHandler")
		}

		m, err := workflow.LoadHookModels(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getWorkflowHookModelsHandler")
		}

		// Post processing  on repositoryWebHook
		hasRepoManager := false
		repoWebHookEnable := false
		if node.Context.Application != nil && node.Context.Application.RepositoryFullname != "" {
			hasRepoManager = true
		}
		var webHookInfo repositoriesmanager.WebhooksInfos
		if hasRepoManager {
			// Call VCS to know if repository allows webhook and get the configuration fields
			vcsServer := repositoriesmanager.GetProjectVCSServer(p, node.Context.Application.VCSServer)
			if vcsServer != nil {
				client, errclient := repositoriesmanager.AuthorizedClient(api.mustDB(), api.Cache, vcsServer)
				if errclient != nil {
					return sdk.WrapError(errclient, "getWorkflowHookModelsHandler> Cannot get vcs client")
				}
				var errWH error
				webHookInfo, errWH = repositoriesmanager.GetWebhooksInfos(client)
				if errWH != nil {
					return sdk.WrapError(errWH, "getWorkflowHookModelsHandler> Cannot get vcs web hook info")
				}
				repoWebHookEnable = webHookInfo.WebhooksSupported && !webHookInfo.WebhooksDisabled
			}
		}

		indexToDelete := -1
		for i := range m {
			if m[i].Name == workflow.RepositoryWebHookModel.Name {
				if !repoWebHookEnable {
					indexToDelete = i
					break
				} else {
					m[i].Icon = webHookInfo.Icon
				}
			}

		}
		if indexToDelete > -1 {
			m = append(m[0:indexToDelete], m[indexToDelete+1:]...)
		}

		return WriteJSON(w, r, m, http.StatusOK)
	}
}

func (api *API) getWorkflowHookModelHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["model"]
		m, err := workflow.LoadHookModelByName(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowHookModelHandler")
		}
		return WriteJSON(w, r, m, http.StatusOK)
	}
}

func (api *API) postWorkflowHookModelHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		m := &sdk.WorkflowHookModel{}
		if err := UnmarshalBody(r, m); err != nil {
			return sdk.WrapError(err, "postWorkflowHookModelHandler")
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "postWorkflowHookModelHandler> Unable to start transaction")
		}
		defer tx.Rollback()

		if err := workflow.InsertHookModel(tx, m); err != nil {
			return sdk.WrapError(err, "postWorkflowHookModelHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowHookModelHandler> Unable to commit transaction")
		}

		return WriteJSON(w, r, m, http.StatusCreated)
	}
}

func (api *API) putWorkflowHookModelHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		m := &sdk.WorkflowHookModel{}
		if err := UnmarshalBody(r, m); err != nil {
			return err
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "putWorkflowHookModelHandler> Unable to start transaction")
		}

		defer tx.Rollback()

		if err := workflow.UpdateHookModel(tx, m); err != nil {
			return sdk.WrapError(err, "putWorkflowHookModelHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errtx, "putWorkflowHookModelHandler> Unable to commit transaction")
		}

		return WriteJSON(w, r, m, http.StatusOK)
	}
}
