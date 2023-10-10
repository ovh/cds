package hooks

import (
	"context"
	"fmt"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/telemetry"
)

func (s *Service) triggerWorkflows(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerWorkflows")
	defer next()

	log.Info(ctx, "triggering workflow for event [%s] %s", hre.EventName, hre.GetFullName())

	// Check if we know the user that trigger the event
	if hre.UserID == "" {
		r, err := s.Client.RetrieveHookEventUser(ctx, sdk.HookRetrieveUserRequest{
			ProjectKey:     hre.WorkflowHooks[0].ProjectKey,
			VCSServerName:  hre.VCSServerName,
			VCSServerType:  hre.VCSServerType,
			RepositoryName: hre.RepositoryName,
			Commit:         hre.ExtractData.Commit,
			Branch:         hre.ExtractData.Branch,
			SignKey:        hre.SignKey,
		})
		if err != nil {
			return err
		}
		if r.UserID == "" {
			hre.Status = sdk.HookEventStatusSkipped
			hre.LastError = fmt.Sprintf("User with key %s not found in CDS", hre.SignKey)
		}
		hre.UserID = r.UserID
		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}
	}

	if hre.UserID != "" {
		allEnded := true
		for i := range hre.WorkflowHooks {
			wh := &hre.WorkflowHooks[i]
			if wh.Status == sdk.HookEventWorkflowStatusScheduler {
				targetCommit := "HEAD"
				if wh.Type == sdk.WorkflowHookTypeRepository {
					targetCommit = hre.ExtractData.Commit
				}
				runRequest := sdk.V2WorkflowRunRequest{
					UserID: hre.UserID,
					Branch: wh.TargetBranch,
					Commit: targetCommit,
				}
				if _, err := s.Client.WorkflowV2RunFromHook(ctx, wh.ProjectKey, wh.VCSIdentifier, wh.RepositoryIdentifier, wh.WorkflowName,
					runRequest, cdsclient.WithQueryParameter("branch", wh.Branch)); err != nil {
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
	}

	return nil
}