package hooks

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

func (s *Service) triggerGetGitInfo(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerGetGitInfo")
	defer next()

	log.Info(ctx, "triggering get git info for event [%s] %s", hre.EventName, hre.GetFullName())

	repositoryOperationCache := make(map[string]sdk.Operation)

	for i := range hre.WorkflowHooks {
		wh := &hre.WorkflowHooks[i]

		repoKeyUniqueKey := fmt.Sprintf("%s-%s-%s", wh.Data.VCSServer, wh.Data.RepositoryName, wh.TargetCommit)

		if wh.OperationUUID == "" {
			existingOpe, has := repositoryOperationCache[repoKeyUniqueKey]
			if !has {
				var ref string
				withChangeSet := true
				if wh.Data.TargetBranch != "" {
					ref = sdk.GitRefBranchPrefix + wh.Data.TargetBranch
				} else if wh.Data.TargetTag != "" {
					ref = sdk.GitRefTagPrefix + wh.Data.TargetTag
					withChangeSet = false
				}

				ope, err := s.Client.RetrieveHookEventSigningKey(ctx, sdk.HookRetrieveSignKeyRequest{
					HookEventUUID:         hre.UUID,
					HookEventKey:          cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hre.VCSServerName, hre.RepositoryName), hre.UUID),
					ProjectKey:            hre.WorkflowHooks[0].ProjectKey,
					VCSServerName:         wh.Data.VCSServer,
					RepositoryName:        wh.Data.RepositoryName,
					Commit:                wh.TargetCommit,
					Ref:                   ref,
					GetSigninKey:          false,
					GetChangesets:         withChangeSet,
					GetSemver:             true,
					GetCommitMessage:      true,
					ChangesetsCommitSince: hre.ExtractData.CommitFrom,
					ChangesetsBranchTo:    hre.ExtractData.PullRequestRefTo,
				})
				if err != nil {
					return err
				}
				wh.OperationStatus = ope.Status
				wh.OperationUUID = ope.UUID
				repositoryOperationCache[repoKeyUniqueKey] = ope
				log.Info(ctx, "operation %s created to retrieve git info", ope.UUID)
			} else {
				wh.OperationStatus = existingOpe.Status
				wh.OperationUUID = existingOpe.UUID
			}

			if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
				return err
			}

		} else if wh.OperationStatus != sdk.OperationStatusDone && wh.OperationStatus != sdk.OperationStatusError {
			if time.Now().UnixMilli()-wh.LastCheck > RetryDelayMilli {
				// Call CDS api to get operation
				ope, err := s.Client.RetrieveHookEventSigningKeyOperation(ctx, wh.OperationUUID)
				if err != nil {
					return err
				}
				log.Info(ctx, "check operation %s status: %s", ope.UUID, ope.Status)
				wh.LastCheck = time.Now().UnixMilli()
				wh.OperationRetry++
				// Operation in progress : do nothing
				if ope.Status == sdk.OperationStatusPending || ope.Status == sdk.OperationStatusProcessing {
					if wh.OperationRetry >= OperationMaxRretry {
						wh.OperationStatus = sdk.OperationStatusError
						wh.Error = "unable to retrieve git info: exceeded max retry delay"
						wh.Status = sdk.HookEventWorkflowStatusError
					}
					if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
						return err
					}
					continue
				}
				if err := s.manageRepositoryOperationCallback(ctx, ope, hre); err != nil {
					return err
				}
				if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
					return err
				}
			}

		}
	}

	allFailed := true
	for i := range hre.WorkflowHooks {
		wh := hre.WorkflowHooks[i]
		if wh.OperationStatus != sdk.OperationStatusError {
			allFailed = false
		}
		// If we don't have all operation callbacks, return and wait for them
		if wh.OperationStatus != sdk.OperationStatusDone && wh.OperationStatus != sdk.OperationStatusError {
			return nil
		}
	}

	if allFailed {
		hre.Status = sdk.HookEventStatusError
		hre.LastError = hre.WorkflowHooks[0].Error
	} else {
		hre.Status = sdk.HookEventStatusWorkflow
	}

	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	if hre.IsTerminated() {
		return s.Dao.RemoveRepositoryEventFromInProgressList(ctx, hre.UUID)
	}
	return s.triggerWorkflows(ctx, hre)
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
	if hre.ExtractData.CommitAuthor == "" {
		hre.ExtractData.CommitAuthor = ope.Setup.Checkout.Result.Author
	}
	if hre.ExtractData.CommitAuthorEmail == "" {
		hre.ExtractData.CommitAuthorEmail = ope.Setup.Checkout.Result.AuthorEmail
	}

	// Update repository hook status
	for i := range hre.WorkflowHooks {
		wh := &hre.WorkflowHooks[i]

		// Check if callback is for the current workflow hook
		if wh.OperationUUID != ope.UUID {
			continue
		}

		// Update workflow hook status
		if ope.Status == sdk.OperationStatusError {
			wh.Status = sdk.HookEventWorkflowStatusError
			wh.Error = ope.Error.ToError().Error()
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

	allHooksSkipped := true
	for _, wh := range hre.WorkflowHooks {
		if wh.Status != sdk.HookEventWorkflowStatusSkipped {
			allHooksSkipped = false
		}
	}
	if allHooksSkipped {
		hre.Status = sdk.HookEventStatusSkipped
		hre.LastError = ope.Setup.Checkout.Result.Msg + fmt.Sprintf("(Operation ID: %s)", ope.UUID)
	}
	return nil
}
