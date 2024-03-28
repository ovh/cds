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
				if wh.Data.TargetBranch != "" {
					ref = sdk.GitRefBranchPrefix + wh.Data.TargetBranch
				} else {
					ref = sdk.GitRefTagPrefix + wh.Data.TargetTag
				}
				ope, err := s.Client.RetrieveHookEventSigningKey(ctx, sdk.HookRetrieveSignKeyRequest{
					HookEventUUID:  hre.UUID,
					HookEventKey:   cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hre.VCSServerName, hre.RepositoryName), hre.UUID),
					ProjectKey:     hre.WorkflowHooks[0].ProjectKey,
					VCSServerName:  wh.Data.VCSServer,
					RepositoryName: wh.Data.RepositoryName,
					Commit:         wh.TargetCommit,
					Ref:            ref,
					GetSigninKey:   false,
					GetChangesets:  true,
					GetSemver:      true,
				})
				if err != nil {
					return err
				}
				wh.OperationStatus = ope.Status
				wh.OperationUUID = ope.UUID
				repositoryOperationCache[repoKeyUniqueKey] = ope
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
				// Operation in progress : do nothing
				if ope.Status == sdk.OperationStatusPending || ope.Status == sdk.OperationStatusProcessing {
					wh.LastCheck = time.Now().UnixMilli()
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

	for _, wh := range hre.WorkflowHooks {
		if wh.OperationStatus != sdk.OperationStatusDone && wh.OperationStatus != sdk.OperationStatusError {
			return nil
		}
	}

	hre.Status = sdk.HookEventStatusWorkflow
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	return s.triggerWorkflows(ctx, hre)
}
