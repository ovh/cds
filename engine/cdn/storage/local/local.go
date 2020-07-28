package local

import (
	"context"
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk/log"
)

type Local struct {
}

var _ storage.StorageUnit = new(Local)

func init() {
	storage.RegisterDriver("local", new(Local))
}

func (s *Local) Name() string {
	return ""
}
func (s *Local) Init(m *gorpmapper.Mapper, db *gorp.DbMap, u storage.Unit, cfg interface{}) error {
	return nil
}
func (s *Local) Run() {
	log.Debug("local.Run")
	// Load all the items we have to process

	// do the stuff
	if err := storage.Run(nil, nil, index.Item{}); err != nil {
		log.Error(context.Background(), "storage error: %v", err)
	}

}
func (s *Local) ItemExists(i index.Item) error                  { return nil }
func (s *Local) NewWriter(i index.Item) (io.WriteCloser, error) { return nil, nil }
func (s *Local) NewReader(i index.Item) (io.ReadCloser, error)  { return nil, nil }
