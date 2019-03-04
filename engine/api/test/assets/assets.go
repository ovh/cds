package assets

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

// InsertTestProject create a test project
func InsertTestProject(t *testing.T, db *gorp.DbMap, store cache.Store, key, name string, u *sdk.User) *sdk.Project {
	proj := sdk.Project{
		Key:  key,
		Name: name,
	}

	g := InsertTestGroup(t, db, name+"-group")

	if err := project.Insert(db, store, &proj, u); err != nil {
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
func DeleteTestProject(t *testing.T, db gorp.SqlExecutor, store cache.Store, key string) error {
	t.Logf("Delete Project %s", key)
	return project.Delete(db, store, key)
}

// InsertTestGroup create a test group
func InsertTestGroup(t *testing.T, db *gorp.DbMap, name string) *sdk.Group {
	g := sdk.Group{
		Name: name,
	}

	eg, _ := group.LoadGroup(db, g.Name)
	if eg != nil {
		g = *eg
	} else if err := group.InsertGroup(db, &g); err != nil {
		t.Fatalf("Cannot insert group : %s", err)
		return nil
	}

	return &g
}

// DeleteTestGroup delete a test group.
func DeleteTestGroup(t *testing.T, db gorp.SqlExecutor, g *sdk.Group) error {
	t.Logf("Delete Group %s", g.Name)
	return group.DeleteGroupAndDependencies(db, g)
}

// InsertAdminUser have to be used only for tests
func InsertAdminUser(db *gorp.DbMap) (*sdk.User, string) {
	s := sdk.RandomString(10)
	_, hash, _ := user.GeneratePassword()
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

	t, _ := user.NewPersistentSession(db, u)
	return u, string(t)
}

// InsertLambdaUser have to be used only for tests
func InsertLambdaUser(db gorp.SqlExecutor, groups ...*sdk.Group) (*sdk.User, string) {
	s := sdk.RandomString(10)
	_, hash, _ := user.GeneratePassword()
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

	t, _ := user.NewPersistentSession(db, u)
	return u, string(t)
}

// AuthentifyRequestFromWorker have to be used only for tests
func AuthentifyRequestFromWorker(t *testing.T, req *http.Request, w *sdk.Worker) {
	req.Header.Set("User-Agent", string(sdk.WorkerAgent))
	req.Header.Add(sdk.AuthHeader, base64.StdEncoding.EncodeToString([]byte(w.ID)))
}

// AuthentifyRequestFromService have to be used only for tests
func AuthentifyRequestFromService(t *testing.T, req *http.Request, hash string) {
	req.Header.Add("User-Agent", string(sdk.ServiceAgent))
	req.Header.Add(sdk.AuthHeader, base64.StdEncoding.EncodeToString([]byte(hash)))
}

// AuthentifyRequestFromProvider have to be used only for tests
func AuthentifyRequestFromProvider(t *testing.T, req *http.Request, name, token string) {
	req.Header.Add("X-Provider-Name", name)
	req.Header.Add("X-Provider-Token", token)
}

// NewAuthentifiedRequestFromWorker prepare a request
func NewAuthentifiedRequestFromWorker(t *testing.T, w *sdk.Worker, method, uri string, i interface{}) *http.Request {
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	AuthentifyRequestFromWorker(t, req, w)

	return req
}

// NewAuthentifiedRequestFromProvider prepare a request
func NewAuthentifiedRequestFromProvider(t *testing.T, name, token, method, uri string, i interface{}) *http.Request {
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	AuthentifyRequestFromProvider(t, req, name, token)

	return req
}

// NewAuthentifiedMultipartRequestFromWorker  prepare multipart request with file to upload
func NewAuthentifiedMultipartRequestFromWorker(t *testing.T, w *sdk.Worker, method, uri string, path string, fileName string, params map[string]string) *http.Request {
	file, err := os.Open(path)
	if err != nil {
		t.Fail()
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fileName, filepath.Base(path))
	if err != nil {
		t.Fail()
	}
	if _, err := io.Copy(part, file); err != nil {
		t.Fail()
	}

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	contextType := writer.FormDataContentType()

	if err := writer.Close(); err != nil {
		t.Fail()
	}

	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		t.Fail()
	}
	req.Header.Set("Content-Type", contextType)
	req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	req.Header.Set("ARTIFACT-FILENAME", fileName)

	AuthentifyRequestFromWorker(t, req, w)

	return req
}

// NewAuthentifiedRequestFromHatchery prepare a request
func NewAuthentifiedRequestFromHatchery(t *testing.T, h *sdk.Service, method, uri string, i interface{}) *http.Request {
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

	AuthentifyRequestFromService(t, req, h.Hash)
	return req
}

// AuthentifyRequest  have to be used only for tests
func AuthentifyRequest(t *testing.T, req *http.Request, u *sdk.User, token string) {
	req.Header.Add(sdk.RequestedWithHeader, sdk.RequestedWithValue)
	req.Header.Add(sdk.SessionTokenHeader, token)
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(u.Username+":"+token))
	req.Header.Add("Authorization", auth)
}

