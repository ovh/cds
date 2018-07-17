package pipeline

import (
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

type pipeliner interface {
	Pipeline() (*sdk.Pipeline, error)
}

// ImportOptions are options to import pipeline
type ImportOptions struct {
	Force        bool
	PipelineName string
}

// ParseAndImport parse an exportentities.pipeline and insert or update the pipeline in database
func ParseAndImport(db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, epip pipeliner, u *sdk.User, opts ImportOptions) (*sdk.Pipeline, []sdk.Message, error) {
	//Transform payload to a sdk.Pipeline
	pip, errP := epip.Pipeline()
	if errP != nil {
		return pip, nil, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, errP), "ParseAndImport> Unable to parse pipeline")
	}

	if opts.PipelineName != "" && pip.Name != opts.PipelineName {
		return nil, nil, sdk.ErrPipelineNameImport
	}

	// Check if pipeline exists
	exist, errE := ExistPipeline(db, proj.ID, pip.Name)
	if errE != nil {
		return pip, nil, sdk.WrapError(errE, "ParseAndImport> Unable to check if pipeline %v exists", pip.Name)
	}

	// Load group in permission
	for i := range pip.GroupPermission {
		eg := &pip.GroupPermission[i]
		g, errg := group.LoadGroup(db, eg.Group.Name)
		if errg != nil {
			return pip, nil, sdk.WrapError(errg, "ParseAndImport> Error loading groups for permission")
		}
		eg.Group = *g
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

	var globalError error

	if exist && !opts.Force {
		return pip, nil, sdk.ErrPipelineAlreadyExists
	} else if exist {
		globalError = ImportUpdate(db, proj, pip, msgChan, u)
	} else {
		globalError = Import(db, cache, proj, pip, msgChan, u)
	}

	close(msgChan)
	done.Wait()

	return pip, msgList, globalError
}
