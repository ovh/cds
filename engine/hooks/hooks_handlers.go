package hooks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) postGenerateWorkflowWebHookSecretHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		pKey := vars["projectKey"]
		vcsServerName := vars["vcsServer"]
		repoName, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return sdk.WithStack(err)
		}
		workflowName := vars["workflowName"]

		uuid := sdk.UUID()

		key := sdk.GenerateWorkflowWebHookSecret(s.Cfg.RepositoryWebHookKey, pKey, vcsServerName, repoName, workflowName, uuid)
		return service.WriteJSON(w, sdk.GeneratedWebhook{Key: key, UUID: uuid, HookPublicURL: s.Cfg.URLPublic}, http.StatusOK)
	}
}

func (s *Service) postGenerateRepositoryWebHookSecretHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		pKey := vars["projectKey"]
		vcsServerName := vars["vcsServer"]
		repoName, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return sdk.WithStack(err)
		}
		uuid := sdk.UUID()

		key := sdk.GenerateRepositoryWebHookSecret(s.Cfg.RepositoryWebHookKey, pKey, vcsServerName, repoName, uuid)
		return service.WriteJSON(w, sdk.GeneratedWebhook{Key: key, UUID: uuid, HookPublicURL: s.Cfg.URLPublic}, http.StatusOK)
	}
}

func (s *Service) postRepositoryEventAnalysisCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var hookCallback sdk.HookEventCallback
		if err := service.UnmarshalBody(r, &hookCallback); err != nil {
			return err
		}
		if err := s.Dao.EnqueueRepositoryEventCallback(ctx, hookCallback); err != nil {
			return err
		}
		s.Dao.enqueuedRepositoryEventCallbackIncr()
		return nil
	}
}

func (s *Service) deleteSchedulerByWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		vcsServerName := vars["vcsServer"]
		repoName, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return sdk.WithStack(err)
		}
		workflowName := vars["workflowName"]

		if err := s.removeSchedulersAndNextExecution(ctx, vcsServerName, repoName, workflowName); err != nil {
			return err
		}

		return nil
	}
}

func (s *Service) deleteSchedulerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hookID := vars["hookID"]

		exec, err := s.Dao.GetSchedulerExecution(ctx, hookID)
		if err != nil {
			return err
		}
		if exec == nil {
			return nil
		}

		if err := s.Dao.RemoveScheduler(ctx, exec.SchedulerDef.VCSName, exec.SchedulerDef.RepositoryName, exec.SchedulerDef.WorkflowName, hookID); err != nil {
			return err
		}

		return nil
	}
}

func (s *Service) getAllSchedulersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		schedulers, err := s.listAllSchedulers(ctx)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, schedulers, http.StatusOK)
	}
}

func (s *Service) getWorkflowSchedulersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		workflowName := vars["workflowName"]
		vcsServerName := vars["vcsServer"]
		repoName, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return err
		}

		schedulers, err := s.listSchedulersByWorkflow(ctx, vcsServerName, repoName, workflowName)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, schedulers, http.StatusOK)
	}
}

func (s *Service) geSchedulerExecutionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hookID := vars["hookID"]

		exec, err := s.Dao.GetSchedulerExecution(ctx, hookID)
		if err != nil {
			return err
		}
		if exec != nil {
			return service.WriteJSON(w, exec, http.StatusOK)
		}
		return sdk.WithStack(sdk.ErrNotFound)
	}
}

func (s *Service) postInstantiateSchedulerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var hooks []sdk.V2WorkflowHook
		if err := service.UnmarshalBody(r, &hooks); err != nil {
			return sdk.WithStack(err)
		}

		if err := s.instantiateScheduler(ctx, hooks); err != nil {
			return err
		}
		return nil
	}
}

func (s *Service) workflowManualHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var runRequest sdk.HookManualWorkflowRun
		if err := service.UnmarshalBody(r, &runRequest); err != nil {
			return sdk.WithStack(err)
		}
		exec, err := s.handleManualWorkflowEvent(ctx, runRequest)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, exec, http.StatusAccepted)
	}
}

