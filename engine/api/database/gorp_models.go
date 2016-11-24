package database

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

//DBMap returns a propor intialized gorp.DBMap pointer
func DBMap(db *sql.DB) *gorp.DbMap {
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}

	dbmap.AddTableWithName(TemplateExtension{}, "template").SetKeys(true, "id")
	dbmap.AddTableWithName(WorkerModel{}, "worker_model").SetKeys(true, "id")

	return dbmap
}

//TemplateExtension is a gorp wrapper around sdk.TemplateExtension
type TemplateExtension sdk.TemplateExtension

//WorkerModel is a gorp wrapper around sdk.Model
type WorkerModel sdk.Model
