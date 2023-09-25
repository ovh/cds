package assets

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	workerauth "github.com/ovh/cds/engine/api/authentication/worker"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/worker_v2"
	sdkhatch "github.com/ovh/cds/sdk/hatchery"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
)

func InsertRBAcPublicProject(t *testing.T, db gorpmapper.SqlExecutorWithTx, role string, projKey string) {
	perm := fmt.Sprintf(`name: perm-%s-project-%s
projects:
  - role: %s
    projects: [%s]
    all_users: true
`, role, projKey, role, projKey)

	var rb sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &rb))
	rb.Projects[0].RBACProjectKeys = []string{projKey}
	require.NoError(t, rbac.Insert(context.Background(), db, &rb))
}

func InsertRBAcProject(t *testing.T, db gorpmapper.SqlExecutorWithTx, role string, projKey string, user sdk.AuthentifiedUser) {
	perm := fmt.Sprintf(`name: perm-%s-project-%s
projects:
  - role: %s
    projects: [%s]
    users: [%s]
`, role, projKey, role, projKey, user.Username)

	var rb sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &rb))
	rb.Projects[0].RBACProjectKeys = []string{projKey}
	rb.Projects[0].RBACUsersIDs = []string{user.ID}
	require.NoError(t, rbac.Insert(context.Background(), db, &rb))
}

func InsertWorker(t *testing.T, ctx context.Context, db gorpmapper.SqlExecutorWithTx, hatcheryConsumer *sdk.AuthHatcheryConsumer, hatch sdk.Hatchery, workerName string, jobRun sdk.V2WorkflowRunJob) (*sdk.V2Worker, string) {
	workerConsumer, err := authentication.NewConsumerWorkerV2(ctx, db, workerName, hatcheryConsumer)
	require.NoError(t, err)

	spawnaargs := sdkhatch.SpawnArgumentsJWTV2{
		WorkerName: workerName,
		RunJobID:   jobRun.ID,
		ModelName:  jobRun.Job.WorkerModel,
	}
	wk, err := worker_v2.RegisterWorker(ctx, db, spawnaargs, hatch, workerConsumer, sdk.WorkerRegistrationForm{Arch: "amd64", OS: "linux"})
	require.NoError(t, err)

	workerSession, err := authentication.NewSession(ctx, db, &workerConsumer.AuthConsumer, workerauth.SessionDuration)
	require.NoError(t, err)

	jwt, err := authentication.NewSessionJWT(workerSession, "")
	require.NoError(t, err)

	return wk, jwt
}

// InsertTestProject create a test project.
func InsertTestProject(t *testing.T, db gorpmapper.SqlExecutorWithTx, store cache.Store, key, name string) *sdk.Project {
	oldProj, _ := project.Load(context.TODO(), db, key,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithWorkflows,
	)
	if oldProj != nil {
		for _, w := range oldProj.Workflows {
			require.NoError(t, workflow.Delete(context.TODO(), db, store, *oldProj, &w))
		}
		for _, app := range oldProj.Applications {
			require.NoError(t, application.DeleteApplication(db, app.ID))
		}
		for _, pip := range oldProj.Pipelines {
			require.NoError(t, pipeline.DeletePipeline(context.TODO(), db, pip.ID))
		}
		require.NoError(t, project.Delete(db, key))
	}

	proj := &sdk.Project{Key: key, Name: name}

	g := InsertTestGroup(t, db, name+"-group")

	require.NoError(t, project.Insert(db, proj))

	require.NoError(t, group.InitializeDefaultGroupName(db, ""))

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	var err error
	proj, err = project.LoadByID(db, proj.ID, project.LoadOptions.WithGroups)
	require.NoError(t, err)

	return proj
}

func InsertTestVCSProject(t *testing.T, db gorpmapper.SqlExecutorWithTx, projID int64, name string, vcsType string) *sdk.VCSProject {
	vcsProj := sdk.VCSProject{
		Name:         name,
		Type:         vcsType,
		Created:      time.Now(),
		LastModified: time.Now(),
		CreatedBy:    "InsertTestVCSProject",
		ProjectID:    projID,
	}
	err := vcs.Insert(context.TODO(), db, &vcsProj)
	require.NoError(t, err)
	return &vcsProj
}

