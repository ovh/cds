package services

import (
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

type Repository struct {
	db    database.DBConnectionFactory
	store cache.Store
}

func (r *Repository) Find(name string) (*sdk.Service, error) {
	return nil, nil
}
