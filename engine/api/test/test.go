package test

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//DBDriver is exported for testing purpose
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
		flag.String("dbDriver", "", "driver")
		flag.String("dbUser", "cds", "user")
		flag.String("dbPassword", "cds", "password")
		flag.String("dbName", "cds", "database name")
		flag.String("dbHost", "localhost", "host")
		flag.String("dbPort", "15432", "port")
		flag.String("sslMode", "disable", "ssl mode")

		log.Initialize(&log.Conf{Level: "debug"})
		flag.Parse()
	}
}

type bootstrapf func(bootstrap.DefaultValues, func() *gorp.DbMap) error

// SetupPG setup PG DB for test
func SetupPG(t *testing.T, bootstrapFunc ...bootstrapf) *gorp.DbMap {
	DBDriver = flag.Lookup("dbDriver").Value.String()
	dbUser = flag.Lookup("dbUser").Value.String()
	dbPassword = flag.Lookup("dbPassword").Value.String()
	dbName = flag.Lookup("dbName").Value.String()
	dbHost = flag.Lookup("dbHost").Value.String()
	dbPort = flag.Lookup("dbPort").Value.String()
	dbSSLMode = flag.Lookup("sslMode").Value.String()

	log.SetLogger(t)

	cache.Initialize("local", "", "", 30)

	if DBDriver == "" {
		t.Skip("This should be run with a database")
		return nil
	}
	if database.DB() == nil {
		db, err := database.Init(dbUser, dbPassword, dbName, dbHost, dbPort, dbSSLMode, 2000, 100)
		if err != nil {
			t.Fatalf("Cannot open database: %s", err)
			return nil
		}

		if err = db.Ping(); err != nil {
			t.Fatalf("Cannot ping database: %s", err)
			return nil
		}
		database.Set(db)

		db.SetMaxOpenConns(100)
		db.SetMaxIdleConns(20)

		// Gracefully shutdown sql connections
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		signal.Notify(c, syscall.SIGTERM)
		signal.Notify(c, syscall.SIGKILL)
		go func() {
			<-c
			log.Warning("Cleanup SQL connections")
			db.Close()
			os.Exit(0)
		}()
	}

	for _, f := range bootstrapFunc {
		if err := f(bootstrap.DefaultValues{SharedInfraToken: sdk.RandomString(32)}, database.GetDBMap); err != nil {
			return nil
		}
	}

	return database.DBMap(database.DB())
}