func InsertTestProjectRepository(t *testing.T, db gorpmapper.SqlExecutorWithTx, pKey string, vcsServerID string, name string) *sdk.ProjectRepository {
	repo := sdk.ProjectRepository{
		Name:         strings.ToLower(name),
		Created:      time.Now(),
		VCSProjectID: vcsServerID,
		CreatedBy:    "test",
		CloneURL:     "myurl",
		ProjectKey:   pKey,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))
	return &repo
}

// DeleteTestProject delete a test project
func DeleteTestProject(t *testing.T, db gorp.SqlExecutor, store cache.Store, key string) error {
	t.Logf("Delete Project %s", key)
	return project.Delete(db, key)
}

// InsertTestGroup create a test group
func InsertTestGroup(t *testing.T, db gorpmapper.SqlExecutorWithTx, name string) *sdk.Group {
	return InsertTestGroupInOrganization(t, db, name, "default")
}

// InsertTestGroupInOrganization create a test group
func InsertTestGroupInOrganization(t *testing.T, db gorpmapper.SqlExecutorWithTx, name string, orgaName string) *sdk.Group {
	g := sdk.Group{
		Name: name,
	}

	eg, _ := group.LoadByName(context.TODO(), db, g.Name, group.LoadOptions.WithOrganization)
	if eg != nil {
		g = *eg
	} else if err := group.Insert(context.TODO(), db, &g); err != nil {
		t.Fatalf("cannot insert group: %s", err)
		return nil
	}

	if g.Organization == "" {
		o, err := organization.LoadOrganizationByName(context.TODO(), db, orgaName)
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			o = &sdk.Organization{Name: orgaName}
			require.NoError(t, organization.Insert(context.TODO(), db, o))
			err = nil
		}
		grpOrg := group.GroupOrganization{
			OrganizationID: o.ID,
			GroupID:        g.ID,
		}
		require.NoError(t, group.InsertGroupOrganization(context.TODO(), db, &grpOrg))
	}

	return &g
}

// SetUserGroupAdmin allows a user to perform operations on given group
func SetUserGroupAdmin(t *testing.T, db gorpmapper.SqlExecutorWithTx, groupID int64, userID string) {
	l, err := group.LoadLinkGroupUserForGroupIDAndUserID(context.TODO(), db, groupID, userID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		t.Fatalf("cannot load link between group %d and user %s", groupID, userID)
		return
	}
	if l == nil {
		t.Fatalf("given user %s is not member of group %d", userID, groupID)
		return
	}

	if l.Admin {
		return
	}
	l.Admin = true

	if err := group.UpdateLinkGroupUser(context.Background(), db, l); err != nil {
		t.Fatalf("cannot set user %s group admin of %d", userID, groupID)
		return
	}
}

// DeleteTestGroup delete a test group.
func DeleteTestGroup(t *testing.T, db gorp.SqlExecutor, g *sdk.Group) {
	t.Logf("Delete Group %s", g.Name)
	require.NoError(t, group.Delete(context.TODO(), db, g))
}

// InsertAdminUser have to be used only for tests.
func InsertAdminUser(t *testing.T, db gorpmapper.SqlExecutorWithTx) (*sdk.AuthentifiedUser, string) {
	data := sdk.AuthentifiedUser{
		Username: "admin-" + sdk.RandomString(10),
		Fullname: "admin-" + sdk.RandomString(10),
		Ring:     sdk.UserRingAdmin,
	}
	require.NoError(t, user.Insert(context.TODO(), db, &data), "unable to insert user")

	o, err := organization.LoadOrganizationByName(context.TODO(), db, "default")
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		o = &sdk.Organization{Name: "default"}
		require.NoError(t, organization.Insert(context.TODO(), db, o))
		err = nil
	}
	require.NoError(t, err)

	uo := user.UserOrganization{
		OrganizationID:     o.ID,
		AuthentifiedUserID: data.ID,
	}
	require.NoError(t, user.InsertUserOrganization(context.TODO(), db, &uo))

	u, err := user.LoadByID(context.Background(), db, data.ID, user.LoadOptions.WithContacts)
	require.NoError(t, err, "user cannot be load for id %s", data.ID)

	consumer, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err, "cannot create auth consumer")

	session, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err, "cannot create auth session")

	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err, "cannot create jwt")

	return u, jwt
}

