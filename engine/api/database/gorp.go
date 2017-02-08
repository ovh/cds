package database

import (
	"database/sql"
	"log"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/spf13/viper"
)

type gorpLogger struct {
}

func (g gorpLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

var (
	lastDB    *sql.DB
	lastDBMap *gorp.DbMap
)

//DBMap returns a propor intialized gorp.DBMap pointer
func DBMap(db *sql.DB) *gorp.DbMap {
	if db == lastDB && lastDBMap != nil && db == lastDBMap.Db {
		return lastDBMap
	}

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}

	if viper.GetBool("gorp_trace") {
		dbmap.TraceOn("[GORP]     Query>", gorpLogger{})
	}

	for _, m := range gorpmapping.Mapping {
		dbmap.AddTableWithName(m.Target, m.Name).SetKeys(m.AutoIncrement, m.Keys...)
	}

	lastDB = db
	lastDBMap = dbmap

	return dbmap
}
