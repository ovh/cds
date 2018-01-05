package pipeline

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

type pipeliner interface {
	Pipeline() (*sdk.Pipeline, error)
}

func ParseAndImport(db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, epip pipeliner, force bool, u *sdk.User) ([]sdk.Message, error) {
	//Transform payload to a sdk.Pipeline
	pip, errP := epip.Pipeline()
	if errP != nil {
		return nil, sdk.WrapError(errP, "importPipelineHandler> Unable to parse pipeline")
	}

	// Check if pipeline exists
	exist, errE := ExistPipeline(db, proj.ID, pip.Name)
	if errE != nil {
		return nil, sdk.WrapError(errE, "importPipelineHandler> Unable to check if pipeline %v exists", pip.Name)
	}

	// Load group in permission
	for i := range pip.GroupPermission {
		eg := &pip.GroupPermission[i]
		g, errg := group.LoadGroup(db, eg.Group.Name)
		if errg != nil {
			return nil, sdk.WrapError(errg, "importPipelineHandler> Error loading groups for permission")
		}
		eg.Group = *g
	}

	allMsg := []sdk.Message{}
	msgChan := make(chan sdk.Message, 1)
	done := make(chan bool)

	go func() {
		for {
			msg, ok := <-msgChan
			allMsg = append(allMsg, msg)
			if !ok {
				done <- true
				return
			}
		}
	}()

	var globalError error

	if exist && !force {
		return nil, sdk.ErrPipelineAlreadyExists
	} else if exist {
		globalError = ImportUpdate(db, proj, pip, msgChan, u)
	} else {
		globalError = Import(db, cache, proj, pip, msgChan, u)
	}

	close(msgChan)
	<-done

	return allMsg, globalError
}
