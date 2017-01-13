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
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
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

type bootstrap func(db *sql.DB) error

// SetupPG setup PG DB for test
func SetupPG(t *testing.T, bootstrapFunc ...bootstrap) *gorp.DbMap {
	if DBDriver == "" {
		t.Skip("This is should be run with a database")
		return nil
	}
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s connect_timeout=10 statement_timeout=5000", dbUser, dbPassword, dbName, dbHost, dbPort, dbSSLMode)

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

	for _, f := range bootstrapFunc {
		if err := f(db); err != nil {
			return nil
		}
	}

	return database.DBMap(db)
}

// InsertTestProject create a test project
func InsertTestProject(t *testing.T, db *gorp.DbMap, key, name string) *sdk.Project {
	proj := sdk.Project{
		Key:  key,
		Name: name,
	}
	t.Logf("Insert Project %s", key)

	g := sdk.Group{
		Name: name + "-group",
	}

	eg, _ := group.LoadGroup(db, g.Name)
	if eg != nil {
		g = *eg
	} else if err := group.InsertGroup(db, &g); err != nil {
		t.Fatalf("Cannot insert group : %s", err)
		return nil
	}

	if err := project.InsertProject(db, &proj); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
		return nil
	}

	if err := group.InsertGroupInProject(db, proj.ID, g.ID, permission.PermissionReadWriteExecute); err != nil {
		t.Fatalf("Cannot insert permission : %s", err)
		return nil
	}

	if err := group.LoadGroupByProject(db, &proj); err != nil {
		t.Fatalf("Cannot load permission : %s", err)
		return nil
	}

	return &proj
}

// DeleteTestProject delete a test project
func DeleteTestProject(t *testing.T, db gorp.SqlExecutor, key string) error {
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
func InsertAdminUser(t *testing.T, db *gorp.DbMap) (*sdk.User, string) {
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
	return u, password
}

// InsertLambaUser have to be used only for tests
func InsertLambaUser(t *testing.T, db gorp.SqlExecutor, groups ...*sdk.Group) (*sdk.User, string) {
	s := RandomString(t, 10)
	password, hash, _ := user.GeneratePassword()
	u := &sdk.User{
		Admin:    false,
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
	for _, g := range groups {
		group.InsertGroup(db, g)
		group.InsertUserInGroup(db, g.ID, u.ID, false)
	}
	return u, password
}