func (s *Service) handleManualWorkflowEvent(ctx context.Context, runRequest sdk.HookManualWorkflowRun) (*sdk.HookRepositoryEvent, error) {
	repoKey := s.Dao.GetRepositoryMemberKey(runRequest.VCSServer, runRequest.Repository)
	if s.Dao.FindRepository(ctx, repoKey) == nil {
		if _, err := s.Dao.CreateRepository(ctx, runRequest.VCSServer, runRequest.Repository); err != nil {
			return nil, sdk.WrapError(err, "unable to create repository %s", repoKey)
		}
	}

	extractedData := sdk.HookRepositoryEventExtractData{
		CDSEventName: sdk.WorkflowHookEventNameManual,
		Commit:       runRequest.WorkflowCommit,
		Ref:          runRequest.WorkflowRef,
		Manual: sdk.HookRepositoryEventExtractedDataManual{
			Project:          runRequest.Project,
			Workflow:         runRequest.Workflow,
			TargetRepository: runRequest.TargetRepo,
			TargetCommit:     runRequest.UserRequest.Sha,
			TargetBranch:     runRequest.UserRequest.Branch,
			TargetTag:        runRequest.UserRequest.Tag,
		},
		DeprecatedAdminMFA: runRequest.AdminMFA,
	}

	exec := &sdk.HookRepositoryEvent{
		UUID:               sdk.UUID(),
		DeprecatedUserID:   runRequest.UserID,
		DeprecatedUsername: runRequest.Username,
		EventName:          sdk.WorkflowHookEventNameManual,
		VCSServerName:      runRequest.VCSServer,
		RepositoryName:     runRequest.Repository,
		Body:               nil,
		Created:            time.Now().UnixNano(),
		Status:             sdk.HookEventStatusScheduled,
		ExtractData:        extractedData,
		Initiator: &sdk.V2Initiator{
			UserID:         runRequest.UserID,
			IsAdminWithMFA: runRequest.AdminMFA,
		},
	}

	// Save event
	if err := s.Dao.SaveRepositoryEvent(ctx, exec); err != nil {
		return nil, sdk.WrapError(err, "unable to create repository event %s", exec.GetFullName())
	}

	// Enqueue event
	if err := s.Dao.EnqueueRepositoryEvent(ctx, exec); err != nil {
		return exec, sdk.WrapError(err, "unable to enqueue repository event %s", exec.GetFullName())
	}

	return exec, nil
}

func (s *Service) getOutgoingHooksExecutionsByWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		proj := vars["projectKey"]
		vcs := vars["vcsServer"]
		workflow := vars["workflowName"]
		repo, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return sdk.WithStack(err)
		}

		events, err := s.Dao.ListWorkflowRunOutgoingEvents(ctx, proj, vcs, repo, workflow)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, events, http.StatusOK)
	}
}

func (s *Service) getOutgoingHookExecutionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		vcs := vars["vcsServer"]
		proj := vars["projectKey"]
		workflow := vars["workflowName"]
		hookID := vars["hookID"]

		repo, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return sdk.WithStack(err)
		}

		k := strings.ToLower(cache.Key(workflowRunOutgoingEventRootKey, s.Dao.GetOutgoingMemberKey(proj, vcs, repo, workflow), hookID))

		var e sdk.HookWorkflowRunOutgoingEvent
		find, err := s.Cache.Get(k, &e)
		if err != nil {
			return err
		}
		if !find {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find outgoing event")
		}
		return service.WriteJSON(w, e, http.StatusOK)
	}
}

func (s *Service) workflowRunOutgoingEventHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var runRequest sdk.HookWorkflowRunEvent
		if err := service.UnmarshalBody(r, &runRequest); err != nil {
			return sdk.WithStack(err)
		}
		exec, err := s.handleWorkflowRunOutgoingEvent(ctx, runRequest)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, exec, http.StatusAccepted)
	}
}

