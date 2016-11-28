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

	"bytes"
	"encoding/json"

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
func SetupPG(t *testing.T, bootstrapFunc ...bootstrap) (*sql.DB, error) {
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s connect_timeout=10 statement_timeout=3000", dbUser, dbPassword, dbName, dbHost, dbPort, dbSSLMode)

	db, err := sql.Open(DBDriver, dsn)
	if err != nil {
		t.Fatalf("Cannot open database: %s\n", err)
		return db, err
	}

	if err = db.Ping(); err != nil {
		t.Fatalf("Cannot ping database: %s\n", err)
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

	for _, f := range bootstrapFunc {
		if err := f(db); err != nil {
			return nil, err
		}
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

	g := sdk.Group{
		Name: name + "-group",
	}

	eg, _ := group.LoadGroup(db, g.Name)
	if eg != nil {
		g = *eg
	} else {
		if err := group.InsertGroup(db, &g); err != nil {
			t.Fatalf("Cannot insert group : %s", err)
			return nil, err
		}
	}

	if err := project.InsertProject(db, &proj); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
		return nil, err
	}

	if err := group.InsertGroupInProject(db, proj.ID, g.ID, permission.PermissionReadWriteExecute); err != nil {
		t.Fatalf("Cannot insert permission : %s", err)
		return nil, err
	}

	if err := group.LoadGroupByProject(db, &proj); err != nil {
		t.Fatalf("Cannot load permission : %s", err)
		return nil, err
	}

	return &proj, nil
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

// InsertLambaUser have to be used only for tests
func InsertLambaUser(t *testing.T, db database.QueryExecuter, groups ...*sdk.Group) (*sdk.User, string, error) {
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
	return u, password, nil
}

// AuthentifyRequest  have to be used only for tests
func AuthentifyRequest(t *testing.T, req *http.Request, u *sdk.User, pass string) {
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(u.Username+":"+pass))
	req.Header.Add("Authorization", auth)
}

//NewAuthentifiedRequest prepare a request
func NewAuthentifiedRequest(t *testing.T, u *sdk.User, pass, method, uri string, i interface{}) *http.Request {
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		if err != nil {
			t.FailNow()
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.FailNow()
	}
	AuthentifyRequest(t, req, u, pass)

	return req

}
