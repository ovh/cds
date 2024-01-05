package hooks

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (s *Service) triggerGetSigningKey(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerGetSigningKey")
	defer next()

	log.Info(ctx, "triggering get signing key for event [%s] %s", hre.EventName, hre.GetFullName())
	// Check if we know the user that trigger the event
	if hre.UserID == "" {
		// If it's a push event, check signingKey
		if hre.SignKey == "" {
			// If operation not started
			if hre.SigningKeyOperation == "" {
				ope, err := s.Client.RetrieveHookEventSigningKey(ctx, sdk.HookRetrieveSignKeyRequest{
					HookEventUUID:  hre.UUID,
					ProjectKey:     hre.WorkflowHooks[0].ProjectKey,
					VCSServerName:  hre.VCSServerName,
					VCSServerType:  hre.VCSServerType,
					RepositoryName: hre.RepositoryName,
					Commit:         hre.ExtractData.Commit,
					Branch:         hre.ExtractData.Branch,
				})
				if err != nil {
					return err
				}
				hre.SigningKeyOperation = ope.UUID
				if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
					return err
				}
				// Return and wait callback
				return nil
			} else {
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
					// Operation in error : remove uuid, return error and it will be retried
					if ope.Status == sdk.OperationStatusError {
						hre.SigningKeyOperation = ""
						if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
							return err
						}
						return ope.Error.ToError()
					}
					if ope.Status == sdk.OperationStatusDone {
						hre.SignKey = ope.Setup.Checkout.Result.SignKeyID
						hre.SemverCurrent = ope.Setup.Checkout.Result.Semver.Current
						hre.SemverNext = ope.Setup.Checkout.Result.Semver.Next
						if !ope.Setup.Checkout.Result.CommitVerified {
							hre.Status = sdk.HookEventStatusSkipped
							hre.LastError = fmt.Sprintf("User with key '%s' not found in CDS", hre.SignKey)
							if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
								return err
							}
							return nil
						}
						// Continue process and update hook event  status
					}
				} else {
					return nil
				}
			}
		}
	}

	hre.Status = sdk.HookEventStatusWorkflow
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	return s.triggerWorkflows(ctx, hre)
}
