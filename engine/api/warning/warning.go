package warning

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type warn interface {
	name() string
	events() []string
	compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error
}

var warnings = []warn{
	unusedProjectVariableWarning{},
	missingProjectVariableEnv{},
	missingProjectVariableWorkflow{},
	missingProjectVariableApplication{},
	missingProjectVariablePipelineParameter{},
	missingProjectVariablePipelineJob{},
	missingProjectPermissionEnvWarning{},
	missingProjectPermissionWorkflowWarning{},
	unusedProjectKeyWarning{},
	missingProjectKeyApplicationWarning{},
	missingProjectKeyPipelineJobWarning{},
	missingProjectKeyPipelineParameterWarning{},
	unusedProjectVCSWarning{},
	missingProjectVCSWarning{},
}

// Start starts compute warning from events
func Start(ctx context.Context, DBFunc func() *gorp.DbMap, ch <-chan sdk.Event) {
	var computeMap = make(map[string][]warn)
	for _, w := range warnings {
		for _, e := range w.events() {
			if _, ok := computeMap[e]; !ok {
				computeMap[e] = make([]warn, 0, 1)
			}
			computeMap[e] = append(computeMap[e], w)
		}
	}

	db := DBFunc()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Warning.Start: %v", ctx.Err())
			}
			return
		case e := <-ch:
			if warns, ok := computeMap[e.EventType]; ok {
				for _, w := range warns {
					tx, errT := db.Begin()
					if errT != nil {
						log.Warning(ctx, "Warning.Start> Unable to start transaction")
						continue
					}
					if err := w.compute(ctx, tx, e); err != nil {
						log.Warning(ctx, "Warning.Start> Unable to compute warnning %s: %v", w.name(), err)
						_ = tx.Rollback()
					}
					if err := tx.Commit(); err != nil {
						log.Warning(ctx, "Warning.Start> Unable to commit transaction: %v", err)
						_ = tx.Rollback()
					}
				}
			}

		}
	}

}
