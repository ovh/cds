package hooks

import (
	"context"

	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"

	"github.com/ovh/cds/sdk"
)

func (s *Service) triggerGetWorkflowHooks(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerWorkflowHooks")
	defer next()
	log.Info(ctx, "triggering workflow hooks for event [%s] %s", hre.EventName, hre.GetFullName())

	switch hre.EventName {
	case sdk.WorkflowHookEventNameManual:
		if err := s.handleManualHook(ctx, hre); err != nil {
			return err
		}
	case sdk.WorkflowHookEventNameWebHook:
		if err := s.handleWebhookHook(ctx, hre); err != nil {
			return err
		}
	case sdk.WorkflowHookEventNameScheduler:
		if err := s.handleScheduler(ctx, hre); err != nil {
			return err
		}
	default:
		if err := s.handleWorkflowHook(ctx, hre); err != nil {
			return err
		}
	}

	// If no hooks, we can end the process
	if len(hre.WorkflowHooks) == 0 {
		hre.Status = sdk.HookEventStatusDone
		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}
		if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, hre.UUID); err != nil {
			return err
		}
		return nil
	}

	switch hre.EventName {
	case sdk.WorkflowHookEventNameWebHook:
		hre.Status = sdk.HookEventStatusGitInfo
	default:
		hre.Status = sdk.HookEventStatusSignKey
	}

	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}

	return s.executeEvent(ctx, hre)
}

func (s *Service) handleScheduler(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	wh := sdk.HookRepositoryEventWorkflow{
		Type:                 sdk.WorkflowHookTypeScheduler,
		Status:               sdk.HookEventWorkflowStatusScheduled,
		ProjectKey:           hre.ExtractData.Scheduler.TargetProject,
		VCSIdentifier:        hre.VCSServerName,
		RepositoryIdentifier: hre.RepositoryName,
		WorkflowName:         hre.ExtractData.Scheduler.TargetWorkflow,
		Ref:                  hre.ExtractData.Ref,
		Commit:               hre.ExtractData.Commit,
		Data: sdk.V2WorkflowHookData{
			VCSServer:      hre.ExtractData.Scheduler.TargetVCS,
			RepositoryName: hre.ExtractData.Scheduler.TargetRepo,
			Cron:           hre.ExtractData.Scheduler.Cron,
			CronTimeZone:   hre.ExtractData.Scheduler.Timezone,
		},
	}
	hre.WorkflowHooks = []sdk.HookRepositoryEventWorkflow{wh}
	return nil
}

func (s *Service) handleWorkflowHook(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	// Retrieve hooks from API
	request := sdk.HookListWorkflowRequest{
		HookEventUUID:       hre.UUID,
		Ref:                 hre.ExtractData.Ref,
		PullRequestRefTo:    hre.ExtractData.PullRequestRefTo,
		Sha:                 hre.ExtractData.Commit,
		Models:              hre.ModelUpdated,
		Workflows:           hre.WorkflowUpdated,
		Paths:               hre.ExtractData.Paths,
		RepositoryEventName: hre.EventName,
		RepositoryEventType: hre.EventType,
		VCSName:             hre.VCSServerName,
		RepositoryName:      hre.RepositoryName,
		AnalyzedProjectKeys: sdk.StringSlice{},
		SkippedWorkflows:    hre.SkippedWorkflows,
		SkippedHooks:        hre.SkippedHooks,
	}
	for _, a := range hre.Analyses {
		// Only retrieve hooks from project where analysis is OK
		if a.Status == sdk.RepositoryAnalysisStatusSucceed {
			request.AnalyzedProjectKeys = append(request.AnalyzedProjectKeys, a.ProjectKey)
		}
	}
	request.AnalyzedProjectKeys.Unique()
	workflowHooks, err := s.Client.ListWorkflowToTrigger(ctx, request)
	if err != nil {
		return err
	}

	// If no hooks, we can end the process
	if len(workflowHooks) == 0 {
		return nil
	}

	hre.WorkflowHooks = make([]sdk.HookRepositoryEventWorkflow, 0, len(workflowHooks))
	for _, wh := range workflowHooks {
		if hre.ExtractData.HookProjectKey != "" && hre.ExtractData.HookProjectKey != wh.ProjectKey {
			log.Info(ctx, "Workflow hook %s not on the right project, got %s want %s", wh.Data.RepositoryEvent, wh.ProjectKey, hre.ExtractData.HookProjectKey)
			continue
		}
		w := sdk.HookRepositoryEventWorkflow{
			ProjectKey:           wh.ProjectKey,
			VCSIdentifier:        wh.VCSName,
			RepositoryIdentifier: wh.RepositoryName,
			WorkflowName:         wh.WorkflowName,
			EntityID:             wh.EntityID,
			Type:                 wh.Type,
			Status:               sdk.HookEventWorkflowStatusScheduled,
			Ref:                  wh.Ref,
			ModelFullName:        wh.Data.Model,
			PathFilters:          wh.Data.PathFilter,
			Commit:               wh.Commit,
			Data:                 wh.Data,
		}
		if wh.Type == sdk.WorkflowHookTypeRepository {
			w.TargetCommit = hre.ExtractData.Commit
		}
		hre.WorkflowHooks = append(hre.WorkflowHooks, w)
	}
	return nil
}

