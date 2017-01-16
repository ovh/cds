package secret

import (
	"flag"
	"fmt"

	"github.com/ovh/cds/engine/log"
)

func init() {
	if flag.Lookup("dbDriver") == nil {
		fmt.Println("secret")
		flag.String("dbDriver", "", "driver")
		flag.String("dbUser", "cds", "user")
		flag.String("dbPassword", "cds", "password")
		flag.String("dbName", "cds", "database name")
		flag.String("dbHost", "localhost", "host")
		flag.String("dbPort", "15432", "port")
		flag.String("sslMode", "disable", "ssl mode")
		flag.Parse()

		log.SetLevel(log.DebugLevel)
	}
}
