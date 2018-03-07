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
	log.Info("repositories > processing > %v", op.UUID)
	log.Debug("repositories > processing > %+v", op)

	r := s.Repo(op)
	if s.dao.lock(r.ID()) == errLockUnavailable {
		return errLockUnavailable
	}
	defer s.dao.unlock(r.ID(), 24*time.Hour*time.Duration(s.Cfg.RepositoriesRentention))

	switch {
	case op.Setup.Checkout.Branch != "":
		if err := s.processCheckout(&op); err != nil {
			op.Error = err.Error()
			op.Status = sdk.OperationStatusError
		} else {
			op.Error = ""
			op.Status = sdk.OperationStatusDone
		}
	default:
		op.Error = "unrecognized setup"
		op.Status = sdk.OperationStatusError
	}

	if op.Error != "" {
		return s.dao.saveOperation(&op)
	}

	switch {
	case op.LoadFiles.Pattern != "":
		if err := s.processLoadFiles(&op); err != nil {
			op.Error = err.Error()
			op.Status = sdk.OperationStatusError
		} else {
			op.Error = ""
			op.Status = sdk.OperationStatusDone
		}
	default:
		op.Error = "unrecognized operation"
		op.Status = sdk.OperationStatusError
	}

	log.Debug("repositories > processing done > %+v", op)

	return s.dao.saveOperation(&op)
}
