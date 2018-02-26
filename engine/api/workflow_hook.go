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

		return WriteJSON(w, hooks, http.StatusOK)
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

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithPlatforms)
		if errP != nil {
			return sdk.WrapError(errP, "getWorkflowHookModelsHandler > project.Load")
		}

		wf, errW := workflow.Load(api.mustDB(), api.Cache, key, workflowName, getUser(ctx), workflow.LoadOptions{})
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
		repoWebHookEnable, repoPollerEnable := false, false
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

				pollInfo, errPoll := repositoriesmanager.GetPollingInfos(client)
				if errPoll != nil {
					return sdk.WrapError(errPoll, "getWorkflowHookModelsHandler> Cannot get vcs poller info")
				}
				repoPollerEnable = pollInfo.PollingSupported && !pollInfo.PollingDisabled
			}
		}

		hasKafka := false
		for _, platform := range p.Platforms {
			if platform.Model.Name == sdk.KafkaPlatformModel {
				hasKafka = true
				break
			}
		}

		models := []sdk.WorkflowHookModel{}
		for i := range m {
			switch m[i].Name {
			case sdk.RepositoryWebHookModelName:
				if repoWebHookEnable {
					m[i].Icon = webHookInfo.Icon
					models = append(models, m[i])
				}
			case sdk.GitPollerModelName:
				if repoPollerEnable {
					models = append(models, m[i])
				}
			case sdk.KafkaHookModelName:
				if hasKafka {
					models = append(models, m[i])
				}
			default:
				models = append(models, m[i])
			}
		}

		return WriteJSON(w, models, http.StatusOK)
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
		return WriteJSON(w, m, http.StatusOK)
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

		return WriteJSON(w, m, http.StatusCreated)
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

		return WriteJSON(w, m, http.StatusOK)
	}
}
