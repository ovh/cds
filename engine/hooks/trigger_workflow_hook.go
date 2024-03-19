package hooks

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (s *Service) triggerWorkflowHooks(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerWorkflowHooks")
	defer next()
	log.Info(ctx, "triggering workflow hooks for event [%s] %s", hre.EventName, hre.GetFullName())

	if hre.EventName == sdk.WorkflowHookManual {
		var userRequest sdk.V2WorkflowRunManualRequest
		if err := json.Unmarshal(hre.Body, &userRequest); err != nil {
			return err
		}
		var destRef string
		if userRequest.Git.Branch != "" {
			destRef = sdk.GitRefBranchPrefix + userRequest.Git.Branch
		} else {
			destRef = sdk.GitRefTagPrefix + userRequest.Git.Tag
		}
		// Create Manual Hook
		hre.WorkflowHooks = []sdk.HookRepositoryEventWorkflow{
			{
				ProjectKey:           hre.ExtractData.ProjectManual,
				VCSIdentifier:        hre.VCSServerName,
				RepositoryIdentifier: hre.RepositoryName,
				WorkflowName:         hre.ExtractData.WorkflowManual,
				Type:                 sdk.WorkflowHookManual,
				Status:               sdk.HookEventWorkflowStatusScheduler,
				Ref:                  destRef,
				TargetBranch:         userRequest.Git.Sha,
			},
		}
	} else {
		// Retrieve hooks from API
		request := sdk.HookListWorkflowRequest{
			HookEventUUID:       hre.UUID,
			Ref:                 hre.ExtractData.Ref,
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
			request.AnayzedProjectKeys = append(request.AnayzedProjectKeys, a.ProjectKey)
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
				TargetBranch:         wh.Data.TargetBranch,
				ModelFullName:        wh.Data.Model,
			}
			if wh.Type == sdk.WorkflowHookTypeWorkflow || wh.Type == sdk.WorkflowHookTypeWorkerModel {
				w.TargetCommit = "HEAD"
			}
			hre.WorkflowHooks = append(hre.WorkflowHooks, w)
		}
	}

	hre.Status = sdk.HookEventStatusSignKey
	return s.triggerGetSigningKey(ctx, hre)
}