func (s *Service) handleWorkflowRunOutgoingEvent(ctx context.Context, runRequest sdk.HookWorkflowRunEvent) (*sdk.HookWorkflowRunOutgoingEvent, error) {
	event := &sdk.HookWorkflowRunOutgoingEvent{
		UUID:    sdk.UUID(),
		Created: time.Now().UnixNano(),
		Status:  sdk.HookEventStatusScheduled,
		Event:   runRequest,
	}

	// Save event
	if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, event); err != nil {
		return nil, sdk.WrapError(err, "unable to create repository event %s", event.GetFullName())
	}

	// Enqueue event
	if err := s.Dao.EnqueueWorkflowRunOutgoingEvent(ctx, event); err != nil {
		return event, sdk.WrapError(err, "unable to enqueue repository event %s", event.GetFullName())
	}

	return event, nil
}

func (s *Service) repositoryHooksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get repository data
		vcsName := r.Header.Get(sdk.SignHeaderVCSName)
		vcsType := r.Header.Get(sdk.SignHeaderVCSType)
		eventName := r.Header.Get(sdk.SignHeaderEventName)

		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to read body: %v", err)
		}

		repoName, extractData, err := s.extractDataFromPayload(r.Header, vcsType, body, eventName)
		if err != nil {
			return err
		}

		exec, err := s.handleRepositoryEvent(ctx, vcsName, strings.ToLower(repoName), extractData, body)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return err
		}

		return service.WriteJSON(w, exec, http.StatusAccepted)
	}
}

func (s *Service) workflowWebHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projKey := vars["projectKey"]
		vcsServerName := vars["vcsServer"]
		repoName, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return sdk.WithStack(err)
		}
		repoName = strings.ToLower(repoName)
		wkfName := vars["workflowName"]
		webhookID := vars["uuid"]
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to read body: %v", err)
		}

		extractedData := sdk.HookRepositoryEventExtractData{
			WebHook: sdk.HookRepositoryEventExtractedDataWebHook{
				Project:    projKey,
				VCS:        vcsServerName,
				Repository: repoName,
				Workflow:   wkfName,
				ID:         webhookID,
			},
			CDSEventName: sdk.WorkflowHookEventNameWebHook,
		}

		exec, err := s.handleRepositoryEvent(ctx, vcsServerName, repoName, extractedData, body)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, exec, http.StatusAccepted)

	}
}

func (s *Service) repositoryWebHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projKey := vars["projectKey"]
		vcsServerType := vars["vcsServerType"]
		vcsServerName := vars["vcsServer"]

		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to read body: %v", err)
		}

		eventName, err := s.extractEventFromHeader(ctx, vcsServerType, r.Header)
		if err != nil {
			return err
		}

		repoName, extractedData, err := s.extractDataFromPayload(r.Header, vcsServerType, body, eventName)
		if err != nil {
			return err
		}
		extractedData.HookProjectKey = projKey

		exec, err := s.handleRepositoryEvent(ctx, vcsServerName, strings.ToLower(repoName), extractedData, body)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, exec, http.StatusAccepted)
	}
}

func (s *Service) handleRepositoryEvent(ctx context.Context, vcsServerName string, repoName string, extractedData sdk.HookRepositoryEventExtractData, event []byte) (*sdk.HookRepositoryEvent, error) {
	repoKey := s.Dao.GetRepositoryMemberKey(vcsServerName, repoName)
	if s.Dao.FindRepository(ctx, repoKey) == nil {
		if _, err := s.Dao.CreateRepository(ctx, vcsServerName, repoName); err != nil {
			return nil, sdk.WrapError(err, "unable to create repository %s", repoKey)
		}
	}

	exec := &sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		EventName:      extractedData.CDSEventName, // WorkflowHookEventPush, sdk.WorkflowHookEventPullRequest, sdk.WorkflowHookEventPullRequestComment
		EventType:      extractedData.CDSEventType, // WorkflowHookEventPullRequestTypeOpened, WorkflowHookEventPullRequestTypeEdited, etc...
		VCSServerName:  vcsServerName,
		RepositoryName: repoName,
		Body:           event,
		Created:        time.Now().UnixNano(),
		Status:         sdk.HookEventStatusScheduled,
		ExtractData:    extractedData,
	}

	// Save event
	if err := s.Dao.SaveRepositoryEvent(ctx, exec); err != nil {
		return nil, sdk.WrapError(err, "unable to create repository event %s", exec.GetFullName())
	}

	// Enqueue event
	if err := s.Dao.EnqueueRepositoryEvent(ctx, exec); err != nil {
		return exec, sdk.WrapError(err, "unable to enqueue repository event %s", exec.GetFullName())
	}

	return exec, nil
}

