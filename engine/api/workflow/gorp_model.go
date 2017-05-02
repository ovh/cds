package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

// Workflow is a gorp wrapper around sdk.Workflow
type Workflow sdk.Workflow

func init() {
	gorpmapping.Register(gorpmapping.New(Workflow{}, "workflow", true, "id"))
}

// PostGet is a db hook
func (w *Workflow) PostGet(db gorp.SqlExecutor) error {

	return nil
}
