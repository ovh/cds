package repositories

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (s *Service) processor(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var uuid string
		if err := s.dao.store.DequeueWithContext(ctx, processorKey, 250*time.Millisecond, &uuid); err != nil {
			log.Error(ctx, "repositories > processor > store.DequeueWithContext err: %v", err)
			continue
		}
		if uuid != "" {
			op := s.dao.loadOperation(ctx, uuid)
			ctx = context.WithValue(ctx, cdslog.RequestID, op.RequestID)
			if err := s.do(ctx, *op); err != nil {
				if err == errLockUnavailable {
					s.GoRoutines.Exec(ctx, "operation "+uuid+" retry", func(ctx context.Context) {
						op.NbRetries++
						log.Info(ctx, "repositories > processor > lock unavailable. retry")
						time.Sleep(time.Duration(2*op.NbRetries) * time.Second)
						if err := s.dao.pushOperation(ctx, op); err != nil {
							log.Error(ctx, "repositories > processor > %v", err)
						}
					})
				} else {
					log.Error(ctx, "repositories > processor > %v", err)
				}
			}
		}
	}
}

func (s *Service) do(ctx context.Context, op sdk.Operation) error {
	ctx = context.WithValue(ctx, cdslog.Operation, op.UUID)
	ctx = context.WithValue(ctx, cdslog.VCSServer, op.VCSServer)
	ctx = context.WithValue(ctx, cdslog.Repository, op.RepoFullName)

	log.Debug(ctx, "processing > %v", op.UUID)

	r := s.Repo(op)
	if s.dao.lock(ctx, r.ID()) == errLockUnavailable {
		return errLockUnavailable
	}
	defer func() {
		s.dao.unlock(ctx, r.ID())
		ttl := 3600 * 24 * s.Cfg.RepositoriesRetention
		ttlTime := time.Now().Add(time.Duration(ttl) * time.Second)
		log.Info(ctx, "%s protected until %s", r.ID(), ttlTime.String())
		s.dao.store.SetWithTTL(ctx, cache.Key(lastAccessKey, r.ID()), ttlTime, ttl)
	}()

	switch {
	// Load workflow as code file
	case op.Setup.Checkout.Branch != "" || op.Setup.Checkout.Tag != "":
		if err := s.processCheckout(ctx, &op); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, err.Error())
			op.Error = sdk.ToOperationError(sdk.FromGitToHumanError(sdk.ErrUnknownError, err))
			op.Status = sdk.OperationStatusError
		} else {
			op.Error = nil
			op.Status = sdk.OperationStatusDone
			switch {
			case op.LoadFiles.Pattern == "" && op.Setup.Checkout.CheckSignature,
				op.LoadFiles.Pattern == "" && op.Setup.Checkout.GetChangeSet,
				op.LoadFiles.Pattern == "" && op.Setup.Checkout.ProcessSemver:
				op.Error = nil
				op.Status = sdk.OperationStatusDone
				// do nothing
			case op.LoadFiles.Pattern != "":
				if err := s.processLoadFiles(ctx, &op); err != nil {
					ctx := sdk.ContextWithStacktrace(ctx, err)
					log.Error(ctx, err.Error())
					op.Error = sdk.ToOperationError(sdk.FromGitToHumanError(sdk.ErrUnknownError, err))
					op.Status = sdk.OperationStatusError
				} else {
					op.Error = nil
					op.Status = sdk.OperationStatusDone
				}
			default:
				op.Error = sdk.ToOperationError(sdk.NewErrorFrom(sdk.ErrUnknownError, "unrecognized operation"))
				op.Status = sdk.OperationStatusError
			}
		}
	// Push workflow as code file
	case op.Setup.Push.FromBranch != "":
		if err := s.processPush(ctx, &op); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, err.Error())
			op.Error = sdk.ToOperationError(sdk.FromGitToHumanError(sdk.ErrUnknownError, err))
			op.Status = sdk.OperationStatusError
		} else {
			op.Error = nil
			op.Status = sdk.OperationStatusDone
		}
	default:
		op.Error = sdk.ToOperationError(sdk.NewErrorFrom(sdk.ErrUnknownError, "unrecognized setup"))
		op.Status = sdk.OperationStatusError
	}

	log.Debug(ctx, "repositories > operation %s: %+v ", op.UUID, op.Error)
	log.Info(ctx, "repositories > operation %s status: %v ", op.UUID, op.Status)
	if op.Status == sdk.OperationStatusError {
		log.Error(ctx, "operation %s error %s", op.UUID, op.Error.Message)
	}

	return s.dao.saveOperation(ctx, &op)
}
