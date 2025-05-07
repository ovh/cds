package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (s *Service) processor(ctx context.Context) error {
	chanOperation := make(chan sdk.Operation, s.Cfg.MaxWorkers)
	for w := 1; w <= s.Cfg.MaxWorkers; w++ {
		s.GoRoutines.RunWithRestart(ctx, fmt.Sprintf("operation-worker-%d", w), func(ctx context.Context) {
			for op := range chanOperation {
				ctx := context.WithValue(ctx, cdslog.Operation, op.UUID)
				ctx = context.WithValue(ctx, cdslog.VCSServer, op.VCSServer)
				ctx = context.WithValue(ctx, cdslog.Repository, op.RepoFullName)
				log.Debug(ctx, "work on %s on branch %s", op.URL, op.Setup.Checkout.Branch)
				if err := s.do(ctx, op); err != nil {
					log.Error(ctx, "repositories > processor > %v", err)
				}
			}
		})
	}

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
			r := s.Repo(*op)
			_, has := s.localCache.Get(r.ID())
			if has {
				s.GoRoutines.Exec(ctx, "operation "+uuid+" retry", func(ctx context.Context) {
					op.NbRetries++
					log.Info(ctx, "repositories > processor > lock unavailable on repository %s. Retry", op.RepoFullName)
					time.Sleep(time.Duration(2*op.NbRetries) * time.Second)
					if err := s.dao.pushOperation(op); err != nil {
						log.Error(ctx, "repositories > processor > %v", err)
					}
				})
				continue
			}
			s.localCache.Set(r.ID(), true, 10*time.Minute)
			chanOperation <- *op
		}
	}
}

func (s *Service) do(ctx context.Context, op sdk.Operation) error {
	ctx = context.WithValue(ctx, cdslog.Operation, op.UUID)
	ctx = context.WithValue(ctx, cdslog.VCSServer, op.VCSServer)
	ctx = context.WithValue(ctx, cdslog.Repository, op.RepoFullName)

	log.Debug(ctx, "processing > %v", op.UUID)

	r := s.Repo(op)
	defer func() {
		s.localCache.Delete(r.ID())
		ttl := 3600 * 24 * s.Cfg.RepositoriesRetention
		ttlTime := time.Now().Add(time.Duration(ttl) * time.Second)
		log.Info(ctx, "%s protected until %s", r.ID(), ttlTime.String())
		s.dao.store.SetWithTTL(cache.Key(lastAccessKey, r.ID()), ttlTime, ttl)
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

	return s.dao.saveOperation(&op)
}
