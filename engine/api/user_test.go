package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// TestVerifyUserToken test token verification when OK
func TestVerifyUserToken(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := &sdk.User{
		Username: "foo",
		Email:    "foo.bar@ovh.com",
		Fullname: "foo bar",
	}

	token, hashedToken, err := user.GeneratePassword()
	if err != nil {
		t.Fatalf("Cannot create token: %s", err)
	}

	a := &sdk.Auth{
		HashedTokenVerify: hashedToken,
	}

	user.DeleteUserWithDependenciesByName(api.mustDB(context.Background()), u.Username)

	err = user.InsertUser(api.mustDB(context.Background()), u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(api.mustDB(context.Background()), "foo")
	if err != nil {
		t.Fatalf("Cannot load %s: %s\n", "foo", err)
	}

	password, hashedpassword, err := user.Verify(u2, token)
	if err != nil {
		t.Fatalf("User shoud be verified : %s", err)
	}
	if password == "" {
		t.Fatalf("Password should not be empty")
	}
	if hashedpassword == "" {
		t.Fatalf("hashedPassword should not be empty")
	}
}

// TestWrongTokenUser  test token verification when token is wrong
func TestWrongTokenUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := &sdk.User{
		Username: "foo",
		Email:    "foo.bar@ovh.com",
		Fullname: "foo bar",
	}

	_, hashedToken, err := user.GeneratePassword()
	if err != nil {
		t.Fatalf("Cannot create token: %s", err)
	}

	a := &sdk.Auth{
		HashedTokenVerify: hashedToken,
	}

	user.DeleteUserWithDependenciesByName(api.mustDB(context.Background()), u.Username)

	err = user.InsertUser(api.mustDB(context.Background()), u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(api.mustDB(context.Background()), "foo")
	if err != nil {
		t.Fatalf("Cannot load %s: %s\n", "foo", err)
	}

	_, _, err = user.Verify(u2, "blabla")
	if err == nil {
		t.Fatalf("User shoud not be verified : %s", err)
	}
}

// TestVerifyResetExpired test validating reset token when time expired
func TestVerifyResetExpired(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := &sdk.User{
		Username: "foo",
		Email:    "foo.bar@ovh.com",
		Fullname: "foo bar",
	}

	token, hashedToken, err := user.GeneratePassword()
	if err != nil {
		t.Fatalf("Cannot create token: %s", err)
	}

	a := &sdk.Auth{
		HashedTokenVerify: hashedToken,
		DateReset:         1,
		EmailVerified:     true,
	}

	user.DeleteUserWithDependenciesByName(api.mustDB(context.Background()), u.Username)

	err = user.InsertUser(api.mustDB(context.Background()), u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(api.mustDB(context.Background()), "foo")
	if err != nil {
		t.Fatalf("Cannot load %s: %s\n", "foo", err)
	}

	_, _, err = user.Verify(u2, token)
	if err == nil {
		t.Fatalf("User shoud not be verified : %s", err)
	}
	if err.Error() != "Reset operation expired" {
		t.Fatalf("Reset sould be expired")
	}
}

// TestVerifyAlreadyDone test token verification when it's already done
func TestVerifyAlreadyDone(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := &sdk.User{
		Username: "foo",
		Email:    "foo.bar@ovh.com",
		Fullname: "foo bar",
	}

	token, hashedToken, err := user.GeneratePassword()
	if err != nil {
		t.Fatalf("Cannot create token: %s", err)
	}

	a := &sdk.Auth{
		HashedTokenVerify: hashedToken,
		DateReset:         0,
		EmailVerified:     true,
	}

	user.DeleteUserWithDependenciesByName(api.mustDB(context.Background()), u.Username)

	err = user.InsertUser(api.mustDB(context.Background()), u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(api.mustDB(context.Background()), "foo")
	if err != nil {
		t.Fatalf("Cannot load %s: %s\n", "foo", err)
	}

	_, _, err = user.Verify(u2, token)
	if err == nil {
		t.Fatalf("User shoud not be verified : %s", err)
	}
	if err.Error() != "Account already verified" {
		t.Fatalf("Account should be already verified")
	}
}

// TestVerifyAlreadyDone test token verification when it's already done
func TestLoadUserWithGroup(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := &sdk.User{
		Username: "foo",
		Email:    "foo.bar@ovh.com",
		Fullname: "foo bar",
	}

	user.DeleteUserWithDependenciesByName(api.mustDB(context.Background()), u.Username)

	err := user.InsertUser(api.mustDB(context.Background()), u, nil)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	project1 := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: "foo",
	}
	project2 := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: "bar",
	}

	project.Delete(api.mustDB(context.Background()), api.Cache, project1.Key)
	project.Delete(api.mustDB(context.Background()), api.Cache, project2.Key)

	err = project.Insert(api.mustDB(context.Background()), api.Cache, project1, u)
	if err != nil {
		t.Fatalf("cannot insert project1: %s", err)
	}
	err = project.Insert(api.mustDB(context.Background()), api.Cache, project2, u)
	if err != nil {
		t.Fatalf("cannot insert project2: %s", err)
	}

	pipelinePip1 := &sdk.Pipeline{
		Name:      "PIP1",
		ProjectID: project1.ID,
		Type:      sdk.BuildPipeline,
	}

	err = pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, project1, pipelinePip1, nil)
	if err != nil {
		t.Fatalf("cannot insert pipeline: %s", err)
	}

	groupInsert := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	err = group.InsertGroup(api.mustDB(context.Background()), groupInsert)
	if err != nil {
		t.Fatalf("cannot insert group: %s", err)
	}

	err = group.InsertGroupInProject(api.mustDB(context.Background()), project1.ID, groupInsert.ID, 4)
	if err != nil {
		t.Fatalf("cannot insert project1 in group: %s", err)
	}
	err = group.InsertGroupInProject(api.mustDB(context.Background()), project2.ID, groupInsert.ID, 5)
	if err != nil {
		t.Fatalf("cannot insert project1 in group: %s", err)
	}
	err = group.InsertGroupInPipeline(api.mustDB(context.Background()), pipelinePip1.ID, groupInsert.ID, 7)
	if err != nil {
		t.Fatalf("cannot insert pipeline1 in group: %s", err)
	}

	err = group.InsertUserInGroup(api.mustDB(context.Background()), groupInsert.ID, u.ID, false)
	if err != nil {
		t.Fatalf("cannot insert user1 in group: %s", err)
	}

	if err := loadUserPermissions(api.mustDB(context.Background()), api.Cache, u); err != nil {
		t.Fatalf("cannot load user group and project: %s", err)
	}

	if len(u.Permissions.ProjectsPerm) != 2 {
		t.Fatalf("Missing/TooMuch project on u.Permissions.ProjectsPerm 2, got %d", len(u.Permissions.ProjectsPerm))
	}
	if len(u.Permissions.PipelinesPerm) != 1 {
		t.Fatalf("Missing/TooMuch pipeline on u.Permissions.PipelinesPerm 1, got %d", len(u.Permissions.PipelinesPerm))
	}
}

