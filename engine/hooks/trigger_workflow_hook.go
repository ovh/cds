package hooks

import (
	"context"
	"encoding/json"

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

	if hre.EventName == sdk.WorkflowHookManual {
		var userRequest sdk.V2WorkflowRunManualRequest
		if err := json.Unmarshal(hre.Body, &userRequest); err != nil {
			return err
		}

		// Get workflow definition
		e, err := s.Client.EntityGet(ctx, hre.ExtractData.ProjectManual, hre.VCSServerName, hre.RepositoryName, sdk.EntityTypeWorkflow, hre.ExtractData.WorkflowManual,
			cdsclient.WithQueryParameter("ref", hre.ExtractData.Ref), cdsclient.WithQueryParameter("commit", hre.ExtractData.Commit))
		if err != nil {
			return err
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
			ProjectKey:           hre.ExtractData.ProjectManual,
			VCSIdentifier:        hre.VCSServerName,
			RepositoryIdentifier: hre.RepositoryName,
			WorkflowName:         hre.ExtractData.WorkflowManual,
			Type:                 sdk.WorkflowHookTypeManual,
			Status:               sdk.HookEventWorkflowStatusScheduler,
			Ref:                  hre.ExtractData.Ref,
			Commit:               hre.ExtractData.Commit,
			TargetCommit:         userRequest.Sha,
			Data: sdk.V2WorkflowHookData{
				VCSServer:      workflowVCS,
				RepositoryName: workflowRepo,
				TargetBranch:   userRequest.Branch,
				TargetTag:      userRequest.Tag,
			},
		}
		// Create Manual Hook
		hre.WorkflowHooks = []sdk.HookRepositoryEventWorkflow{wh}
	} else {
		// Retrieve hooks from API
		request := sdk.HookListWorkflowRequest{
			HookEventUUID:       hre.UUID,
			Ref:                 hre.ExtractData.Ref,
			Sha:                 hre.ExtractData.Commit,
			Models:              hre.ModelUpdated,
			Workflows:           hre.WorkflowUpdated,
			Paths:               hre.ExtractData.Paths,
			RepositoryEventName: hre.EventName,
			RepositoryEventType: hre.EventType,
			VCSName:             hre.VCSServerName,
			RepositoryName:      hre.RepositoryName,
			AnayzedProjectKeys:  sdk.StringSlice{},
		}
		for _, a := range hre.Analyses {
			// Only retrieve hooks from project where analysis is OK
			if a.Status == sdk.RepositoryAnalysisStatusSucceed {
				request.AnayzedProjectKeys = append(request.AnayzedProjectKeys, a.ProjectKey)
			}
		}
		request.AnayzedProjectKeys.Unique()
		workflowHooks, err := s.Client.ListWorkflowToTrigger(ctx, request)
		if err != nil {
			return err
		}

		// If no hooks, we can end the process
		if len(workflowHooks) == 0 {
			hre.Status = sdk.HookEventStatusDone
			if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
				return err
			}
			if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, *hre); err != nil {
				return err
			}
			return nil
		}

		hre.WorkflowHooks = make([]sdk.HookRepositoryEventWorkflow, 0, len(workflowHooks))
		for _, wh := range workflowHooks {
			w := sdk.HookRepositoryEventWorkflow{
				ProjectKey:           wh.ProjectKey,
				VCSIdentifier:        wh.VCSName,
				RepositoryIdentifier: wh.RepositoryName,
				WorkflowName:         wh.WorkflowName,
				EntityID:             wh.EntityID,
				Type:                 wh.Type,
				Status:               sdk.HookEventWorkflowStatusScheduler,
				Ref:                  wh.Ref,
				ModelFullName:        wh.Data.Model,
				PathFilters:          wh.Data.PathFilter,
				Commit:               wh.Commit,
				Data:                 wh.Data,
			}
			hre.WorkflowHooks = append(hre.WorkflowHooks, w)
		}
	}

	hre.Status = sdk.HookEventStatusSignKey
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	return s.triggerGetSigningKey(ctx, hre)
}