func (s *Service) extractEventFromHeader(ctx context.Context, vcsServerType string, header http.Header) (string, error) {
	var eventName string
	var headerName string
	switch vcsServerType {
	case sdk.VCSTypeBitbucketServer:
		headerName = BitbucketHeader
	case sdk.VCSTypeGithub:
		headerName = GithubHeader
	case sdk.VCSTypeGitea:
		headerName = GiteaHeader
	case sdk.VCSTypeGitlab:
		headerName = GitlabHeader
	default:
		log.Warn(ctx, "invalid vcs server of type %s", vcsServerType)
		return "", sdk.WithStack(sdk.ErrNotImplemented)
	}
	if v, has := header[headerName]; has && len(v) > 0 {
		eventName = v[0]
	}
	if eventName == "" {
		return "", sdk.WrapError(sdk.ErrNotFound, "unable to found event from header")
	}
	return eventName, nil
}

func (s *Service) webhookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the webhook
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Invalid uuid or name")
		}

		//Load the task
		webHook := s.Dao.FindTask(ctx, uuid)
		if webHook == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown uuid")
		}

		//Check method
		confValue := webHook.Config[sdk.WebHookModelConfigMethod]
		if r.Method != confValue.Value {
			return sdk.WrapError(sdk.ErrMethodNotAllowed, "Unsupported method %s : %v", r.Method, webHook.Config)
		}

		//Read the body
		req, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "Unable to read request")
		}

		//Prepare a web hook execution
		exec := &sdk.TaskExecution{
			Timestamp: time.Now().UnixNano(),
			Type:      webHook.Type,
			UUID:      webHook.UUID,
			Config:    webHook.Config,
			Status:    TaskExecutionScheduled,
			WebHook: &sdk.WebHookExecution{
				RequestBody:   req,
				RequestHeader: r.Header,
				RequestURL:    r.URL.RawQuery,
			},
		}

		//Save the web hook execution
		s.Dao.SaveTaskExecution(exec)

		//Return the execution
		return service.WriteJSON(w, exec, http.StatusOK)
	}
}

func (s *Service) startTasksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := s.startTasks(ctx); err != nil {
			return sdk.WithStack(err)
		}
		return nil
	}
}

func (s *Service) stopTasksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := s.stopTasks(ctx); err != nil {
			return sdk.WithStack(err)
		}
		return nil
	}
}

func (s *Service) startTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Invalid uuid")
		}

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown uuid")
		}

		//Start the task
		if _, err := s.startTask(ctx, t); err != nil {
			return sdk.WrapError(err, "Start task")
		}
		return nil
	}
}

func (s *Service) stopTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Invalid uuid")
		}

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown uuid")
		}

		//Stop the task
		if err := s.stopTask(ctx, t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Stop task")
		}
		return nil
	}
}

func (s *Service) postTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeHook from the body
		var hook sdk.NodeHook
		if err := service.UnmarshalBody(r, &hook); err != nil {
			return sdk.WithStack(err)
		}
		if err := s.addTask(ctx, &hook); err != nil {
			return sdk.WithStack(err)
		}
		return nil
	}
}

func (s *Service) postAndExecuteTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeOutgoingHook from the body
		var nr sdk.WorkflowNodeRun
		if err := service.UnmarshalBody(r, &nr); err != nil {
			return sdk.WrapError(err, "Hooks> postAndExecuteTaskHandler")
		}
		t, e, err := s.addAndExecuteTask(ctx, nr)
		if err != nil {
			return sdk.WrapError(err, "Hooks> postAndExecuteTaskHandler> unable to add Task")
		}
		t.Executions = []sdk.TaskExecution{e}
		return service.WriteJSON(w, t, http.StatusOK)
	}
}