// DeleteAdmins delete all cds admins from database.
func DeleteAdmins(t *testing.T, db gorp.SqlExecutor) {
	us, err := user.LoadAllByRing(context.TODO(), db, sdk.UserRingAdmin)
	require.NoError(t, err)
	for i := range us {
		require.NoError(t, user.DeleteByID(db, us[i].ID))
	}
}

// DeleteConsumers delete all cds consumers from database.
func DeleteConsumers(t *testing.T, db gorp.SqlExecutor) {
	_, err := db.Exec("DELETE FROM auth_consumer")
	require.NoError(t, err, "can't to delete all auth consumer")
}

// InsertMaintainerUser have to be used only for tests.
func InsertMaintainerUser(t *testing.T, db gorpmapper.SqlExecutorWithTx) (*sdk.AuthentifiedUser, string) {
	data := sdk.AuthentifiedUser{
		Username: "maintainer-" + sdk.RandomString(10),
		Fullname: "maintainer-" + sdk.RandomString(10),
		Ring:     sdk.UserRingMaintainer,
	}
	require.NoError(t, user.Insert(context.TODO(), db, &data), "unable to insert user")

	o, err := organization.LoadOrganizationByName(context.TODO(), db, "default")
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		o = &sdk.Organization{Name: "default"}
		require.NoError(t, organization.Insert(context.TODO(), db, o))
		err = nil
	}
	uo := user.UserOrganization{
		OrganizationID:     o.ID,
		AuthentifiedUserID: data.ID,
	}
	require.NoError(t, user.InsertUserOrganization(context.TODO(), db, &uo))

	u, err := user.LoadByID(context.Background(), db, data.ID, user.LoadOptions.WithContacts)
	require.NoErrorf(t, err, "user cannot be load for id %s", data.ID)

	consumer, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err, "cannot create auth consumer")

	session, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err, "cannot create auth session")

	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err, "cannot create jwt")

	return u, jwt
}

// InsertLambdaUser have to be used only for tests.
func InsertLambdaUser(t *testing.T, db gorpmapper.SqlExecutorWithTx, groups ...*sdk.Group) (*sdk.AuthentifiedUser, string) {
	return InsertLambdaUserInOrganization(t, db, "default", groups...)
}

// InsertLambdaUserInOrganization have to be used only for tests.
func InsertLambdaUserInOrganization(t *testing.T, db gorpmapper.SqlExecutorWithTx, orgaName string, groups ...*sdk.Group) (*sdk.AuthentifiedUser, string) {
	u := &sdk.AuthentifiedUser{
		Username: "lambda-" + sdk.RandomString(10),
		Fullname: "lambda-" + sdk.RandomString(10),
		Ring:     sdk.UserRingUser,
	}
	require.NoError(t, user.Insert(context.TODO(), db, u))

	o, err := organization.LoadOrganizationByName(context.TODO(), db, orgaName)
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		o = &sdk.Organization{Name: orgaName}
		require.NoError(t, organization.Insert(context.TODO(), db, o))
		err = nil
	}
	require.NoError(t, err)
	uo := user.UserOrganization{
		OrganizationID:     o.ID,
		AuthentifiedUserID: u.ID,
	}
	require.NoError(t, user.InsertUserOrganization(context.TODO(), db, &uo))

	u, err = user.LoadByID(context.Background(), db, u.ID, user.LoadOptions.WithOrganization)
	require.NoError(t, err)

	for i := range groups {
		existingGroup, _ := group.LoadByName(context.TODO(), db, groups[i].Name)
		if existingGroup == nil {
			require.NoError(t, group.Create(context.Background(), db, groups[i], u))
			require.NoError(t, group.EnsureOrganization(context.TODO(), db, groups[i]))
		} else {
			groups[i].ID = existingGroup.ID
			require.NoError(t, group.InsertLinkGroupUser(context.Background(), db,
				&group.LinkGroupUser{
					GroupID:            groups[i].ID,
					AuthentifiedUserID: u.ID,
					Admin:              false,
				}), "unable to insert user in group")
		}
		u.Groups = append(u.Groups, *groups[i])
	}

	btes, err := json.Marshal(u)
	require.NoError(t, err)
	log.Debug(context.TODO(), "lambda user: %s", string(btes))

	consumer, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err, "cannot create auth consumer")

	session, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err, "cannot create session")

	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err, "cannot create jwt")

	return u, jwt
}

