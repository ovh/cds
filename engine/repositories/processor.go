package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processor(ctx context.Context) error {
	for {
		var uuid string
		if err := s.dao.store.DequeueWithContext(ctx, processorKey, 250*time.Millisecond, &uuid); err != nil {
			log.Error(ctx, "repositories > processor > store.DequeueWithContext err: %v", err)
			continue
		}
		if uuid != "" {
			op := s.dao.loadOperation(ctx, uuid)
			ctx = context.WithValue(ctx, log.ContextLoggingRequestIDKey, op.RequestID)
			if err := s.do(ctx, *op); err != nil {
				if err == errLockUnavailable {
					s.GoRoutines.Exec(ctx, "operation "+uuid+" retry", func(ctx context.Context) {
						op.NbRetries++
						log.Info(ctx, "repositories > processor > lock unavailable. retry")
						time.Sleep(time.Duration(2*op.NbRetries) * time.Second)
						if err := s.dao.pushOperation(op); err != nil {
							log.Error(ctx, "repositories > processor > %v", err)
						}
					})
				} else {
					log.Error(ctx, "repositories > processor > %v", err)
				}
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (s *Service) do(ctx context.Context, op sdk.Operation) error {
	log.Debug("repositories > processing > %v", op.UUID)

	r := s.Repo(op)
	if s.dao.lock(r.ID()) == errLockUnavailable {
		return errLockUnavailable
	}
	defer s.dao.unlock(ctx, r.ID(), 24*time.Hour*time.Duration(s.Cfg.RepositoriesRetention)) // nolint

	switch {
	// Load workflow as code file
	case op.Setup.Checkout.Branch != "" || op.Setup.Checkout.Tag != "":
		if err := s.processCheckout(ctx, &op); err != nil {
			isErrWithStack := sdk.IsErrorWithStack(err)
			fields := log.Fields{}
			if isErrWithStack {
				fields["stack_trace"] = fmt.Sprintf("%+v", err)
			}
			log.ErrorWithFields(ctx, fields, "%s", err)

			op.Error = sdk.ToOperationError(err)
			op.Status = sdk.OperationStatusError
		} else {
			op.Error = nil
			op.Status = sdk.OperationStatusDone
			switch {
			case op.LoadFiles.Pattern != "":
				if err := s.processLoadFiles(ctx, &op); err != nil {
					isErrWithStack := sdk.IsErrorWithStack(err)
					fields := log.Fields{}
					if isErrWithStack {
						fields["stack_trace"] = fmt.Sprintf("%+v", err)
					}
					log.ErrorWithFields(ctx, fields, "%s", err)

					op.Error = sdk.ToOperationError(err)
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
			isErrWithStack := sdk.IsErrorWithStack(err)
			fields := log.Fields{}
			if isErrWithStack {
				fields["stack_trace"] = fmt.Sprintf("%+v", err)
			}
			log.ErrorWithFields(ctx, fields, "%s", err)

			op.Error = sdk.ToOperationError(err)
			op.Status = sdk.OperationStatusError
		} else {
			op.Error = nil
			op.Status = sdk.OperationStatusDone
		}
	default:
		op.Error = sdk.ToOperationError(sdk.NewErrorFrom(sdk.ErrUnknownError, "unrecognized setup"))
		op.Status = sdk.OperationStatusError
	}

	log.Debug("repositories > operation %s: %+v ", op.UUID, op.Error)
	log.Info(ctx, "repositories > operation %s status: %v ", op.UUID, op.Status)
	if op.Status == sdk.OperationStatusError {
		log.Error(ctx, "repositories> operation %s error %s", op.UUID, op.Error.Message)
	}

	return s.dao.saveOperation(&op)
}
