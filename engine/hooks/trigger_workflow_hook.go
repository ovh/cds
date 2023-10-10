package hooks

import (
	"context"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (s *Service) triggerWorkflowHooks(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerWorkflowHooks")
	defer next()

	log.Info(ctx, "triggering workflow hooks for event [%s] %s", hre.EventName, hre.GetFullName())
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
			Branch:               wh.Branch,
			TargetBranch:         wh.Data.TargetBranch,
			ModelFullName:        wh.Data.Model,
		}
		hre.WorkflowHooks = append(hre.WorkflowHooks, w)
	}
	if hre.UserID == "" {
		hre.Status = sdk.HookEventStatusSignKey
	} else {
		hre.Status = sdk.HookEventStatusWorkflow
	}
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}

	switch hre.Status {
	case sdk.HookEventStatusSignKey:
		return s.triggerGetSigningKey(ctx, hre)
	case sdk.HookEventStatusWorkflow:
		return s.triggerWorkflows(ctx, hre)
	}

	return nil
}