// AuthentifyRequest  have to be used only for tests
func AuthentifyRequest(t *testing.T, req *http.Request, _ *sdk.AuthentifiedUser, jwt string) {
	auth := "Bearer " + jwt
	req.Header.Add("Authorization", auth)
}

func NewAuthentifiedStringRequest(t *testing.T, _ *sdk.AuthentifiedUser, pass, method, uri string, i string) *http.Request {
	req, err := http.NewRequest(method, uri, bytes.NewBuffer([]byte(i)))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	AuthentifyRequest(t, req, nil, pass)
	date := sdk.FormatDateRFC5322(time.Now())
	req.Header.Set("Date", date)
	req.Header.Set("X-CDS-RemoteTime", date)

	return req
}

// NewAuthentifiedRequest prepare a request
func NewAuthentifiedRequest(t *testing.T, _ *sdk.AuthentifiedUser, pass, method, uri string, i interface{}) *http.Request {
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
	AuthentifyRequest(t, req, nil, pass)
	date := sdk.FormatDateRFC5322(time.Now())
	req.Header.Set("Date", date)
	req.Header.Set("X-CDS-RemoteTime", date)

	return req
}

func NewUploadFileRequest(t *testing.T, method string, uri string, body io.Reader, headers map[string]string) *http.Request {
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	date := sdk.FormatDateRFC5322(time.Now())
	req.Header.Set("Date", date)
	req.Header.Set("X-CDS-RemoteTime", date)
	return req
}

func NewRequest(t *testing.T, method, uri string, i interface{}) *http.Request {
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

	date := sdk.FormatDateRFC5322(time.Now())
	req.Header.Set("Date", date)
	req.Header.Set("X-CDS-RemoteTime", date)

	return req
}

// NewJWTAuthentifiedRequest prepare a request
func NewJWTAuthentifiedRequest(t *testing.T, jwt string, method, uri string, i interface{}) *http.Request {
	req := NewRequest(t, method, uri, i)

	auth := "Bearer " + jwt
	req.Header.Add("Authorization", auth)

	return req
}

