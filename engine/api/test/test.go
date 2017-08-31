package test

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"os/user"
	"path"
	"syscall"
	"testing"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/secret"
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

type bootstrapf func(sdk.DefaultValues, func() *gorp.DbMap) error

// SetupPG setup PG DB for test
func SetupPG(t *testing.T, bootstrapFunc ...bootstrapf) *gorp.DbMap {
	log.SetLogger(t)

	//Try to load flags from config flags, else load from flags
	var f string
	u, _ := user.Current()
	if u != nil {
		f = path.Join(u.HomeDir, ".cds", "tests.cfg.json")
	}
	if _, err := os.Stat(f); err == nil {
		t.Logf("Tests database configuration read from %s", f)
		btes, err := ioutil.ReadFile(f)
		if err != nil {
			t.Fatalf("Error reading %s: %v", f, err)
		}
		if len(btes) != 0 {
			cfg := map[string]string{}
			if err := json.Unmarshal(btes, &cfg); err == nil {
				DBDriver = cfg["dbDriver"]
				dbUser = cfg["dbUser"]
				dbPassword = cfg["dbPassword"]
				dbName = cfg["dbName"]
				dbHost = cfg["dbHost"]
				dbPort = cfg["dbPort"]
				dbSSLMode = cfg["sslMode"]
			} else {
				t.Errorf("Error when unmarshal config %s", err)
			}
		}
	} else {
		t.Logf("Error reading %s: %v", f, err)
		DBDriver = flag.Lookup("dbDriver").Value.String()
		dbUser = flag.Lookup("dbUser").Value.String()
		dbPassword = flag.Lookup("dbPassword").Value.String()
		dbName = flag.Lookup("dbName").Value.String()
		dbHost = flag.Lookup("dbHost").Value.String()
		dbPort = flag.Lookup("dbPort").Value.String()
		dbSSLMode = flag.Lookup("sslMode").Value.String()
	}

	cache.Initialize("local", "", "", 30)

	secret.Init("3dojuwevn94y7orh5e3t4ejtmbtstest")

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
		if err := f(sdk.DefaultValues{SharedInfraToken: sdk.RandomString(32)}, database.GetDBMap); err != nil {
			return nil
		}
	}

	return database.DBMap(database.DB())
}
