package assets

import (
	"bytes"
	"context"
	"crypto/rsa"
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

	"github.com/ovh/cds/engine/api/authentication"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

// InsertTestProject create a test project
func InsertTestProject(t *testing.T, db *gorp.DbMap, store cache.Store, key, name string, u *sdk.AuthentifiedUser) *sdk.Project {
	proj := sdk.Project{
		Key:  key,
		Name: name,
	}

	g := InsertTestGroup(t, db, name+"-group")

	if err := project.Insert(db, store, &proj); err != nil {
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

	eg, _ := group.LoadByName(context.TODO(), db, g.Name)
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
func InsertAdminUser(db gorp.SqlExecutor) (*sdk.AuthentifiedUser, string) {
	data := sdk.AuthentifiedUser{
		Username:     sdk.RandomString(10),
		Fullname:     sdk.RandomString(10),
		Ring:         sdk.UserRingAdmin,
		DateCreation: time.Now(),
	}

	user.Insert(db, &data)

	u, err := user.LoadByID(context.Background(), db, data.ID, user.LoadOptions.WithDeprecatedUser, user.LoadOptions.WithContacts)
	if err != nil {
		log.Error("user cannot be load for id %s: %v", data.ID, err)
	}

	consumer, err := authentication.NewConsumerBuiltin(db, "Test consumer for user "+data.Username, "", u.ID, nil, []string{sdk.AccessTokenScopeALL})
	if err != nil {
		log.Error("cannot create auth consumer: %v", err)
	}

	expiration := time.Now().Add(5 * time.Minute)
	session, err := authentication.NewSession(db, consumer, expiration)
	if err != nil {
		log.Error("cannot create auth session: %v", err)
	}

	jwt, err := authentication.NewSessionJWT(session)
	if err != nil {
		log.Error("cannot create jwt: %v", err)
	}

	return u, jwt
}

// InsertLambdaUser have to be used only for tests
func InsertLambdaUser(db gorp.SqlExecutor, groups ...*sdk.Group) (*sdk.AuthentifiedUser, string) {
	var u = &sdk.AuthentifiedUser{
		Username:     sdk.RandomString(10),
		Fullname:     sdk.RandomString(10),
		Ring:         sdk.UserRingUser,
		DateCreation: time.Now(),
	}

	if err := user.Insert(db, u); err != nil {
		log.Fatalf(" user.Insert: %v", err)
	}

	u, err := user.LoadByID(context.Background(), db, u.ID, user.LoadOptions.WithDeprecatedUser, user.LoadOptions.WithContacts)
	if err != nil {
		log.Fatalf(" user.LoadUserByID: %v", err)
	}

	for _, g := range groups {
		group.InsertGroup(db, g)
		group.InsertUserInGroup(db, g.ID, u.OldUserStruct.ID, false)
		u.OldUserStruct.Groups = append(u.OldUserStruct.Groups, *g)
	}

	btes, _ := json.Marshal(u)

	log.Debug("lambda user: %s", string(btes))

	consumer, err := authentication.NewConsumerBuiltin(db, "Test consumer for user "+u.Username, "", u.ID,
		sdk.GroupPointersToIDs(groups), []string{sdk.AccessTokenScopeALL})
	if err != nil {
		log.Error("cannot create auth consumer: %v", err)
	}

	expiration := time.Now().Add(5 * time.Minute)
	session, err := authentication.NewSession(db, consumer, expiration)
	if err != nil {
		log.Error("cannot create auth session: %v", err)
	}

	jwt, err := authentication.NewSessionJWT(session)
	if err != nil {
		log.Error("cannot create jwt: %v", err)
	}

	return u, jwt
}

// AuthentifyRequestFromProvider have to be used only for tests
func AuthentifyRequestFromProvider(t *testing.T, req *http.Request, name, token string) {
	req.Header.Add("X-Provider-Name", name)
	req.Header.Add("X-Provider-Token", token)
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

	AuthentifyRequestFromService(t, req, "h.Hash")
	return req
}

// AuthentifyRequest  have to be used only for tests
func AuthentifyRequest(t *testing.T, req *http.Request, u *sdk.AuthentifiedUser, token string) {
	req.Header.Add(cdsclient.RequestedWithHeader, cdsclient.RequestedWithValue)
	req.Header.Add(cdsclient.SessionTokenHeader, token)
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(u.Username+":"+token))
	req.Header.Add("Authorization", auth)
}

//NewAuthentifiedRequest prepare a request
func NewAuthentifiedRequest(t *testing.T, u *sdk.AuthentifiedUser, pass, method, uri string, i interface{}) *http.Request {
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

/*func NewJWTToken(t *testing.T, db gorp.SqlExecutor, u sdk.AuthentifiedUser, groups ...sdk.Group) (string, error) {
	expiration := time.Now().Add(5 * time.Minute)
	token, jwt, err := authentication.New(u, groups, []string{sdk.AccessTokenScopeALL}, "test", sdk.RandomString(5), expiration)
	if err != nil {
		return "", err
	}
	err = authentication.Insert(db, &token)
	return jwt, err
}*/

/*func NewJWTTokenWithXSRF(t *testing.T, db gorp.SqlExecutor, store cache.Store, u sdk.AuthentifiedUser, groups ...sdk.Group) (string, string, error) {
	expiration := time.Now().Add(5 * time.Minute)
	token, jwt, err := authentication.New(u, groups, []string{sdk.AccessTokenScopeALL}, authentication.OriginUI, sdk.RandomString(5), expiration)
	if err != nil {
		return "", "", err
	}
	err = authentication.Insert(db, &token)
	if err != nil {
		return "", "", err
	}

	xsrf := authentication.StoreXSRFToken(store, token)
	return jwt, xsrf, err
}*/

// GetBuiltinOrPluginActionByName returns a builtin or plugin action for given name if exists.
func GetBuiltinOrPluginActionByName(t *testing.T, db gorp.SqlExecutor, name string) *sdk.Action {
	a, err := action.LoadByTypesAndName(context.TODO(), db, []string{sdk.BuiltinAction, sdk.PluginAction}, name,
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

func InsertGroup(t *testing.T, db gorp.SqlExecutor) *sdk.Group {
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	g1, _ := group.LoadByName(context.TODO(), db, g.Name)
	if g1 != nil {
		models, _ := workermodel.LoadAllByGroupIDs(context.Background(), db, []int64{g.ID}, nil)
		for _, m := range models {
			workermodel.Delete(db, m.ID)
		}

		if err := group.DeleteGroupAndDependencies(db, g1); err != nil {
			t.Logf("unable to delete group: %v", err)
		}
	}

	if err := group.InsertGroup(db, g); err != nil {
		t.Fatalf("Unable to create group %s", err)
	}

	return g
}

func InsertWorkerModel(t *testing.T, db gorp.SqlExecutor, name string, groupID int64) *sdk.Model {
	m := sdk.Model{
		Name: name,
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "foo/bar:3.4",
		},
		GroupID: groupID,
		RegisteredCapabilities: sdk.RequirementList{
			{
				Name:  "capa_1",
				Type:  sdk.BinaryRequirement,
				Value: "capa_1",
			},
		},
		UserLastModified: time.Now(),
	}

	if err := workermodel.Insert(db, &m); err != nil {
		t.Fatalf("Cannot insert worker model: %s", err)
	}

	assert.NotEqual(t, 0, m.ID)
	return &m
}

func InsertHatchery(t *testing.T, db gorp.SqlExecutor, grp sdk.Group) (*sdk.Service, *rsa.PrivateKey, *sdk.AuthConsumer, string) {
	usr1, _ := InsertLambdaUser(db)

	hConsumer, err := authentication.NewConsumerBuiltin(db, sdk.RandomString(10), "", usr1.ID, []int64{grp.ID}, []string{sdk.AccessTokenScopeALL})
	test.NoError(t, err)

	privateKey, err := jws.NewRandomRSAKey()
	test.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	test.NoError(t, err)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:       hConsumer.Name,
			Type:       services.TypeHatchery,
			PublicKey:  publicKey,
			Maintainer: *usr1,
			ConsumerID: hConsumer.ID,
		},
	}

	test.NoError(t, services.Insert(db, &srv))

	session, err := authentication.NewSession(db, hConsumer, time.Now().Add(time.Hour))
	test.NoError(t, err)

	jwt, err := authentication.NewSessionJWT(session)
	test.NoError(t, err)

	return &srv, privateKey, hConsumer, jwt
}

func InsertService(t *testing.T, db gorp.SqlExecutor, name, serviceType string) (*sdk.Service, *rsa.PrivateKey) {
	usr1, _ := InsertAdminUser(db)

	//exp := time.Now().Add(5 * time.Minute)
	//token, _, err := authentication.New(*usr1, nil, []string{sdk.AccessTokenScopeALL}, "cds_test", name, exp)
	//test.NoError(t, err)

	//test.NoError(t, authentication.Insert(db, &token))

	privateKey, err := jws.NewRandomRSAKey()
	test.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	test.NoError(t, err)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:       name,
			Type:       serviceType,
			PublicKey:  publicKey,
			Maintainer: *usr1,
			//TokenID:    token.ID,
		},
	}

	test.NoError(t, services.Insert(db, &srv))

	return &srv, privateKey
}
