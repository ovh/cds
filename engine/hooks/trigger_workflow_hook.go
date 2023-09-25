package hooks

import (
	"context"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/telemetry"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (s *Service) triggerWorkflowHooks(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerWorkflowHooks")
	defer next()

	log.Info(ctx, "triggering workflow hooks for event [%s] %s", hre.EventName, hre.GetFullName())
	if len(hre.WorkflowHooks) == 0 {
		request := sdk.HookListWorkflowRequest{
			Branch:              hre.ExtractData.Branch,
			Models:              hre.ModelUpdated,
			Workflows:           hre.WorkflowUpdated,
			Paths:               hre.ExtractData.Paths,
			RepositoryEventName: hre.EventName,
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
				Branch:               wh.Branch,
				TargetBranch:         wh.Data.TargetBranch,
			}
			hre.WorkflowHooks = append(hre.WorkflowHooks, w)

		}
		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}
	}

	allEnded := true
	for i := range hre.WorkflowHooks {
		wh := &hre.WorkflowHooks[i]
		if wh.Status == sdk.HookEventWorkflowStatusScheduler {
			targetCommit := "HEAD"
			if wh.Type == sdk.WorkflowHookTypeRepository {
				targetCommit = hre.ExtractData.Commit
			}
			if _, err := s.Client.WorkflowV2RunFromHook(ctx, wh.ProjectKey, wh.VCSIdentifier, wh.RepositoryIdentifier, wh.WorkflowName,
				cdsclient.WithQueryParameter("branch", wh.Branch),
				cdsclient.WithQueryParameter("target_branch", wh.TargetBranch),
				cdsclient.WithQueryParameter("target_commit", targetCommit)); err != nil {
				log.ErrorWithStackTrace(ctx, err)
				allEnded = false
				continue
			}
			wh.Status = sdk.HookEventWorkflowStatusDone
			if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
				return err
			}
		}
	}

	if allEnded {
		hre.Status = sdk.HookEventStatusDone
		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}
		if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, *hre); err != nil {
			return err
		}
	}

	return nil
}
