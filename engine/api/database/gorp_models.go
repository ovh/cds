package database

import (
	"database/sql"
	"log"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/viper"
)

type gorpLogger struct {
}

func (g gorpLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

//DBMap returns a propor intialized gorp.DBMap pointer
func DBMap(db *sql.DB) *gorp.DbMap {
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}

	if viper.GetBool("gorp_trace") {
		dbmap.TraceOn("[GORP]     Query>", gorpLogger{})
	}

	dbmap.AddTableWithName(TemplateExtension{}, "template").SetKeys(true, "id")
	dbmap.AddTableWithName(WorkerModel{}, "worker_model").SetKeys(true, "id")
	dbmap.AddTableWithName(PipelineScheduler{}, "pipeline_scheduler").SetKeys(true, "id")

	return dbmap
}

//TemplateExtension is a gorp wrapper around sdk.TemplateExtension
type TemplateExtension sdk.TemplateExtension

//WorkerModel is a gorp wrapper around sdk.Model
type WorkerModel sdk.Model

//PipelineScheduler is a gorp wrapper around sdk.PipelineScheduler
type PipelineScheduler sdk.PipelineScheduler
