package workermodel

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(WorkerModel{}, "worker_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(sdk.ModelPattern{}, "worker_model_pattern", true, "id"))
	gorpmapping.Register(gorpmapping.New(workerModelCapability{}, "worker_capability", false, "worker_model_id", "type", "name"))
}

// WorkerModel is a gorp wrapper around sdk.Model.
type WorkerModel sdk.Model

type workerModelCapability struct {
	WorkerModelID int64  `db:"worker_model_id"`
	Type          string `db:"type"`
	Name          string `db:"name"`
	Argument      string `db:"argument"`
}
