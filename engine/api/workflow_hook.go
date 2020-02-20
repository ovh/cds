package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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

		nodeID, errN := requestVarInt(r, "nodeID")
		if errN != nil {
			return sdk.WithStack(errN)
		}

		p, errP := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithIntegrations)
		if errP != nil {
			return sdk.WithStack(errP)
		}

		wf, errW := workflow.Load(ctx, api.mustDB(), api.Cache, p, workflowName, workflow.LoadOptions{})
		if errW != nil {
			return sdk.WithStack(errW)
		}

		node := wf.WorkflowData.NodeByID(nodeID)
		if node == nil {
			return sdk.WithStack(sdk.ErrWorkflowNodeNotFound)
		}

		m, err := workflow.LoadHookModels(api.mustDB())
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
			vcsServer := repositoriesmanager.GetProjectVCSServer(p, wf.GetApplication(node.Context.ApplicationID).VCSServer)
			if vcsServer != nil {
				client, errclient := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, p.Key, vcsServer)
				if errclient != nil {
					return sdk.WrapError(errclient, "cannot get vcs client")
				}
				var errWH error
				webHookInfo, errWH = repositoriesmanager.GetWebhooksInfos(ctx, client)
				if errWH != nil {
					return sdk.WrapError(errWH, "cannot get vcs web hook info")
				}
				repoWebHookEnable = webHookInfo.WebhooksSupported && !webHookInfo.WebhooksDisabled

				pollInfo, errPoll := repositoriesmanager.GetPollingInfos(ctx, client, *p)
				if errPoll != nil {
					return sdk.WrapError(errPoll, "cannot get vcs poller info")
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
		number, errnum := requestVarInt(r, "number")
		if errnum != nil {
			return errnum
		}

		var callback sdk.WorkflowNodeOutgoingHookRunCallback
		if err := service.UnmarshalBody(r, &callback); err != nil {
			return sdk.WrapError(err, "Unable to unmarshal body")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return err
		}

		defer tx.Rollback() // nolint

		_, next := observability.Span(ctx, "project.Load")
		proj, errP := project.Load(tx, api.Cache, key,
			project.LoadOptions.WithVariables,
			project.LoadOptions.WithFeatures,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationVariables,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
		)
		next()
		if errP != nil {
			return sdk.WrapError(errP, "postWorkflowJobHookCallbackHandler> Cannot load project")
		}
		wr, err := workflow.LoadRun(ctx, tx, key, workflowName, number, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			return sdk.WrapError(err, "postWorkflowJobHookCallbackHandler> Cannot load workflow run")
		}

		pv, err := project.GetAllVariableInProject(tx, wr.Workflow.ProjectID, project.WithClearPassword())
		if err != nil {
			return sdk.WrapError(err, "Cannot load project variable")
		}

		secrets, errSecret := workflow.LoadSecrets(tx, api.Cache, nil, wr, pv)
		if errSecret != nil {
			return sdk.WrapError(errSecret, "postWorkflowJobHookCallbackHandler> Cannot load secrets")
		}

		// Hide secrets in payload
		for _, s := range secrets {
			callback.Log = strings.Replace(callback.Log, s.Value, "**"+s.Name+"**", -1)
		}

		report, err := workflow.UpdateOutgoingHookRunStatus(ctx, tx, api.Cache, proj, wr, hookRunID, callback)
		if err != nil {
			return sdk.WrapError(err, "Unable to update outgoing hook run status")
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		go WorkflowSendEvent(context.Background(), api.mustDB(), api.Cache, key, report)

		report, err = updateParentWorkflowRun(ctx, api.mustDB, api.Cache, wr)
		if err != nil {
			return sdk.WrapError(err, "postWorkflowJobHookCallbackHandler")
		}

		go WorkflowSendEvent(context.Background(), api.mustDB(), api.Cache, key, report)

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

		pv, err := project.GetAllVariableInProject(db, wr.Workflow.ProjectID, project.WithClearPassword())
		if err != nil {
			return sdk.WrapError(err, "cannot load project variable")
		}

		secrets, errSecret := workflow.LoadSecrets(db, api.Cache, nil, wr, pv)
		if errSecret != nil {
			return sdk.WrapError(errSecret, "cannot load secrets")
		}
		hr.BuildParameters = append(hr.BuildParameters, sdk.VariablesToParameters("", secrets)...)
		return service.WriteJSON(w, hr, http.StatusOK)
	}
}