const (
	sortKeyNbExecutionsTotal = "nb_executions_total"
	sortKeyNbExecutionsTodo  = "nb_executions_todo"
)

func (s *Service) getTasksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sortParams, err := api.QuerySort(r)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		for k := range sortParams {
			if k != sortKeyNbExecutionsTotal && k != sortKeyNbExecutionsTodo {
				return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("invalid given sort key"))
			}
		}

		tasks, err := s.Dao.FindAllTasks(ctx)
		if err != nil {
			return sdk.WithStack(err)
		}

		execs, err := s.Dao.FindAllTaskExecutionsForTasks(ctx, tasks...)
		if err != nil {
			return sdk.WithStack(err)
		}

		m := make(map[string][]sdk.TaskExecution, len(tasks))
		for _, e := range execs {
			m[e.UUID] = append(m[e.UUID], e)
		}

		for i, t := range tasks {
			var nbTodo int
			for _, e := range m[t.UUID] {
				if e.ProcessingTimestamp == 0 {
					nbTodo++
				}
			}
			tasks[i].NbExecutionsTotal = len(m[t.UUID])
			tasks[i].NbExecutionsTodo = nbTodo
		}

		for k, p := range sortParams {
			switch k {
			case sortKeyNbExecutionsTotal:
				sort.Slice(tasks, func(i, j int) bool {
					return api.SortCompareInt(tasks[i].NbExecutionsTotal, tasks[j].NbExecutionsTotal, p)
				})
			case sortKeyNbExecutionsTodo:
				sort.Slice(tasks, func(i, j int) bool {
					return api.SortCompareInt(tasks[i].NbExecutionsTodo, tasks[j].NbExecutionsTodo, p)
				})
			}
		}

		return service.WriteJSON(w, tasks, http.StatusOK)
	}
}

func (s *Service) putTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Hooks> putTaskHandler> invalid uuid")
		}

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> putTaskHandler> unknown uuid")
		}

		//Stop the task
		if err := s.stopTask(ctx, t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> putTaskHandler> stop task")
		}

		//Save it
		if err := s.Dao.SaveTask(t); err != nil {
			return sdk.WrapError(err, "Unable to save task %v", t)
		}

		//Start the task
		if _, err := s.startTask(ctx, t); err != nil {
			return sdk.WrapError(err, "Unable start task %v", t)
		}

		return nil
	}
}

func (s *Service) getTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to load executions")
		}

		t.Executions = execs

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (s *Service) deleteTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		//Stop the task
		if err := s.stopTask(ctx, t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> putTaskHandler> stop task")
		}

		return s.deleteTask(ctx, t)
	}
}

func (s *Service) getTaskExecutionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}
		t.Executions = execs

		sort.Slice(t.Executions, func(i, j int) bool {
			return t.Executions[i].Timestamp > t.Executions[j].Timestamp
		})

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (s *Service) deleteAllTaskExecutionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Stop the task
		if err := s.stopTask(ctx, t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> deleteAllTaskExecutionsHandler> stop task")
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}
		for i := range execs {
			if err := s.Dao.DeleteTaskExecution(&execs[i]); err != nil {
				return err
			}
		}

		//Start the task
		if _, err := s.startTask(ctx, t); err != nil {
			return sdk.WrapError(err, "Unable start task %+v", t)
		}

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (s *Service) deleteTaskBulkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		hooks := map[string]sdk.NodeHook{}
		if err := service.UnmarshalBody(r, &hooks); err != nil {
			return sdk.WithStack(err)
		}

		for _, h := range hooks {
			//Load the task
			t := s.Dao.FindTask(ctx, h.UUID)
			if t == nil {
				continue
			}

			//Stop the task
			if err := s.stopTask(ctx, t); err != nil {
				return sdk.WrapError(sdk.ErrNotFound, "Stop task %s", err)
			}

			if err := s.deleteTask(ctx, t); err != nil {
				return err
			}

		}

		return nil
	}
}

