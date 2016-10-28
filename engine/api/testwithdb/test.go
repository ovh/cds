package testwithdb

import (
	"database/sql"
	"encoding/base64"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//DBDriver is exported for testing purpose
var DBDriver string
var dbUser string
var dbPassword string
var dbName string
var dbHost string
var dbPort string
var dbSSLMode string

func init() {
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

// SetupPG setup PG DB for test
func SetupPG(t *testing.T) (*sql.DB, error) {
	log.Info("Setug PG Database connection")
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s connect_timeout=10 statement_timeout=3000", dbUser, dbPassword, dbName, dbHost, dbPort, dbSSLMode)

	db, err := sql.Open(DBDriver, dsn)
	if err != nil {
		log.Warning("Cannot open database: %s\n", err)
		return db, err
	}

	if err = db.Ping(); err != nil {
		return db, err
	}
	database.Set(db)

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

	if err := artifact.CreateBuiltinArtifactActions(db); err != nil {
		log.Critical("Cannot setup builtin Artifact actions: %s\n", err)
		return nil, err
	}

	if err := group.CreateDefaultGlobalGroup(db); err != nil {
		log.Critical("Cannot setup default global group: %s\n", err)
		return nil, err
	}

	if err := worker.CreateBuiltinActions(db); err != nil {
		log.Critical("Cannot setup builtin actions: %s\n", err)
		return nil, err
	}

	if err := worker.CreateBuiltinEnvironments(db); err != nil {
		log.Critical("Cannot setup builtin environments: %s\n", err)
		return nil, err
	}

	return db, nil
}

// InsertTestProject create a test project
func InsertTestProject(t *testing.T, db database.QueryExecuter, key, name string) (*sdk.Project, error) {
	proj := sdk.Project{
		Key:  key,
		Name: name,
	}
	t.Logf("Insert Project %s", key)
	return &proj, project.InsertProject(db, &proj)
}

// DeleteTestProject delete a test project
func DeleteTestProject(t *testing.T, db database.QueryExecuter, key string) error {
	t.Logf("Delete Project %s", key)
	return project.DeleteProject(db, key)
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

// InsertAdminUser have to be used only for tests
func InsertAdminUser(t *testing.T, db *sql.DB) (*sdk.User, string, error) {
	s := RandomString(t, 10)
	password, hash, _ := user.GeneratePassword()
	u := &sdk.User{
		Admin:    true,
		Email:    "no-reply-" + s + "@corp.ovh.com",
		Username: s,
		Origin:   "local",
		Fullname: "Test " + s,
		Auth: sdk.Auth{
			EmailVerified:  true,
			HashedPassword: hash,
		},
	}
	user.InsertUser(db, u, &u.Auth)
	return u, password, nil
}

// AuthentifyRequest  have to be used only for tests
func AuthentifyRequest(t *testing.T, req *http.Request, u *sdk.User, pass string) {
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(u.Username+":"+pass))
	req.Header.Add("Authorization", auth)
}
