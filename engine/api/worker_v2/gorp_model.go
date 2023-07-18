package worker_v2

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type dbWorker struct {
	sdk.V2Worker
	gorpmapper.SignedEntity
}

func (r dbWorker) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{r.ID, r.JobRunID, r.HatcheryID, r.HatcheryName, r.ConsumerID, r.PrivateKey}
	return gorpmapper.CanonicalForms{
		"{{.ID}}{{.JobRunID}}{{.HatcheryID}}{{.HatcheryName}}{{.ConsumerID}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbWorker{}, "v2_worker", false, "id"))
}
