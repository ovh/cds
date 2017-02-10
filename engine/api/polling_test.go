package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func testfindLinkedProject(t *testing.T, db gorp.SqlExecutor) (*sdk.Project, *sdk.RepositoriesManager) {
	query := `
		select 	project.ID, repositories_manager_project.id_repositories_manager
		from 	project, repositories_manager_project
		where 	project.id = repositories_manager_project.id_project
		limit 	1
		`
	var projectID, rmID int64
	err := db.QueryRow(query).Scan(&projectID, &rmID)
	if err != nil {
		t.Skip("Cant find any project linked to a repository. Skipping this tests.")
		return nil, nil
	}

	projs, err := project.LoadAllProjects(db)
	if err != nil {
		t.Error(err.Error())
		return nil, nil
	}
	var proj *sdk.Project
	for _, p := range projs {
		if p.ID == projectID {
			proj = p
			break
		}
	}

	rm, err := repositoriesmanager.LoadByID(db, rmID)
	if err != nil {
		t.Error(err)
		return nil, nil
	}

	return proj, rm
}

func TestAddPollerHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestAddPollerHandler"}
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := assets.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	//4. Insert Application
	appName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.InsertApplication(db, proj, app))

	//5. Attach pipeline to application
	test.NoError(t, application.AttachPipeline(db, app.ID, pip.ID))

	app.RepositoriesManager = rm
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(db, app, proj.Key)
	//6. Prepare a poller
	popol := &sdk.RepositoryPoller{
		Application: *app,
		Pipeline:    *pip,
		Enabled:     true,
		Name:        "github",
	}

	//7. Prepare the request
	jsonBody, _ := json.Marshal(popol)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}
	uri := router.getRoute("POST", addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.Application{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol2.RepositoryPollers[0].Name)
}

func TestUpdatePollerHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestUpdatePollerHandler"}
	router.init()

	//1. Crerouter.ate admin user
	u, pass := assets.InsertAdminUser(t, db)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := assets.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	//4. Insert Application
	appName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.InsertApplication(db, proj, app))

	//5. Attach pipeline to application
	test.NoError(t, application.AttachPipeline(db, app.ID, pip.ID))

	app.RepositoriesManager = rm
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(db, app, proj.Key)
	//6. Prepare a poller
	popol := &sdk.RepositoryPoller{
		Application: *app,
		Pipeline:    *pip,
		Enabled:     true,
		Name:        "github",
	}

	//7. Prepare the request
	jsonBody, _ := json.Marshal(popol)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}
	uri := router.getRoute("POST", addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.Application{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol2.RepositoryPollers[0].Name)

	//9. Update the poller
	popol2.RepositoryPollers[0].Enabled = false
	jsonBody, _ = json.Marshal(popol2.RepositoryPollers[0])
	body = bytes.NewBuffer(jsonBody)

	uri = router.getRoute("PUT", updatePollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ = http.NewRequest("PUT", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ = ioutil.ReadAll(w.Body)
	popol3 := &sdk.Application{}
	json.Unmarshal(res, &popol3)
	assert.NotEqual(t, popol.Enabled, popol3.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol3.RepositoryPollers[0].Name)
}

func TestGetApplicationPollersHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestGetApplicationPollersHandler"}
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := assets.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	//4. Insert Application
	appName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.InsertApplication(db, proj, app))

	//5. Attach pipeline to application
	test.NoError(t, application.AttachPipeline(db, app.ID, pip.ID))

	app.RepositoriesManager = rm
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(db, app, proj.Key)
	//6. Prepare a poller
	popol := &sdk.RepositoryPoller{
		Application: *app,
		Pipeline:    *pip,
		Enabled:     true,
		Name:        "github",
	}

	//7. Prepare the request
	jsonBody, _ := json.Marshal(popol)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}
	uri := router.getRoute("POST", addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	t.Logf("%s", res)
	popol2 := &sdk.Application{}
	test.NoError(t, json.Unmarshal(res, &popol2))
	assert.Equal(t, popol.Enabled, popol2.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol2.RepositoryPollers[0].Name)

	t.Logf("Poller : %s", string(res))

	//9. Load the pollers
	uri = router.getRoute("GET", getApplicationPollersHandler, vars)
	req, _ = http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ = ioutil.ReadAll(w.Body)
	t.Logf("%s", res)

	a := []sdk.RepositoryPoller{}
	test.NoError(t, json.Unmarshal(res, &a))

	assert.Equal(t, 1, len(a))
	assert.Equal(t, popol.Name, a[0].Name)
	assert.Equal(t, popol.Enabled, a[0].Enabled)
	assert.Equal(t, popol.Application.Name, a[0].Application.Name)
	assert.Equal(t, popol.Pipeline.Name, a[0].Pipeline.Name)
}

func TestGetPollersHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestGetPollersHandler"}
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := assets.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	//4. Insert Application
	appName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.InsertApplication(db, proj, app))

	//5. Attach pipeline to application
	test.NoError(t, application.AttachPipeline(db, app.ID, pip.ID))

	app.RepositoriesManager = rm
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(db, app, proj.Key)
	//6. Prepare a poller
	popol := &sdk.RepositoryPoller{
		Application: *app,
		Pipeline:    *pip,
		Enabled:     true,
		Name:        "github",
	}

	//7. Prepare the request
	jsonBody, _ := json.Marshal(popol)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}
	uri := router.getRoute("POST", addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.Application{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol2.RepositoryPollers[0].Name)

	t.Logf("Poller : %s", string(res))

	//9. Load the pollers
	uri = router.getRoute("GET", getPollersHandler, vars)
	req, _ = http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ = ioutil.ReadAll(w.Body)
	pollers := sdk.RepositoryPoller{}
	json.Unmarshal(res, &pollers)

	assert.Equal(t, popol.Name, pollers.Name)
	assert.Equal(t, popol.Enabled, pollers.Enabled)
	assert.Equal(t, popol.Application.Name, pollers.Application.Name)
	assert.Equal(t, popol.Pipeline.Name, pollers.Pipeline.Name)
}

func TestDeletePollerHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestGetPollersHandler"}
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := assets.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	//4. Insert Application
	appName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.InsertApplication(db, proj, app))

	//5. Attach pipeline to application
	test.NoError(t, application.AttachPipeline(db, app.ID, pip.ID))

	app.RepositoriesManager = rm
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(db, app, proj.Key)

	//6. Prepare a poller
	popol := &sdk.RepositoryPoller{
		Application: *app,
		Pipeline:    *pip,
		Enabled:     true,
		Name:        "github",
	}

	//7. Prepare the request
	jsonBody, _ := json.Marshal(popol)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}
	uri := router.getRoute("POST", addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.Application{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol2.RepositoryPollers[0].Name)

	t.Logf("Poller : %s", string(res))

	//9. Load the pollers
	uri = router.getRoute("DELETE", deletePollerHandler, vars)
	req, _ = http.NewRequest("DELETE", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	//9. Load the pollers
	uri = router.getRoute("GET", getApplicationPollersHandler, vars)
	req, _ = http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ = ioutil.ReadAll(w.Body)
	pollers := []sdk.RepositoryPoller{}
	json.Unmarshal(res, &pollers)

	assert.Equal(t, 0, len(pollers))
}
