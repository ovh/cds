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

func (e dbWorker) GetWorker() sdk.Worker {
	return sdk.Worker{
		ID:         e.ID,
		PrivateKey: e.PrivateKey,
		Status:     e.Status,
		JobRunID:   e.JobRunID,
		Arch:       e.Arch,
		ConsumerID: e.ConsumerID,
		HatcheryID: e.HatcheryID,
		LastBeat:   e.LastBeat,
		ModelID:    e.ModelID,
		Name:       e.Name,
		OS:         e.OS,
		Uptodate:   e.Uptodate,
		Version:    e.Version,
	}
}
