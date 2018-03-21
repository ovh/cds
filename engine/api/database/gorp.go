package database

import (
	"database/sql"
	"errors"
	"os"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk/log"
)

type gorpLogger struct {
}

func (g gorpLogger) Printf(format string, v ...interface{}) {
	log.Debug(format, v...)
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

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}, TypeConverter: new(TypeConverter)}

	if os.Getenv("gorp_trace") == "true" {
		dbmap.TraceOn("[GORP]     Query>", gorpLogger{})
	}

	for _, m := range gorpmapping.Mapping {
		dbmap.AddTableWithName(m.Target, m.Name).SetKeys(m.AutoIncrement, m.GetKeys()...)
	}

	lastDB = db
	lastDBMap = dbmap

	return dbmap
}

// TypeConverter is a converter type to assign to gorp
type TypeConverter struct{}

// ToDb is called by gorp to serialize custom types to database values
func (c *TypeConverter) ToDb(val interface{}) (interface{}, error) {
	switch t := val.(type) {
	case timestamp.Timestamp:
		return ptypes.Timestamp(&t)
	case *timestamp.Timestamp:
		if t == nil {
			return nil, nil
		}
		return ptypes.Timestamp(t)
	}
	return val, nil
}

// FromDb is called by gorp to deserialize database values to custom types
func (c *TypeConverter) FromDb(target interface{}) (gorp.CustomScanner, bool) {
	switch t := target.(type) {
	case *time.Time:
		binder := func(holder, target interface{}) error {
			s := holder.(*time.Time)
			if s == nil {
				return nil
			}
			*t = *s
			return nil
		}
		return gorp.CustomScanner{Holder: new(time.Time), Target: new(time.Time), Binder: binder}, true
	case *timestamp.Timestamp:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*time.Time)
			if !ok {
				return errors.New("FromDb: Unable to convert time.Time to *timestamp.Timestamp")
			}
			if s == nil {
				return nil
			}
			date, err := ptypes.TimestampProto(*s)
			if err != nil {
				return err
			}
			*t = *date
			return nil
		}
		return gorp.CustomScanner{Holder: new(time.Time), Target: new(timestamp.Timestamp), Binder: binder}, true
	case **timestamp.Timestamp:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*time.Time)
			if !ok {
				return errors.New("FromDb: Unable to convert time.Time to **timestamp.Timestamp")
			}
			if s == nil {
				return nil
			}
			date, err := ptypes.TimestampProto(*s)
			if err != nil {
				return err
			}
			*t = date
			return nil
		}
		return gorp.CustomScanner{Holder: new(time.Time), Target: new(timestamp.Timestamp), Binder: binder}, true
	}
	return gorp.CustomScanner{}, false
}
