package worker

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type dbWorker struct {
	gorpmapping.SignedEntity
	sdk.Worker
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbWorker{}, "worker", false, "id"))
}

func (e dbWorker) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.ID, e.Name}
	return gorpmapping.CanonicalForms{
		"{{print .ID}}{{.Name}}",
	}
}
