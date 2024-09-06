package hooks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (s *Service) triggerGetSigningKey(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerGetSigningKey")
	defer next()

	log.Info(ctx, "triggering get git signing key for event [%s] %s", hre.EventName, hre.GetFullName())

	// If operation not started and not manual hook => run repository operation to get signinkey
	if hre.EventName != sdk.WorkflowHookManual && hre.SigningKeyOperation == "" {
		changesets := false
		semver := false
		signinkey := true
		commitMessage := false

		for _, wh := range hre.WorkflowHooks {
			switch wh.Type {
			case sdk.WorkflowHookTypeWorkflow:
				signinkey = true
			case sdk.WorkflowHookTypeWorkerModel:
				signinkey = true
			default:
				changesets = true
				semver = true
				signinkey = true
				commitMessage = true
			}
		}

		if strings.HasPrefix(hre.ExtractData.Ref, sdk.GitRefTagPrefix) {
			changesets = false
		}
		vcs := hre.VCSServerName
		repo := hre.RepositoryName

		// For scheduler we need to take target repository information
		if hre.EventName == sdk.WorkflowHookTypeScheduler {
			vcs = hre.ExtractData.Scheduler.TargetVCS
			repo = hre.ExtractData.Scheduler.TargetRepo
		}
		if hre.EventName == sdk.WorkflowHookEventRun {
			vcs = hre.ExtractData.WorkflowRun.TargetVCS
			repo = hre.ExtractData.WorkflowRun.TargetRepository
		}

		req := sdk.HookRetrieveSignKeyRequest{
			HookEventUUID:    hre.UUID,
			HookEventKey:     cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hre.VCSServerName, hre.RepositoryName), hre.UUID),
			ProjectKey:       hre.WorkflowHooks[0].ProjectKey,
			VCSServerName:    vcs,
			RepositoryName:   repo,
			Commit:           hre.ExtractData.Commit,
			Ref:              hre.ExtractData.Ref,
			GetSigninKey:     signinkey,
			GetChangesets:    changesets,
			GetSemver:        semver,
			GetCommitMessage: commitMessage,
		}
		if changesets {
			req.ChangesetsCommitSince = hre.ExtractData.CommitFrom
		}
		ope, err := s.Client.RetrieveHookEventSigningKey(ctx, req)
		if err != nil {
			return err
		}
		hre.SigningKeyOperationStatus = ope.Status
		hre.SigningKeyOperation = ope.UUID

		// For repository webhook and scheduler, signin operation == gitinfo operation
		for i := range hre.WorkflowHooks {
			wh := &hre.WorkflowHooks[i]
			if wh.Type == sdk.WorkflowHookTypeRepository || wh.Type == sdk.WorkflowHookTypeScheduler {
				wh.OperationUUID = ope.UUID
				wh.OperationStatus = ope.Status
			}
		}

		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}
		// Return and wait callback
		return nil

	} else if hre.EventName != sdk.WorkflowHookManual && hre.SigningKeyOperation != "" {
		// If operation status has not been updated through signkey callback
		if hre.SigningKeyOperationStatus != sdk.OperationStatusDone && hre.SigningKeyOperationStatus != sdk.OperationStatusError {
			if time.Now().UnixMilli()-hre.LastUpdate > RetryDelayMilli {
				// Call CDS api to get operation
				ope, err := s.Client.RetrieveHookEventSigningKeyOperation(ctx, hre.SigningKeyOperation)
				if err != nil {
					return err
				}
				// Operation in progress : do nothing
				if ope.Status == sdk.OperationStatusPending || ope.Status == sdk.OperationStatusProcessing {
					return nil
				}

				// Update hook repository event with operation
				if err := s.manageRepositoryOperationCallback(ctx, ope, hre); err != nil {
					return err
				}
				if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
					return err
				}
			} else {
				return nil
			}
		}
	}

	// If Operation is skipped ( commit unverified ) || in error : stop hook event
	if hre.IsTerminated() {
		if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, *hre); err != nil {
			return err
		}
	}

	// Continue to next step
	hre.Status = sdk.HookEventStatusGitInfo
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	return s.triggerGetGitInfo(ctx, hre)
}

func (s *Service) manageRepositoryOperationCallback(ctx context.Context, ope sdk.Operation, hre *sdk.HookRepositoryEvent) error {
	log.Info(ctx, "receive operation callback %s", ope.UUID)
	var opeError string
	if ope.Status == sdk.OperationStatusError {
		opeError = ope.Error.ToError().Error()
	}
	// Get computed changesets
	computeChangeSets := make([]string, 0, len(ope.Setup.Checkout.Result.Files))
	for _, v := range ope.Setup.Checkout.Result.Files {
		computeChangeSets = append(computeChangeSets, v.Filename)
	}
	if hre.ExtractData.CommitMessage == "" {
		hre.ExtractData.CommitMessage = ope.Setup.Checkout.Result.CommitMessage
	}

	// Update repository hook status
	allHooksSkipped := true
	for i := range hre.WorkflowHooks {
		wh := &hre.WorkflowHooks[i]

		// Check if callback is for the current workflow hook
		if ope.UUID != hre.SigningKeyOperation && wh.OperationUUID != ope.UUID {
			continue
		}

		// If signin key operation failed, stop all hooks
		if ope.Status == sdk.OperationStatusError && ope.UUID == hre.SigningKeyOperation {
			wh.Status = sdk.HookEventWorkflowStatusSkipped
		}

		// Update workflow hook status
		if ope.Status == sdk.OperationStatusError {
			if ope.UUID == hre.SigningKeyOperation {
				wh.Status = sdk.HookEventWorkflowStatusSkipped
			} else {
				wh.Status = sdk.HookEventWorkflowStatusError
				wh.Error = ope.Error.ToError().Error()
			}
		}

		// If we found an unverified commit, skip all hooks
		if ope.Status == sdk.OperationStatusDone && !ope.Setup.Checkout.Result.CommitVerified && ope.UUID == hre.SigningKeyOperation && !wh.Data.InsecureSkipSignatureVerify {
			wh.Status = sdk.HookEventWorkflowStatusSkipped
			continue
		} else {
			allHooksSkipped = false
		}

		// Add gitinfo for repositorywebhook
		if wh.OperationUUID == ope.UUID {
			wh.OperationStatus = ope.Status
			wh.OperationError = opeError
			wh.SemverCurrent = ope.Setup.Checkout.Result.Semver.Current
			wh.SemverNext = ope.Setup.Checkout.Result.Semver.Next
			wh.Data.TargetBranch = ope.Setup.Checkout.Branch
			wh.TargetCommit = ope.Setup.Checkout.Commit
			// Set changeset on workflow hooks
			if len(hre.ExtractData.Paths) > 0 {
				wh.UpdatedFiles = hre.ExtractData.Paths
			} else {
				wh.UpdatedFiles = computeChangeSets
			}
		}
	}

	// Update hook repository event if needed
	if ope.UUID == hre.SigningKeyOperation {
		hre.SigningKeyOperationStatus = ope.Status
		hre.LastError = opeError
		hre.SignKey = ope.Setup.Checkout.Result.SignKeyID
		if hre.SigningKeyOperationStatus == sdk.OperationStatusError {
			hre.Status = sdk.HookEventStatusError
		}

		if allHooksSkipped {
			hre.Status = sdk.HookEventStatusSkipped
			hre.LastError = ope.Setup.Checkout.Result.Msg + fmt.Sprintf("(Operation ID: %s)", ope.UUID)
		}
	}
	return nil
}
