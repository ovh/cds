package repogithub

import (
	"flag"

	"github.com/ovh/cds/engine/log"
)

var (
	DBDriver   string
	dbUser     string
	dbPassword string
	dbName     string
	dbHost     string
	dbPort     string
	dbSSLMode  string
)

func init() {
	if flag.Lookup("dbDriver") == nil {
		flag.StringVar(&DBDriver, "dbDriver", "", "driver")
		flag.StringVar(&dbUser, "dbUser", "cds", "user")
		flag.StringVar(&dbPassword, "dbPassword", "cds", "password")
		flag.StringVar(&dbName, "dbName", "cds", "database name")
		flag.StringVar(&dbHost, "dbHost", "localhost", "host")
		flag.StringVar(&dbPort, "dbPort", "15432", "port")
		flag.StringVar(&dbSSLMode, "sslMode", "disable", "ssl mode")
		flag.Parse()

		log.SetLevel(log.DebugLevel)
	}
}
