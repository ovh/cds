package workermodel

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(workerModel{}, "worker_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(sdk.ModelPattern{}, "worker_model_pattern", true, "id"))
	gorpmapping.Register(gorpmapping.New(workerModelSecret{}, "worker_model_secret", false, "id"))
	gorpmapping.Register(gorpmapping.New(workerModelCapability{}, "worker_capability", false, "worker_model_id", "type", "name"))
}

type workerModel struct {
	gorpmapper.SignedEntity
	sdk.Model
}

type workerModelSecret struct {
	gorpmapper.SignedEntity
	sdk.WorkerModelSecret
}

func (w workerModel) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{w.ID, w.Name}
	return gorpmapper.CanonicalForms{
		"{{.ID}}{{.Name}}",
	}
}

func (w workerModelSecret) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{w.ID, w.WorkerModelID, w.Name}
	return gorpmapper.CanonicalForms{
		"{{.ID}}{{.WorkerModelID}}{{.Name}}",
	}
}

type workerModelCapability struct {
	WorkerModelID int64  `db:"worker_model_id"`
	Type          string `db:"type"`
	Name          string `db:"name"`
	Argument      string `db:"argument"`
}
