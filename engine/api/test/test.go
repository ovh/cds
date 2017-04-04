package test

import (
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database"
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

		log.Initialize("debug")
		flag.Parse()
	}
}

type bootstrap func(func() *gorp.DbMap) error

// SetupPG setup PG DB for test
func SetupPG(t *testing.T, bootstrapFunc ...bootstrap) *gorp.DbMap {
	DBDriver = flag.Lookup("dbDriver").Value.String()
	dbUser = flag.Lookup("dbUser").Value.String()
	dbPassword = flag.Lookup("dbPassword").Value.String()
	dbName = flag.Lookup("dbName").Value.String()
	dbHost = flag.Lookup("dbHost").Value.String()
	dbPort = flag.Lookup("dbPort").Value.String()
	dbSSLMode = flag.Lookup("sslMode").Value.String()

	if DBDriver == "" {
		t.Skip("This should be run with a database")
		return nil
	}
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s connect_timeout=10 statement_timeout=5000", dbUser, dbPassword, dbName, dbHost, dbPort, dbSSLMode)

	if database.DB() == nil {
		db, err := sql.Open(DBDriver, dsn)
		if err != nil {
			t.Fatalf("Cannot open database: %s\n", err)
			return nil
		}

		if err = db.Ping(); err != nil {
			t.Fatalf("Cannot ping database: %s\n", err)
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
			log.Warning("Cleanup SQL connections\n")
			db.Close()
			os.Exit(0)
		}()
	}

	for _, f := range bootstrapFunc {
		if err := f(database.GetDBMap); err != nil {
			return nil
		}
	}

	return database.DBMap(database.DB())
}

// RandomString have to be used only for tests
func RandomString(t *testing.T, strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