// Test_getUserHandlerOK checks call on /user/{username}
func Test_getUserHandlerOK(t *testing.T) {
	api, _, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	u1, pass1 := assets.InsertLambdaUser(api.mustDB(context.Background()))
	assert.NotZero(t, u1)
	assert.NotZero(t, pass1)

	uri := api.Router.GetRoute("GET", api.getUserHandler, map[string]string{"username": u1.Username})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u1, pass1, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	res := sdk.User{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
}

// Test_getUserHandlerOK checks call on /user/{username} with an admin user
func Test_getUserHandlerAdmin(t *testing.T) {
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	u1, pass1 := assets.InsertLambdaUser(api.mustDB(context.Background()))
	assert.NotZero(t, u1)
	assert.NotZero(t, pass1)

	uAdmin, passAdmin := assets.InsertAdminUser(api.mustDB(context.Background()))
	assert.NotZero(t, uAdmin)
	assert.NotZero(t, passAdmin)

	uri := router.GetRoute("GET", api.getUserHandler, map[string]string{"username": u1.Username})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, uAdmin, passAdmin, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	res := sdk.User{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
}

// Test_getUserHandlerOK checks call on /user/{username} with a not allowed user
func Test_getUserHandlerForbidden(t *testing.T) {
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	u1, pass1 := assets.InsertLambdaUser(api.mustDB(context.Background()))
	assert.NotZero(t, u1)
	assert.NotZero(t, pass1)

	uri := router.GetRoute("GET", api.getUserHandler, map[string]string{"username": u1.Username})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u1, pass1, "GET", uri, nil)

	u2, pass2 := assets.InsertLambdaUser(api.mustDB(context.Background()))
	assert.NotZero(t, u1)
	assert.NotZero(t, pass2)

	req2 := assets.NewAuthentifiedRequest(t, u2, pass2, "GET", uri, nil)

	//Do the request
	w1 := httptest.NewRecorder()
	router.Mux.ServeHTTP(w1, req)
	assert.Equal(t, 200, w1.Code)

	res := sdk.User{}
	if err := json.Unmarshal(w1.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}

	// user 2 try to call user 1 -> this is forbidden
	w2 := httptest.NewRecorder()
	router.Mux.ServeHTTP(w2, req2)
	assert.Equal(t, 403, w2.Code)

	t.Logf("Body: %s", w2.Body.String())
}

func Test_getUserGroupsHandler(t *testing.T) {
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	g1 := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	g2 := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	u, pass := assets.InsertLambdaUser(api.mustDB(context.Background()), g1, g2)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	uri := router.GetRoute("GET", api.getUserGroupsHandler, map[string]string{"username": u.Username})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	res := map[string][]sdk.Group{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(res["groups"]))
	assert.Equal(t, 0, len(res["groups_admin"]))

}

// Test_getUserTokenListHandlerOK checks call on /user/tokens
func Test_getUserTokenListHandlerOK(t *testing.T) {
	api, _, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	u1, pass1 := assets.InsertAdminUser(api.mustDB(context.Background()))
	assert.NotZero(t, u1)
	assert.NotZero(t, pass1)

	gr := &sdk.Group{
		Admins: []sdk.User{*u1},
		Name:   "testGroup" + sdk.RandomString(10),
	}
	grID, _, err := group.AddGroup(api.mustDB(context.Background()), gr)
	assert.NoError(t, err)

	err = group.InsertUserInGroup(api.mustDB(context.Background()), grID, u1.ID, true)
	assert.NoError(t, err)

	uriGenerateToken := api.Router.GetRoute("POST", api.generateTokenHandler, map[string]string{"permGroupName": gr.Name})
	params := struct {
		Expiration  string `json:"expiration"`
		Description string `json:"description"`
	}{Expiration: "persistent", Description: "this is a test token"}
	reqGenerateToken := assets.NewAuthentifiedRequest(t, u1, pass1, "POST", uriGenerateToken, params)

	//Do the request to generate token
	wGen := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGen, reqGenerateToken)
	assert.Equal(t, 200, wGen.Code)

	resGen := sdk.Token{}
	if err := json.Unmarshal(wGen.Body.Bytes(), &resGen); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, params.Description, resGen.Description)
	assert.Equal(t, gr.Name, resGen.GroupName)
	assert.NotZero(t, resGen.Token)

	uri := api.Router.GetRoute("GET", api.getUserTokenListHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u1, pass1, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	res := []sdk.Token{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}

	found := false
	for _, tok := range res {
		if tok.Description == params.Description && gr.Name == tok.GroupName {
			found = true
			break
		}
	}

	assert.Equal(t, true, found, "Token created is not in the list")
}

