package worker

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type dbWorker struct {
	gorpmapper.SignedEntity
	sdk.Worker
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbWorker{}, "worker", false, "id"))
}

func (e dbWorker) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{e.ID, e.Name}
	return gorpmapper.CanonicalForms{
		"{{printf .ID}}{{.Name}}",
		"{{print .ID}}{{.Name}}",
	}
}
