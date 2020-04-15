package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processor(ctx context.Context) error {
	for {
		var uuid string
		if err := s.dao.store.DequeueWithContext(ctx, processorKey, &uuid); err != nil {
			log.Error(ctx, "repositories > processor > store.DequeueWithContext err: %v", err)
			continue
		}
		if uuid != "" {
			op := s.dao.loadOperation(ctx, uuid)
			if err := s.do(ctx, *op); err != nil {
				if err == errLockUnavailable {
					log.Info(ctx, "repositories > processor > lock unavailable. Retry")
					s.dao.pushOperation(op)
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
			fields := logrus.Fields{}
			if isErrWithStack {
				fields["stack_trace"] = fmt.Sprintf("%+v", err)
			}
			log.ErrorWithFields(ctx, fields, "%s", err)

			op.Error = sdk.Cause(err).Error()
			op.Status = sdk.OperationStatusError
		} else {
			op.Error = ""
			op.Status = sdk.OperationStatusDone
			switch {
			case op.LoadFiles.Pattern != "":
				if err := s.processLoadFiles(ctx, &op); err != nil {
					isErrWithStack := sdk.IsErrorWithStack(err)
					fields := logrus.Fields{}
					if isErrWithStack {
						fields["stack_trace"] = fmt.Sprintf("%+v", err)
					}
					log.ErrorWithFields(ctx, fields, "%s", err)

					op.Error = sdk.Cause(err).Error()
					op.Status = sdk.OperationStatusError
				} else {
					op.Error = ""
					op.Status = sdk.OperationStatusDone
				}
			default:
				op.Error = "unrecognized operation"
				op.Status = sdk.OperationStatusError
			}
		}
	// Push workflow as code file
	case op.Setup.Push.FromBranch != "":
		if err := s.processPush(ctx, &op); err != nil {
			isErrWithStack := sdk.IsErrorWithStack(err)
			fields := logrus.Fields{}
			if isErrWithStack {
				fields["stack_trace"] = fmt.Sprintf("%+v", err)
			}
			log.ErrorWithFields(ctx, fields, "%s", err)

			op.Error = sdk.Cause(err).Error()
			op.Status = sdk.OperationStatusError
		} else {
			op.Error = ""
			op.Status = sdk.OperationStatusDone
		}
	default:
		op.Error = "unrecognized setup"
		op.Status = sdk.OperationStatusError
	}

	return s.dao.saveOperation(&op)
}
