package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/proullon/ramsql/engine/log"
	"github.com/stretchr/testify/assert"

	"encoding/json"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

// TestVerifyUserToken test token verification when OK
func TestVerifyUserToken(t *testing.T) {
	log.UseTestLogger(t)
	db := test.Setup("TestVerifyUserToken", t)

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

	err = user.InsertUser(db, u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(db, "foo")
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

// TestWrongTokenUser  test token verificaiton when token is wrong
func TestWrongTokenUser(t *testing.T) {
	log.UseTestLogger(t)
	db := test.Setup("TestWrongTokenUser", t)

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

	err = user.InsertUser(db, u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(db, "foo")
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
	log.UseTestLogger(t)
	db := test.Setup("TestVerifyResetExpired", t)

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

	err = user.InsertUser(db, u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(db, "foo")
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
	log.UseTestLogger(t)
	db := test.Setup("TestVerifyAlreadyDone", t)

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

	err = user.InsertUser(db, u, a)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	u2, err := user.LoadUserAndAuth(db, "foo")
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
	log.UseTestLogger(t)
	db := test.Setup("TestLoadUserWithGroup", t)

	u := &sdk.User{
		Username: "foo",
		Email:    "foo.bar@ovh.com",
		Fullname: "foo bar",
	}

	err := user.InsertUser(db, u, nil)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}

	project1 := &sdk.Project{
		Key:  "foo",
		Name: "foo",
	}
	project2 := &sdk.Project{
		Key:  "bar",
		Name: "bar",
	}

	err = project.InsertProject(db, project1)
	if err != nil {
		t.Fatalf("cannot insert project1: %s", err)
	}
	err = project.InsertProject(db, project2)
	if err != nil {
		t.Fatalf("cannot insert project2: %s", err)
	}

	pipelinePip1 := &sdk.Pipeline{
		Name:      "PIP1",
		ProjectID: project1.ID,
		Type:      sdk.BuildPipeline,
	}

	err = pipeline.InsertPipeline(db, pipelinePip1)
	if err != nil {
		t.Fatalf("cannot insert pipeline: %s", err)
	}

	groupInsert := &sdk.Group{
		Name: "GroupeFoo",
	}

	err = group.InsertGroup(db, groupInsert)
	if err != nil {
		t.Fatalf("cannot insert group: %s", err)
	}

	err = group.InsertGroupInProject(db, project1.ID, groupInsert.ID, 4)
	if err != nil {
		t.Fatalf("cannot insert project1 in group: %s", err)
	}
	err = group.InsertGroupInProject(db, project2.ID, groupInsert.ID, 5)
	if err != nil {
		t.Fatalf("cannot insert project1 in group: %s", err)
	}
	err = group.InsertGroupInPipeline(db, pipelinePip1.ID, groupInsert.ID, 7)
	if err != nil {
		t.Fatalf("cannot insert pipeline1 in group: %s", err)
	}

	err = group.InsertUserInGroup(db, groupInsert.ID, u.ID, false)
	if err != nil {
		t.Fatalf("cannot insert user1 in group: %s", err)
	}

	err = user.LoadUserPermissions(db, u)
	if err != nil {
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

func Test_getUserGroupsHandler(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_getUserGroupsHandler"}
	router.init()

	g1 := &sdk.Group{
		Name: testwithdb.RandomString(t, 10),
	}

	g2 := &sdk.Group{
		Name: testwithdb.RandomString(t, 10),
	}

	u, pass, err := testwithdb.InsertLambaUser(t, db, g1, g2)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	uri := router.getRoute("GET", getUserGroupsHandler, map[string]string{"name": u.Username})
	if uri == "" {
		t.Fatal("Route not found")
	}

	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	res := map[string][]sdk.Group{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(res["groups"]))
	assert.Equal(t, 0, len(res["groups_admin"]))

}
