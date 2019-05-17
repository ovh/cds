package repositories

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processor(ctx context.Context) error {
	for {
		var uuid string
		s.dao.store.DequeueWithContext(ctx, processorKey, &uuid)
		if uuid != "" {
			op := s.dao.loadOperation(uuid)
			if err := s.do(*op); err != nil {
				if err == errLockUnavailable {
					log.Info("repositories > processor > lock unavailabe. Retry")
					s.dao.pushOperation(op)
				} else {
					log.Error("repositories > processor > %v", err)
				}
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (s *Service) do(op sdk.Operation) error {
	log.Debug("repositories > processing > %v", op.UUID)

	r := s.Repo(op)
	if s.dao.lock(r.ID()) == errLockUnavailable {
		return errLockUnavailable
	}
	defer s.dao.unlock(r.ID(), 24*time.Hour*time.Duration(s.Cfg.RepositoriesRentention))

	switch {
	// Load workflow as code file
	case op.Setup.Checkout.Branch != "":
		if err := s.processCheckout(&op); err != nil {
			op.Error = sdk.Cause(err).Error()
			op.Status = sdk.OperationStatusError
		} else {
			op.Error = ""
			op.Status = sdk.OperationStatusDone
			switch {
			case op.LoadFiles.Pattern != "":
				if err := s.processLoadFiles(&op); err != nil {
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
		if err := s.processPush(&op); err != nil {
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
