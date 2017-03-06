package assets

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

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

// InsertTestProject create a test project
func InsertTestProject(t *testing.T, db *gorp.DbMap, key, name string) *sdk.Project {
	proj := sdk.Project{
		Key:  key,
		Name: name,
	}
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

	if err := project.Insert(db, &proj); err != nil {
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
	return project.Delete(db, key)
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
		u.Groups = append(u.Groups, *g)
	}
	return u, password
}

// AuthentifyRequestFromWorker have to be used only for tests
func AuthentifyRequestFromWorker(t *testing.T, req *http.Request, w *sdk.Worker) {
	req.Header.Add(sdk.AuthHeader, base64.StdEncoding.EncodeToString([]byte(w.ID)))
}

// NewAuthentifiedRequestFromWorker prepare a request
func NewAuthentifiedRequestFromWorker(t *testing.T, w *sdk.Worker, method, uri string, i interface{}) *http.Request {
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

	AuthentifyRequestFromWorker(t, req, w)

	return req
}

// AuthHeaders set auth headers
func AuthHeaders(t *testing.T, u *sdk.User, pass string) http.Header {
	h := http.Header{}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(u.Username+":"+pass))
	h.Add("Authorization", auth)
	return h
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
