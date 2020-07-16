package workermodel

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

func init() {
	gorpmapping.Register(gorpmapping.New(workerModel{}, "worker_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(sdk.ModelPattern{}, "worker_model_pattern", true, "id"))
	gorpmapping.Register(gorpmapping.New(workerModelSecret{}, "worker_model_secret", false, "id"))
	gorpmapping.Register(gorpmapping.New(workerModelCapability{}, "worker_capability", false, "worker_model_id", "type", "name"))
}

type workerModel struct {
	gorpmapping.SignedEntity
	sdk.Model
}

type workerModelSecret struct {
	gorpmapping.SignedEntity
	sdk.WorkerModelSecret
}

func (w workerModel) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{w.ID, w.Name}
	return gorpmapping.CanonicalForms{
		"{{.ID}}{{.Name}}",
	}
}

func (w workerModelSecret) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{w.ID, w.WorkerModelID, w.Name}
	return gorpmapping.CanonicalForms{
		"{{.ID}}{{.WorkerModelID}}{{.Name}}",
	}
}

type workerModelCapability struct {
	WorkerModelID int64  `db:"worker_model_id"`
	Type          string `db:"type"`
	Name          string `db:"name"`
	Argument      string `db:"argument"`
}
