package warning

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Compute warnings from CDS events
func Compute(c context.Context, store cache.Store, DBFunc func() *gorp.DbMap, ch <-chan sdk.Event) {
	db := DBFunc()

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Warning.Compute: %v", c.Err())
			}
			return
		case e := <-ch:
			tx, errT := db.Begin()
			if errT != nil {
				log.Warning("computeWithProjectEvent> Unable to start transaction")
				return
			}
			switch {
			case strings.HasPrefix(e.EventType, "sdk.EventProject"):
				if err := computeWithProjectEvent(tx, e); err != nil {
					log.Warning("warning.Compute: unable to compute project event: %v", err)
					_ = tx.Rollback()
				} else {
					commit(tx)
				}
			case strings.HasPrefix(e.EventType, "sdk.EventApplication"):
				computeWithApplicationEvent(tx, store, e)
				commit(tx)
			case strings.HasPrefix(e.EventType, "sdk.EventEnvironment"):
				computeWithEnvironmentEvent(tx, store, e)
				commit(tx)
			case strings.HasPrefix(e.EventType, "sdk.EventPipeline"):
				computeWithPipelineEvent(tx, store, e)
				commit(tx)
			case strings.HasPrefix(e.EventType, "sdk.EventWorkflow"):
				computeWithWorkflowEvent(tx, store, e)
				commit(tx)
			}
			_ = tx.Rollback()
		}
	}
}

func commit(tx *gorp.Transaction) {
	if err := tx.Commit(); err != nil {
		log.Warning("ComputeWarning.commit: unable to commit transanction: %v", err)
		_ = tx.Rollback()
	}
	return
}

func pipelineSliceMerge(paramDataSlice []pipeline.CountInValueParamData, pipDataSlice []pipeline.CountInPipelineData) []string {
	elementsMap := make(map[string]bool, len(paramDataSlice)+len(pipDataSlice))
	pipelinesName := make([]string, 0, len(paramDataSlice)+len(pipDataSlice))
	for _, v := range paramDataSlice {
		if !elementsMap[v.Name] {
			elementsMap[v.Name] = true
			pipelinesName = append(pipelinesName, v.Name)
		}
	}
	for _, v := range pipDataSlice {
		if !elementsMap[v.PipName] {
			elementsMap[v.PipName] = true
			pipelinesName = append(pipelinesName, v.PipName)
		}
	}
	return pipelinesName
}