//NewAuthentifiedRequest prepare a request
func NewAuthentifiedRequest(t *testing.T, u *sdk.User, pass, method, uri string, i interface{}) *http.Request {
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	AuthentifyRequest(t, req, u, pass)

	return req
}

// NewJWTAuthentifiedRequest prepare a request
func NewJWTAuthentifiedRequest(t *testing.T, jwt string, method, uri string, i interface{}) *http.Request {
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	auth := "Bearer " + jwt
	req.Header.Add("Authorization", auth)

	return req
}

// NewXSRFJWTAuthentifiedRequest prepare a request
func NewXSRFJWTAuthentifiedRequest(t *testing.T, jwt, xsrf string, method, uri string, i interface{}) *http.Request {
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	req.Header.Add("X-XSRF-TOKEN", xsrf)
	c := http.Cookie{
		Name:  "jwt_token",
		Value: jwt,
	}

	req.AddCookie(&c)
	return req
}

func NewJWTToken(t *testing.T, db gorp.SqlExecutor, u sdk.User, groups ...sdk.Group) (string, error) {
	expiration := time.Now().Add(5 * time.Minute)
	token, jwt, err := accesstoken.New(u, groups, "test", sdk.RandomString(5), &expiration)
	if err != nil {
		return "", err
	}
	err = accesstoken.Insert(db, &token)
	return jwt, err
}

func NewJWTTokenWithXSRF(t *testing.T, db gorp.SqlExecutor, store cache.Store, u sdk.User, groups ...sdk.Group) (string, string, error) {
	expiration := time.Now().Add(5 * time.Minute)
	token, jwt, err := accesstoken.New(u, groups, accesstoken.OriginUI, sdk.RandomString(5), &expiration)
	if err != nil {
		return "", "", err
	}
	err = accesstoken.Insert(db, &token)
	if err != nil {
		return "", "", err
	}

	xsrf := accesstoken.StoreXSRFToken(store, token)
	return jwt, xsrf, err
}

// GetBuiltinOrPluginActionByName returns a builtin or plugin action for given name if exists.
func GetBuiltinOrPluginActionByName(t *testing.T, db gorp.SqlExecutor, name string) *sdk.Action {
	a, err := action.LoadByTypesAndName(db, []string{sdk.BuiltinAction, sdk.PluginAction}, name,
		action.LoadOptions.WithRequirements,
		action.LoadOptions.WithParameters,
		action.LoadOptions.WithGroup,
	)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if a == nil {
		t.Errorf("cannot find builtin or plugin action with name %s", name)
		t.FailNow()
	}
	return a
}

// NewAction returns an enabled action.
func NewAction(id int64, ps ...sdk.Parameter) sdk.Action {
	return sdk.Action{
		ID:         id,
		Enabled:    true,
		Parameters: ps,
	}
}