// Test_postUserFavoriteHandler
func Test_postUserFavoriteHandler(t *testing.T) {
	api, _, _ := newTestAPI(t, bootstrap.InitiliazeDB)
	db := api.mustDB(context.Background())
	u1, pass1 := assets.InsertAdminUser(api.mustDB(context.Background()))
	assert.NotZero(t, u1)
	assert.NotZero(t, pass1)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u1)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, nil))

	proj, _ = project.LoadByID(db, api.Cache, proj.ID, u1, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	wf := sdk.Workflow{
		Name:       "wf_test1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
	}

	test.NoError(t, workflow.Insert(db, api.Cache, &wf, proj, u1))

	uri := api.Router.GetRoute("POST", api.postUserFavoriteHandler, nil)
	test.NotEmpty(t, uri)

	params := sdk.FavoriteParams{
		Type:         "workflow",
		ProjectKey:   proj.Key,
		WorkflowName: wf.Name,
	}
	req := assets.NewAuthentifiedRequest(t, u1, pass1, "POST", uri, params)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	res := sdk.Workflow{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}

	test.Equal(t, res.Favorite, true)

	uri2 := api.Router.GetRoute("POST", api.postUserFavoriteHandler, nil)
	test.NotEmpty(t, uri2)

	req2 := assets.NewAuthentifiedRequest(t, u1, pass1, "POST", uri2, params)

	//Do the request
	w2 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w2, req2)
	assert.Equal(t, 200, w2.Code)

	t.Logf("Body: %s", w2.Body.String())

	res2 := sdk.Workflow{}
	if err := json.Unmarshal(w2.Body.Bytes(), &res2); err != nil {
		t.Fatal(err)
	}
	test.Equal(t, res2.Favorite, false)

	uri3 := api.Router.GetRoute("POST", api.postUserFavoriteHandler, nil)
	test.NotEmpty(t, uri3)

	paramsProj := sdk.FavoriteParams{
		Type:       "project",
		ProjectKey: proj.Key,
	}
	req3 := assets.NewAuthentifiedRequest(t, u1, pass1, "POST", uri3, paramsProj)

	//Do the request
	w3 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w3, req3)
	assert.Equal(t, 200, w3.Code)

	t.Logf("Body: %s", w3.Body.String())

	res3 := sdk.Project{}
	if err := json.Unmarshal(w3.Body.Bytes(), &res3); err != nil {
		t.Fatal(err)
	}
	test.Equal(t, res3.Favorite, true)
}
