package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (api *API) getWorkflowHooksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// This handler can only be called by a service managed by an admin
		if isService := isService(ctx); !isService && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		hooks, err := workflow.LoadAllHooks(api.mustDB())
		if err != nil {
			return err
		}

		return service.WriteJSON(w, hooks, http.StatusOK)
	}
}

func (api *API) getWorkflowOutgoingHookModelsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		m, err := workflow.LoadOutgoingHookModels(api.mustDB())
		if err != nil {
			return sdk.WithStack(err)
		}
		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) getWorkflowHookModelsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]

		nodeID, err := requestVarInt(r, "nodeID")
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		p, err := project.Load(ctx, tx, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WithStack(err)
		}

		wf, err := workflow.Load(ctx, tx, api.Cache, *p, workflowName, workflow.LoadOptions{})
		if err != nil {
			return sdk.WithStack(err)
		}

		node := wf.WorkflowData.NodeByID(nodeID)
		if node == nil {
			return sdk.WithStack(sdk.ErrWorkflowNodeNotFound)
		}

		m, err := workflow.LoadHookModels(tx)
		if err != nil {
			return sdk.WithStack(err)
		}

		// Post processing  on repositoryWebHook
		hasRepoManager := false
		repoWebHookEnable, repoPollerEnable, gerritHookEnable := false, false, false
		if node.IsLinkedToRepo(wf) {
			hasRepoManager = true
		}
		var webHookInfo repositoriesmanager.WebhooksInfos
		if hasRepoManager {
			// Call VCS to know if repository allows webhook and get the configuration fields
			vcsServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, api.mustDB(), p.Key, wf.GetApplication(node.Context.ApplicationID).VCSServer)
			if err == nil {
				client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, p.Key, vcsServer)
				if err != nil {
					return sdk.WrapError(err, "cannot get vcs client")
				}

				webHookInfo, err = repositoriesmanager.GetWebhooksInfos(ctx, client)
				if err != nil {
					return sdk.WrapError(err, "cannot get vcs web hook info")
				}
				repoWebHookEnable = webHookInfo.WebhooksSupported && !webHookInfo.WebhooksDisabled

				pollInfo, err := repositoriesmanager.GetPollingInfos(ctx, client, *p)
				if err != nil {
					return sdk.WrapError(err, "cannot get vcs poller info")
				}
				repoPollerEnable = pollInfo.PollingSupported && !pollInfo.PollingDisabled

				gerritHookEnable = !webHookInfo.GerritHookDisabled
			}
		}

		hasKafka := false
		for _, integration := range p.Integrations {
			if integration.Model.Hook {
				hasKafka = true
				break
			}
		}

		models := make([]sdk.WorkflowHookModel, 0, len(m))
		for i := range m {
			switch m[i].Name {
			case sdk.GerritHookModelName:
				if gerritHookEnable {
					m[i].Icon = webHookInfo.Icon
					m[i].DefaultConfig[sdk.HookConfigEventFilter] = sdk.WorkflowNodeHookConfigValue{
						Type:               sdk.HookConfigTypeMultiChoice,
						Value:              webHookInfo.Events[0],
						Configurable:       true,
						MultipleChoiceList: webHookInfo.Events,
					}
					models = append(models, m[i])
				}
			case sdk.RepositoryWebHookModelName:
				if repoWebHookEnable {
					m[i].Icon = webHookInfo.Icon
					m[i].DefaultConfig[sdk.HookConfigEventFilter] = sdk.WorkflowNodeHookConfigValue{
						Type:               sdk.HookConfigTypeMultiChoice,
						Value:              webHookInfo.Events[0],
						Configurable:       true,
						MultipleChoiceList: webHookInfo.Events,
					}
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

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, models, http.StatusOK)
	}
}

func (api *API) getWorkflowHookModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["model"]
		m, err := workflow.LoadHookModelByName(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowHookModelHandler")
		}
		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) postWorkflowHookModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		m := &sdk.WorkflowHookModel{}
		if err := service.UnmarshalBody(r, m); err != nil {
			return sdk.WrapError(err, "postWorkflowHookModelHandler")
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "postWorkflowHookModelHandler> Unable to start transaction")
		}
		defer tx.Rollback() // nolint

		if err := workflow.InsertHookModel(tx, m); err != nil {
			return sdk.WrapError(err, "postWorkflowHookModelHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Unable to commit transaction")
		}

		return service.WriteJSON(w, m, http.StatusCreated)
	}
}

func (api *API) putWorkflowHookModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		m := &sdk.WorkflowHookModel{}
		if err := service.UnmarshalBody(r, m); err != nil {
			return err
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "putWorkflowHookModelHandler> Unable to start transaction")
		}

		defer tx.Rollback() // nolint

		if err := workflow.UpdateHookModel(tx, m); err != nil {
			return sdk.WrapError(err, "putWorkflowHookModelHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errtx, "putWorkflowHookModelHandler> Unable to commit transaction")
		}

		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) postWorkflowJobHookCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]
		hookRunID := vars["hookRunID"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		var callback sdk.WorkflowNodeOutgoingHookRunCallback
		if err := service.UnmarshalBody(r, &callback); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		defer tx.Rollback() // nolint

		_, next := telemetry.Span(ctx, "project.Load")
		proj, err := project.Load(ctx, tx, key,
			project.LoadOptions.WithVariables,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationVariables,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
		)
		next()
		if err != nil {
			return sdk.WrapError(err, "cannot load project")
		}
		wr, err := workflow.LoadRun(ctx, tx, key, workflowName, number, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow run")
		}

		secrets, err := workflow.LoadDecryptSecrets(ctx, tx, wr, nil)
		if err != nil {
			return sdk.WrapError(err, "cannot load secrets")
		}

		// Hide secrets in payload
		for _, s := range secrets {
			callback.Log = strings.Replace(callback.Log, s.Value, "**"+s.Name+"**", -1)
		}

		report, err := workflow.UpdateOutgoingHookRunStatus(ctx, tx, api.Cache, *proj, wr, hookRunID, callback)
		if err != nil {
			return sdk.WrapError(err, "unable to update outgoing hook run status")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		go api.WorkflowSendEvent(context.Background(), *proj, report)

		report, err = api.updateParentWorkflowRun(ctx, wr)
		if err != nil {
			return sdk.WithStack(err)
		}

		go api.WorkflowSendEvent(context.Background(), *proj, report)

		return nil
	}
}

func (api *API) getWorkflowJobHookDetailsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]
		hookRunID := vars["hookRunID"]
		number, errnum := requestVarInt(r, "number")
		if errnum != nil {
			return errnum
		}

		db := api.mustDB()

		wr, err := workflow.LoadRun(ctx, db, key, workflowName, number, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			return err
		}

		hr := wr.GetOutgoingHookRun(hookRunID)
		if hr == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		secrets, errSecret := workflow.LoadDecryptSecrets(ctx, db, wr, nil)
		if errSecret != nil {
			return sdk.WrapError(errSecret, "cannot load secrets")
		}
		hr.BuildParameters = append(hr.BuildParameters, sdk.VariablesToParameters("", secrets)...)
		return service.WriteJSON(w, hr, http.StatusOK)
	}
}
