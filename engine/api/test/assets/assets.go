package assets

import (
	"bytes"
	"context"
	"crypto/rsa"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

// InsertTestProject create a test project
func InsertTestProject(t *testing.T, db *gorp.DbMap, store cache.Store, key, name string, u *sdk.AuthentifiedUser) *sdk.Project {

	oldProj, _ := project.Load(db, store, key, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithWorkflows)
	if oldProj != nil {
		for _, w := range oldProj.Workflows {
			require.NoError(t, workflow.Delete(context.TODO(), db, store, oldProj, &w))
		}
		for _, app := range oldProj.Applications {
			require.NoError(t, application.DeleteApplication(db, app.ID))
		}
		for _, pip := range oldProj.Pipelines {
			require.NoError(t, pipeline.DeletePipeline(context.TODO(), db, pip.ID))
		}
		require.NoError(t, project.Delete(db, store, key))
	}

	proj := sdk.Project{
		Key:  key,
		Name: name,
	}

	g := InsertTestGroup(t, db, name+"-group")

	if err := project.Insert(db, store, &proj); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
		return nil
	}

	if err := group.InsertGroupInProject(db, proj.ID, g.ID, sdk.PermissionReadWriteExecute); err != nil {
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
		Username: sdk.RandomString(10),
		Fullname: sdk.RandomString(10),
		Ring:     sdk.UserRingAdmin,
	}

	if err := user.Insert(db, &data); err != nil {
		log.Error("unable to insert user: %+v", err)
	}

	u, err := user.LoadByID(context.Background(), db, data.ID, user.LoadOptions.WithDeprecatedUser, user.LoadOptions.WithContacts)
	if err != nil {
		log.Error("user cannot be load for id %s: %v", data.ID, err)
	}

	consumer, err := local.NewConsumer(db, u.ID)
	if err != nil {
		log.Error("cannot create auth consumer: %v", err)
	}

	session, err := authentication.NewSession(db, consumer, 5*time.Minute)
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
		Username: sdk.RandomString(10),
		Fullname: sdk.RandomString(10),
		Ring:     sdk.UserRingUser,
	}

	if err := user.Insert(db, u); err != nil {
		log.Fatalf(" user.Insert: %v", err)
	}

	u, err := user.LoadByID(context.Background(), db, u.ID, user.LoadOptions.WithDeprecatedUser, user.LoadOptions.WithContacts)
	if err != nil {
		log.Fatalf(" user.LoadUserByID: %v", err)
	}

	for _, g := range groups {
		if err := group.InsertGroup(db, g); err != nil {
			log.Error("unable to insert group: %v", err)
		}
		if err := group.InsertUserInGroup(db, g.ID, u.OldUserStruct.ID, false); err != nil {
			log.Error("unable to insert user in group: %v", err)

		}
		u.OldUserStruct.Groups = append(u.OldUserStruct.Groups, *g)
	}

	btes, _ := json.Marshal(u)

	log.Debug("lambda user: %s", string(btes))

	consumer, err := local.NewConsumer(db, u.ID)
	if err != nil {
		log.Error("cannot create auth consumer: %v", err)
	}

	session, err := authentication.NewSession(db, consumer, 5*time.Minute)
	if err != nil {
		log.Error("cannot create auth session: %v", err)
	}

	jwt, err := authentication.NewSessionJWT(session)
	if err != nil {
		log.Error("cannot create jwt: %v", err)
	}

	return u, jwt
}

// AuthentifyRequest  have to be used only for tests
func AuthentifyRequest(t *testing.T, req *http.Request, u *sdk.AuthentifiedUser, jwt string) {
	auth := "Bearer " + jwt
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
	date := sdk.FormatDateRFC5322(time.Now())
	req.Header.Set("Date", date)
	req.Header.Set("X-CDS-RemoteTime", date)

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

	date := sdk.FormatDateRFC5322(time.Now())
	req.Header.Set("Date", date)
	req.Header.Set("X-CDS-RemoteTime", date)

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

func NewJWTAuthentifiedMultipartRequest(t *testing.T, jwt string, method, uri string, path string, fileName string, params map[string]string) *http.Request {
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

	auth := "Bearer " + jwt
	req.Header.Add("Authorization", auth)

	date := sdk.FormatDateRFC5322(time.Now())
	req.Header.Set("Date", date)
	req.Header.Set("X-CDS-RemoteTime", date)

	return req
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
	usr1, _ := InsertLambdaUser(db, &grp)

	consumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, usr1.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	hConsumer, _, err := builtin.NewConsumer(db, sdk.RandomString(10), "", consumer, []int64{grp.ID}, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution})
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
			ConsumerID: &hConsumer.ID,
		},
	}

	test.NoError(t, services.Insert(db, &srv))

	session, err := authentication.NewSession(db, hConsumer, 5*time.Minute)
	test.NoError(t, err)

	jwt, err := authentication.NewSessionJWT(session)
	test.NoError(t, err)

	return &srv, privateKey, hConsumer, jwt
}

func InsertService(t *testing.T, db gorp.SqlExecutor, name, serviceType string) (*sdk.Service, *rsa.PrivateKey) {
	usr1, _ := InsertAdminUser(db)

	consumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, usr1.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	sharedGroup, err := group.LoadByName(context.TODO(), db, sdk.SharedInfraGroupName)
	require.NoError(t, err)
	hConsumer, _, err := builtin.NewConsumer(db, sdk.RandomString(10), "", consumer, []int64{sharedGroup.ID}, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeProject})
	require.NoError(t, err)

	privateKey, err := jws.NewRandomRSAKey()
	require.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	require.NoError(t, err)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:       hConsumer.Name,
			Type:       serviceType,
			PublicKey:  publicKey,
			Maintainer: *usr1,
			ConsumerID: &hConsumer.ID,
		},
	}

	require.NoError(t, services.Insert(db, &srv))

	return &srv, privateKey
}

func InsertTestWorkflow(t *testing.T, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, name string) *sdk.Workflow {
	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, store, proj, &pip))

	script := GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(db, s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	w := sdk.Workflow{
		Name:       name,
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, store, &w, proj))

	return &w
}