func (s *Service) postTaskBulkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeHook from the body
		hooks := map[string]sdk.NodeHook{}
		if err := service.UnmarshalBody(r, &hooks); err != nil {
			return sdk.WithStack(err)
		}

		for _, hook := range hooks {
			err := s.updateTask(ctx, &hook)
			if err == errNoTask {
				if err := s.addTask(ctx, &hook); err != nil {
					return sdk.WithStack(err)
				}
			} else if err != nil {
				return sdk.WithStack(err)
			}
		}
		return service.WriteJSON(w, hooks, http.StatusOK)
	}
}

func (s *Service) addTask(ctx context.Context, h *sdk.NodeHook) error {
	//Parse the hook as a task
	t, err := s.nodeHookToTask(h)
	if err != nil {
		return sdk.WrapError(err, "Unable to parse hook")
	}

	//Save the task
	if err := s.Dao.SaveTask(t); err != nil {
		return sdk.WrapError(err, "unable to addTask %v", t)
	}

	//Start the task
	if _, err := s.startTask(ctx, t); err != nil {
		return sdk.WrapError(err, "Unable start task %v", t)
	}
	return nil
}

func (s *Service) addAndExecuteTask(ctx context.Context, nr sdk.WorkflowNodeRun) (sdk.Task, sdk.TaskExecution, error) {
	// Parse the hook as a task
	t, err := s.nodeRunToTask(nr)
	if err != nil {
		return t, sdk.TaskExecution{}, sdk.WrapError(err, "Hooks> addAndExecuteTask> Unable to parse node run (%+v)", nr)
	}
	// Save the task
	if err := s.Dao.SaveTask(&t); err != nil {
		return t, sdk.TaskExecution{}, sdk.WrapError(err, "unable to save task %v", t)
	}

	// Start the task
	e, err := s.startTask(ctx, &t)
	if err != nil {
		return t, sdk.TaskExecution{}, sdk.WrapError(err, "Unable start task %+v", t)
	}

	return t, *e, nil
}

var errNoTask = errors.New("task not found")

func (s *Service) updateTask(ctx context.Context, h *sdk.NodeHook) error {
	//Parse the hook as a task
	t, err := s.nodeHookToTask(h)
	if err != nil {
		return sdk.WrapError(err, "Unable to parse hook")
	}

	task := s.Dao.FindTask(ctx, t.UUID)
	if task == nil {
		return errNoTask
	}

	task.Config = t.Config
	_ = s.stopTask(ctx, t)
	execs, _ := s.Dao.FindAllTaskExecutions(ctx, t)
	for _, e := range execs {
		if e.Status == TaskExecutionScheduled {
			if err := s.Dao.DeleteTaskExecution(&e); err != nil {
				log.Error(ctx, "unable to delete previous task execution: %v", err)
			}
		}
	}
	if _, err := s.startTask(ctx, t); err != nil {
		return sdk.WrapError(err, "Unable start task %+v", t)
	}
	// Save the task
	if err := s.Dao.SaveTask(t); err != nil {
		return sdk.WrapError(err, "unable to save task %v", t)
	}
	return nil
}

