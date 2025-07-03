package hooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/glob"
	"github.com/ovh/cds/sdk/telemetry"
)

func (s *Service) triggerWorkflows(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerWorkflows")
	defer next()

	log.Info(ctx, "triggering workflow for event [%s] %s", hre.EventName, hre.GetFullName())

	// Check if we know the user that trigger the event
	if hre.SignKey != "" && (hre.Initiator == nil || (hre.Initiator.UserID == "" && hre.Initiator.VCSUsername == "")) {
		var req sdk.HookRetrieveUserRequest
		switch {
		case hre.ExtractData.WorkflowRun.OutgoingHookEventUUID != "":
			req = sdk.HookRetrieveUserRequest{
				ProjectKey:     hre.WorkflowHooks[0].ProjectKey,
				VCSServerName:  hre.ExtractData.WorkflowRun.TargetVCS,
				RepositoryName: hre.ExtractData.WorkflowRun.TargetRepository,
				Commit:         hre.WorkflowHooks[0].TargetCommit,
				SignKey:        hre.SignKey,
				HookEventUUID:  hre.UUID,
			}
		default:
			req = sdk.HookRetrieveUserRequest{
				ProjectKey:     hre.WorkflowHooks[0].ProjectKey,
				VCSServerName:  hre.VCSServerName,
				RepositoryName: hre.RepositoryName,
				Commit:         hre.ExtractData.Commit,
				SignKey:        hre.SignKey,
				HookEventUUID:  hre.UUID,
			}
		}

		r, err := s.Client.RetrieveHookEventUser(ctx, req)
		if err != nil {
			return err
		}
		if r.Initiator == nil || (r.Initiator.UserID == "" && r.Initiator.VCSUsername == "") {
			hre.Status = sdk.HookEventStatusSkipped
			hre.LastError = fmt.Sprintf("User with key %s not found", hre.SignKey)
		}

		hre.Initiator = r.Initiator

		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}

		if hre.IsTerminated() {
			return s.Dao.RemoveRepositoryEventFromInProgressList(ctx, hre.UUID)
		}
	}

	event := make(map[string]interface{})
	if len(hre.Body) > 0 {
		if err := json.Unmarshal(hre.Body, &event); err != nil {
			return sdk.WithStack(err)
		}
	}

	for i := range hre.WorkflowHooks {
		wh := &hre.WorkflowHooks[i]
		initiator := hre.Initiator
		if wh.Initiator != nil {
			initiator = wh.Initiator
		}
		if !wh.Data.InsecureSkipSignatureVerify && (initiator == nil || (initiator.UserID == "" && initiator.VCSUsername == "")) {
			wh.Status = sdk.HookEventWorkflowStatusSkipped
			wh.Error = "unknown user"
			if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
				return err
			}
			continue
		}
		if wh.Status == sdk.HookEventWorkflowStatusScheduled {
			// Check path filter
			canTriggerWithChangeSet := false
			if len(wh.PathFilters) > 0 {
			pathLoop:
				for _, hookPathFilter := range wh.PathFilters {
					g := glob.New(hookPathFilter)
					for _, file := range wh.UpdatedFiles {
						result, err := g.MatchString(file)
						if err != nil {
							log.Error(ctx, "unable to check file %s with pattern %s", hookPathFilter)
							continue
						}
						if result == nil {
							continue
						}
						canTriggerWithChangeSet = true
						break pathLoop
					}
				}
			} else {
				canTriggerWithChangeSet = true
			}

			canTriggerWithCommitMessage := false
			if wh.Data.CommitFilter != "" {
				g := glob.New(wh.Data.CommitFilter)
				r, err := g.MatchString(hre.ExtractData.CommitMessage)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					hre.LastError = err.Error()
				}
				if r != nil {
					canTriggerWithCommitMessage = true
				}
			} else {
				canTriggerWithCommitMessage = true
			}

			if !canTriggerWithChangeSet || !canTriggerWithCommitMessage {
				wh.Status = sdk.HookEventWorkflowStatusSkipped
			} else {
				// Query params to select the right workflow version to run
				mods := make([]cdsclient.RequestModifier, 0, 2)
				mods = append(mods, cdsclient.WithQueryParameter("ref", wh.Ref), cdsclient.WithQueryParameter("commit", wh.Commit))

				log.Debug(ctx, "triggerWorkflows - initiator: %+v", initiator)

				runRequest := sdk.V2WorkflowRunHookRequest{
					HookEventID:        hre.UUID,
					Ref:                hre.ExtractData.Ref,
					Sha:                wh.TargetCommit,
					CommitMessage:      hre.ExtractData.CommitMessage,
					CommitAuthor:       hre.ExtractData.CommitAuthor,
					CommitAuthorEmail:  hre.ExtractData.CommitAuthorEmail,
					Payload:            event,
					EventName:          hre.EventName,
					HookType:           wh.Type,
					SemverCurrent:      wh.SemverCurrent,
					SemverNext:         wh.SemverNext,
					ChangeSets:         wh.UpdatedFiles,
					DeprecatedAdminMFA: hre.ExtractData.DeprecatedAdminMFA,
					PullrequestID:      hre.ExtractData.PullRequestID,
					PullrequestToRef:   hre.ExtractData.PullRequestRefTo,
					Initiator:          initiator,
				}
				if initiator != nil {
					runRequest.DeprecatedUserID = initiator.UserID
				}
				if wh.Data.TargetBranch != "" {
					runRequest.Ref = sdk.GitRefBranchPrefix + wh.Data.TargetBranch
				} else if wh.Data.TargetTag != "" {
					runRequest.Ref = sdk.GitRefTagPrefix + wh.Data.TargetTag
				}

				// Override repository ref to clone in the workflow
				switch wh.Type {
				case sdk.WorkflowHookTypeWorkflow:
					runRequest.EntityUpdated = wh.WorkflowName
					runRequest.EventName = "workflow-update"
				case sdk.WorkflowHookTypeWorkerModel:
					runRequest.EntityUpdated = wh.ModelFullName
					runRequest.EventName = "model-update"
				case sdk.WorkflowHookTypeScheduler:
					runRequest.Cron = hre.ExtractData.Scheduler.Cron
					runRequest.CronTimezone = hre.ExtractData.Scheduler.Timezone
					runRequest.Sha = wh.TargetCommit
				case sdk.WorkflowHookTypeWorkflowRun:
					runRequest.WorkflowRun = hre.ExtractData.WorkflowRun.Workflow
					runRequest.WorkflowRunID = hre.ExtractData.WorkflowRun.WorkflowRunID
				case sdk.WorkflowHookTypeManual:
					// Manual run can override repo and vcs
					runRequest.TargetRepository = wh.Data.RepositoryName
				}

				wr, err := s.Client.WorkflowV2RunFromHook(ctx, wh.ProjectKey, wh.VCSIdentifier, wh.RepositoryIdentifier, wh.WorkflowName,
					runRequest, mods...)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					wh.Status = sdk.HookEventWorkflowStatusError
					wh.Error = fmt.Sprintf("unable to run workflow %s: %v", wh.WorkflowName, err)
				} else {
					wh.Status = sdk.HookEventWorkflowStatusDone
					wh.RunID = wr.ID
					wh.RunNumber = wr.RunNumber
				}
			}

			if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
				return err
			}
		}
	}

	allFailed := len(hre.WorkflowHooks) > 0
	var lastError string
	for i := range hre.WorkflowHooks {
		wh := &hre.WorkflowHooks[i]
		if !wh.IsTerminated() {
			return nil
		}

		if wh.Status != sdk.HookEventWorkflowStatusError {
			allFailed = false
		} else {
			lastError = wh.Error
		}
	}

	if allFailed {
		hre.Status = sdk.HookEventStatusError
		hre.LastError = "All workflow hooks failed: " + lastError
	} else {
		hre.Status = sdk.HookEventStatusDone
	}

	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	if hre.IsTerminated() {
		if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, hre.UUID); err != nil {
			return err
		}
	}

	return nil
}
