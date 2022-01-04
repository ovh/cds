package pipeline

import (
	"context"
	"sync"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// ImportOptions are options to import pipeline
type ImportOptions struct {
	Force          bool
	PipelineName   string
	FromRepository string
}

// ParseAndImport parse an exportentities.pipeline and insert or update the pipeline in database
func ParseAndImport(ctx context.Context, db gorp.SqlExecutor, cache cache.Store, proj sdk.Project, epip exportentities.Pipeliner, u sdk.Identifiable, opts ImportOptions) (*sdk.Pipeline, []sdk.Message, error) {
	//Transform payload to a sdk.Pipeline
	pip, err := epip.Pipeline()
	if err != nil {
		return nil, nil, err
	}

	pip.FromRepository = opts.FromRepository

	if opts.PipelineName != "" && pip.Name != opts.PipelineName {
		return nil, nil, sdk.WithStack(sdk.ErrPipelineNameImport)
	}

	// Check if pipeline exists
	exist, err := ExistPipeline(db, proj.ID, pip.Name)
	if err != nil {
		return pip, nil, sdk.WrapError(err, "unable to check if pipeline %v exists", pip.Name)
	}

	done := new(sync.WaitGroup)
	done.Add(1)
	msgChan := make(chan sdk.Message)
	msgList := []sdk.Message{}
	go func(array *[]sdk.Message) {
		defer done.Done()
		for m := range msgChan {
			*array = append(*array, m)
		}
	}(&msgList)

	previousPip := pip
	if exist {
		prevPip, err := LoadPipeline(ctx, db, proj.Key, pip.Name, true)
		if err != nil {
			return pip, nil, sdk.WrapError(err, "cannot load previous pipeline")
		}
		previousPip = prevPip
	}

	var globalError error
	if exist && !opts.Force {
		return pip, nil, sdk.ErrPipelineAlreadyExists
	} else if exist {
		globalError = ImportUpdate(ctx, db, proj, pip, msgChan, opts)
	} else {
		globalError = Import(ctx, db, cache, proj, pip, msgChan, u)
	}

	close(msgChan)
	done.Wait()

	if globalError == nil {
		if err := CreateAudit(db, previousPip, AuditUpdatePipeline, u); err != nil {
			log.Error(ctx, "%v", sdk.WrapError(err, "cannot create audit"))
		}
	}

	return pip, msgList, globalError
}
