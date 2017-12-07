package api

import (
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

	user.DeleteUserWithDependenciesByName(api.mustDB(), u.Username)

	err = user.InsertUser(api.mustDB(), u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(api.mustDB(), "foo")
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

	user.DeleteUserWithDependenciesByName(api.mustDB(), u.Username)

	err = user.InsertUser(api.mustDB(), u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(api.mustDB(), "foo")
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

	user.DeleteUserWithDependenciesByName(api.mustDB(), u.Username)

	err = user.InsertUser(api.mustDB(), u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(api.mustDB(), "foo")
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

	user.DeleteUserWithDependenciesByName(api.mustDB(), u.Username)

	err = user.InsertUser(api.mustDB(), u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(api.mustDB(), "foo")
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

	user.DeleteUserWithDependenciesByName(api.mustDB(), u.Username)

	err := user.InsertUser(api.mustDB(), u, nil)
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

	project.Delete(api.mustDB(), api.Cache, project1.Key)
	project.Delete(api.mustDB(), api.Cache, project2.Key)

	err = project.Insert(api.mustDB(), api.Cache, project1, u)
	if err != nil {
		t.Fatalf("cannot insert project1: %s", err)
	}
	err = project.Insert(api.mustDB(), api.Cache, project2, u)
	if err != nil {
		t.Fatalf("cannot insert project2: %s", err)
	}

	pipelinePip1 := &sdk.Pipeline{
		Name:      "PIP1",
		ProjectID: project1.ID,
		Type:      sdk.BuildPipeline,
	}

	err = pipeline.InsertPipeline(api.mustDB(), api.Cache, project1, pipelinePip1, nil)
	if err != nil {
		t.Fatalf("cannot insert pipeline: %s", err)
	}

	groupInsert := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	err = group.InsertGroup(api.mustDB(), groupInsert)
	if err != nil {
		t.Fatalf("cannot insert group: %s", err)
	}

	err = group.InsertGroupInProject(api.mustDB(), project1.ID, groupInsert.ID, 4)
	if err != nil {
		t.Fatalf("cannot insert project1 in group: %s", err)
	}
	err = group.InsertGroupInProject(api.mustDB(), project2.ID, groupInsert.ID, 5)
	if err != nil {
		t.Fatalf("cannot insert project1 in group: %s", err)
	}
	err = group.InsertGroupInPipeline(api.mustDB(), pipelinePip1.ID, groupInsert.ID, 7)
	if err != nil {
		t.Fatalf("cannot insert pipeline1 in group: %s", err)
	}

	err = group.InsertUserInGroup(api.mustDB(), groupInsert.ID, u.ID, false)
	if err != nil {
		t.Fatalf("cannot insert user1 in group: %s", err)
	}

	if err := loadUserPermissions(api.mustDB(), api.Cache, u); err != nil {
		t.Fatalf("cannot load user group and project: %s", err)
	}

	if len(u.Groups) != 1 {
		t.Fatalf("Missing/TooMuch group on user, need 1, got %d", len(u.Groups))
	}
	if len(u.Groups[0].ProjectGroups) != 2 {
		t.Fatalf("Missing/TooMuch project on group.Need 2, got %d", len(u.Groups[0].ProjectGroups))
	}
	if len(u.Groups[0].PipelineGroups) != 1 {
		t.Fatalf("Missing/TooMuch pipeline on group.Need 1, got %d", len(u.Groups[0].PipelineGroups))
	}
}

// Test_getUserHandlerOK checks call on /user/{username}
func Test_getUserHandlerOK(t *testing.T) {
	api, _, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	u1, pass1 := assets.InsertLambdaUser(api.mustDB())
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

	u1, pass1 := assets.InsertLambdaUser(api.mustDB())
	assert.NotZero(t, u1)
	assert.NotZero(t, pass1)

	uAdmin, passAdmin := assets.InsertAdminUser(api.mustDB())
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

	u1, pass1 := assets.InsertLambdaUser(api.mustDB())
	assert.NotZero(t, u1)
	assert.NotZero(t, pass1)

	uri := router.GetRoute("GET", api.getUserHandler, map[string]string{"username": u1.Username})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u1, pass1, "GET", uri, nil)

	u2, pass2 := assets.InsertLambdaUser(api.mustDB())
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

	u, pass := assets.InsertLambdaUser(api.mustDB(), g1, g2)
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