func (s *Service) deleteTask(ctx context.Context, t *sdk.Task) error {
	switch t.Type {
	case TypeGerrit:
		s.stopGerritHookTask(t)
	}

	//Delete the task
	return s.Dao.DeleteTask(ctx, t)
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := s.NewMonitoringStatus()

	if s.Dao.store == nil {
		return m
	}

	// hook queue in status
	status := sdk.MonitoringStatusOK
	size, errQ := s.Dao.QueueLen()
	if errQ != nil {
		log.Error(ctx, "Status> Unable to retrieve queue len: %v", errQ)
	}

	if size >= 100 {
		status = sdk.MonitoringStatusAlert
	} else if size >= 10 {
		status = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Queue", Value: fmt.Sprintf("%d", size), Status: status})

	// hook balance in status
	in, out := s.Dao.TaskExecutionsBalance()

	status = sdk.MonitoringStatusOK
	if float64(in) > float64(out) {
		status = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Balance", Value: fmt.Sprintf("%d/%d", in, out), Status: status})

	hookEventIn, hookEventOut := s.Dao.RepositoryEventBalance()
	status = sdk.MonitoringStatusOK
	if float64(hookEventIn) > float64(hookEventOut) {
		status = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "BalanceRepositoryEvent", Value: fmt.Sprintf("%d/%d", hookEventIn, hookEventOut), Status: status})

	hookEventCallbackIn, hookEventCallbackOut := s.Dao.RepositoryEventCallbackBalance()
	status = sdk.MonitoringStatusOK
	if float64(hookEventCallbackIn) > float64(hookEventCallbackOut) {
		status = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "BalanceRepositoryEventCallbackEvent", Value: fmt.Sprintf("%d/%d", hookEventCallbackIn, hookEventCallbackOut), Status: status})

	hookOutgoingEventIn, hookOutgoingEventOut := s.Dao.OutgoingEventCallbackBalance()
	status = sdk.MonitoringStatusOK
	if float64(hookOutgoingEventIn) > float64(hookOutgoingEventOut) {
		status = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "BalanceOutgoingEvent", Value: fmt.Sprintf("%d/%d", hookOutgoingEventIn, hookOutgoingEventOut), Status: status})

	var nbHooksKafkaTotal int64

	tasks, err := s.Dao.FindAllTasks(ctx)
	if err != nil {
		log.Error(ctx, "Status> Unable to find all tasks: %v", err)
	}

	for _, t := range tasks {
		if t.Type == TypeKafka {
			nbHooksKafkaTotal++
		}

		if t.Stopped {
			m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Task Stopped", Value: t.UUID, Status: sdk.MonitoringStatusWarn})
		}
	}

	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Hook Kafka", Value: fmt.Sprintf("%d", nbHooksKafkaTotal), Status: status})

	statusConsumer := sdk.MonitoringStatusOK
	if nbKafkaConsumers > nbHooksKafkaTotal {
		statusConsumer = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Hook Kafka Consumers", Value: fmt.Sprintf("%d", nbKafkaConsumers), Status: statusConsumer})

	return m
}

func (s *Service) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(ctx), status)
	}
}

func (s *Service) getTaskExecutionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		timestamp := vars["timestamp"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}

		for _, e := range execs {
			if strconv.FormatInt(e.Timestamp, 10) == timestamp {
				return service.WriteJSON(w, e, http.StatusOK)
			}
		}

		return sdk.WithStack(sdk.ErrNotFound)
	}
}

func (s *Service) postStopTaskExecutionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		timestamp := vars["timestamp"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}

		for i := range execs {
			e := &execs[i]
			if (strconv.FormatInt(e.Timestamp, 10) == timestamp && e.Status == TaskExecutionDoing) || e.Status == TaskExecutionScheduled || e.Status == TaskExecutionEnqueued {
				e.Status = TaskExecutionDone
				e.LastError = TaskExecutionDone
				e.NbErrors = s.Cfg.RetryError + 1
				s.Dao.SaveTaskExecution(e)
				log.Info(ctx, "Hooks> postStopTaskExecutionHandler> task executions %s:%v has been stopped", uuid, timestamp)
				return nil
			}
		}

		return nil
	}
}

func (s *Service) postMaintenanceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		enableS := api.FormString(r, "enable")
		enable, err := strconv.ParseBool(enableS)
		if err != nil {
			return sdk.WrapError(err, "unable to parse maintenance params")
		}

		if err := s.Dao.store.SetWithTTL(MaintenanceHookKey, enable, 0); err != nil {
			return sdk.WrapError(err, "unable to save maintenance state")
		}
		if err := s.Dao.store.Publish(ctx, MaintenanceHookQueue, fmt.Sprintf("%v", enable)); err != nil {
			return sdk.WrapError(err, "unable to publish maintenance state")
		}
		s.Maintenance = enable
		return nil
	}
}