// NewXSRFJWTAuthentifiedRequest prepare a request
func NewXSRFJWTAuthentifiedRequest(t *testing.T, jwt, xsrf string, method, uri string, i interface{}) *http.Request {
	req := NewRequest(t, method, uri, i)

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

func InsertGroup(t *testing.T, db gorpmapper.SqlExecutorWithTx) *sdk.Group {
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	g1, _ := group.LoadByName(context.TODO(), db, g.Name)
	if g1 != nil {
		models, _ := workermodel.LoadAllByGroupIDs(context.Background(), db, []int64{g.ID}, nil)
		for _, m := range models {
			_ = workermodel.DeleteByID(db, m.ID)
		}

		if err := group.Delete(context.TODO(), db, g1); err != nil {
			t.Logf("unable to delete group: %v", err)
		}
	}

	if err := group.Insert(context.TODO(), db, g); err != nil {
		t.Fatalf("Unable to create group %s", err)
	}

	return g
}

func InsertWorkerModel(t *testing.T, db gorpmapper.SqlExecutorWithTx, name string, groupID int64) *sdk.Model {
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

	if err := workermodel.Insert(context.TODO(), db, &m); err != nil {
		t.Fatalf("Cannot insert worker model: %s", err)
	}

	assert.NotEqual(t, 0, m.ID)
	return &m
}

func InsertHatchery(t *testing.T, db gorpmapper.SqlExecutorWithTx, grp sdk.Group) (*sdk.Service, *rsa.PrivateKey, *sdk.AuthUserConsumer, string) {
	usr1, _ := InsertLambdaUser(t, db, &grp)
	SetUserGroupAdmin(t, db, grp.ID, usr1.ID)

	consumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, usr1.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:     sdk.RandomString(10),
		GroupIDs: []int64{grp.ID},
		Scopes: sdk.NewAuthConsumerScopeDetails(
			sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService, sdk.AuthConsumerScopeWorkerModel),
	}
	hConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, consumer)
	require.NoError(t, err)

	privateKey, err := jws.NewRandomRSAKey()
	require.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	require.NoError(t, err)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:       hConsumer.Name,
			Type:       sdk.TypeHatchery,
			PublicKey:  publicKey,
			ConsumerID: &hConsumer.ID,
		},
	}

	require.NoError(t, services.Insert(context.TODO(), db, &srv))

	session, err := authentication.NewSession(context.TODO(), db, &hConsumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err)

	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	require.NoError(t, authentication.LoadUserConsumerOptions.WithAuthentifiedUser(context.TODO(), db, hConsumer))

	return &srv, privateKey, hConsumer, jwt
}

func InsertService(t *testing.T, db gorpmapper.SqlExecutorWithTx, name, serviceType string, scopes ...sdk.AuthConsumerScope) (*sdk.Service, *rsa.PrivateKey) {
	usr1, _ := InsertAdminUser(t, db)

	consumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, usr1.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	sharedGroup, err := group.LoadByName(context.TODO(), db, sdk.SharedInfraGroupName)
	require.NoError(t, err)
	consumerOptions := builtin.NewConsumerOptions{
		Name:     sdk.RandomString(10),
		GroupIDs: []int64{sharedGroup.ID},
		Scopes:   sdk.NewAuthConsumerScopeDetails(append(scopes, sdk.AuthConsumerScopeProject)...),
	}
	hConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, consumer)
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
			ConsumerID: &hConsumer.ID,
		},
	}

	require.NoError(t, services.Insert(context.TODO(), db, &srv))

	return &srv, privateKey
}

func InitCDNService(t *testing.T, db gorpmapper.SqlExecutorWithTx, scopes ...sdk.AuthConsumerScope) (*sdk.Service, *rsa.PrivateKey, string) {
	usr1, _ := InsertAdminUser(t, db)

	consumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, usr1.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	sharedGroup, err := group.LoadByName(context.TODO(), db, sdk.SharedInfraGroupName)
	require.NoError(t, err)
	consumerOptions := builtin.NewConsumerOptions{
		Name:     sdk.RandomString(10),
		GroupIDs: []int64{sharedGroup.ID},
		Scopes:   sdk.NewAuthConsumerScopeDetails(append(scopes, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService)...),
	}
	hConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, consumer)
	require.NoError(t, err)

	privateKey, err := jws.NewRandomRSAKey()
	require.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	require.NoError(t, err)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:       hConsumer.Name,
			Type:       sdk.TypeCDN,
			PublicKey:  publicKey,
			ConsumerID: &hConsumer.ID,
			Config: map[string]interface{}{
				"public_tcp":            "cdn.net:4545",
				"public_http":           "http://cdn.net:8080",
				"public_tcp_enable_tls": true,
			},
		},
	}

	require.NoError(t, services.Insert(context.TODO(), db, &srv))

	session, err := authentication.NewSession(context.TODO(), db, &hConsumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err)

	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	return &srv, privateKey, jwt
}

func InsertTestWorkflow(t *testing.T, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj *sdk.Project, name string) *sdk.Workflow {
	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	script := GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	require.NoError(t, pipeline.InsertStage(db, s))
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	require.NoError(t, pipeline.InsertJob(db, j, s.ID, &pip))
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	w := sdk.Workflow{
		Name:       name,
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
		MaxRuns: 255,
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, store, *proj, &w))

	return &w
}