func (s *Service) handleWebhookHook(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	// Get workflow definition
	e, err := s.Client.EntityGet(ctx, hre.ExtractData.WebHook.Project, hre.ExtractData.WebHook.VCS, hre.ExtractData.WebHook.Repository, sdk.EntityTypeWorkflow, hre.ExtractData.WebHook.Workflow)
	if err != nil {
		return err
	}
	var wk sdk.V2Workflow
	if err := yaml.Unmarshal([]byte(e.Data), &wk); err != nil {
		return err
	}
	workflowVCS := hre.ExtractData.WebHook.VCS
	workflowRepo := hre.ExtractData.WebHook.Repository
	if wk.Repository != nil && wk.Repository.VCSServer != "" {
		workflowVCS = wk.Repository.VCSServer
		workflowRepo = wk.Repository.Name
	}

	wh := sdk.HookRepositoryEventWorkflow{
		ProjectKey:           hre.ExtractData.WebHook.Project,
		VCSIdentifier:        hre.ExtractData.WebHook.VCS,        // vcs workflow definition
		RepositoryIdentifier: hre.ExtractData.WebHook.Repository, // repo workflow definition
		WorkflowName:         hre.ExtractData.WebHook.Workflow,
		Type:                 sdk.WorkflowHookTypeWebhook,
		Status:               sdk.HookEventWorkflowStatusScheduled,
		Ref:                  e.Ref,
		Commit:               e.Commit,
		Data: sdk.V2WorkflowHookData{
			VCSServer:      workflowVCS,  // vcs used by the workflow in the git context
			RepositoryName: workflowRepo, // repo used by the workflow in the git context

		},
	}
	if e.UserID != nil {
		hre.Initiator = &sdk.V2Initiator{
			UserID: *e.UserID,
		}
	}

	hre.WorkflowHooks = []sdk.HookRepositoryEventWorkflow{wh}
	return nil
}

func (s *Service) handleManualHook(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	// Get workflow definition
	e, err := s.Client.EntityGet(ctx, hre.ExtractData.Manual.Project, hre.VCSServerName, hre.RepositoryName, sdk.EntityTypeWorkflow, hre.ExtractData.Manual.Workflow,
		cdsclient.WithQueryParameter("ref", hre.ExtractData.Ref), cdsclient.WithQueryParameter("commit", hre.ExtractData.Commit))
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		// Fallback on HEAD commit
		e, err = s.Client.EntityGet(ctx, hre.ExtractData.Manual.Project, hre.VCSServerName, hre.RepositoryName, sdk.EntityTypeWorkflow, hre.ExtractData.Manual.Workflow,
			cdsclient.WithQueryParameter("ref", hre.ExtractData.Ref), cdsclient.WithQueryParameter("commit", "HEAD"))
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
			// Fallback on default branch, head commit
			e, err = s.Client.EntityGet(ctx, hre.ExtractData.Manual.Project, hre.VCSServerName, hre.RepositoryName, sdk.EntityTypeWorkflow, hre.ExtractData.Manual.Workflow)
			if err != nil {
				return err
			}
		}
	}
	var wk sdk.V2Workflow
	if err := yaml.Unmarshal([]byte(e.Data), &wk); err != nil {
		return err
	}
	workflowVCS := hre.VCSServerName
	workflowRepo := hre.RepositoryName
	if wk.Repository != nil && wk.Repository.VCSServer != "" {
		workflowVCS = wk.Repository.VCSServer
		workflowRepo = wk.Repository.Name
	}

	wh := sdk.HookRepositoryEventWorkflow{
		ProjectKey:           hre.ExtractData.Manual.Project,
		VCSIdentifier:        hre.VCSServerName,
		RepositoryIdentifier: hre.RepositoryName,
		WorkflowName:         hre.ExtractData.Manual.Workflow,
		Type:                 sdk.WorkflowHookTypeManual,
		Status:               sdk.HookEventWorkflowStatusScheduled,
		Ref:                  hre.ExtractData.Ref,
		Commit:               hre.ExtractData.Commit,
		TargetCommit:         hre.ExtractData.Manual.TargetCommit,
		Data: sdk.V2WorkflowHookData{
			VCSServer:      workflowVCS,
			RepositoryName: workflowRepo,
			TargetBranch:   hre.ExtractData.Manual.TargetBranch,
			TargetTag:      hre.ExtractData.Manual.TargetTag,
		},
	}
	// Create Manual Hook
	hre.WorkflowHooks = []sdk.HookRepositoryEventWorkflow{wh}
	return nil
}
