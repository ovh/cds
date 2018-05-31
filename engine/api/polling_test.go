package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func testfindLinkedProject(t *testing.T, db gorp.SqlExecutor, store cache.Store) *sdk.Project {
	query := `
		select 	project.ID
		from 	project
		where 	vcs_servers is not null
		limit 	1
		`
	var projectID, rmID int64
	err := db.QueryRow(query).Scan(&projectID, &rmID)
	if err != nil {
		t.Skip("Cant find any project linked to a repository. Skipping this tests.")
		return nil
	}

	projs, err := project.LoadAll(nil, db, store, nil)
	if err != nil {
		t.Error(err.Error())
		return nil
	}
	var proj *sdk.Project
	for _, p := range projs {
		if p.ID == projectID {
			proj = &p
			break
		}
	}

	return proj
}

func TestAddPollerHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := testfindLinkedProject(t, db, api.Cache)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil))

	//4. Insert Application
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, nil))

	//5. Attach pipeline to application
	_, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID)
	test.NoError(t, err)

	app.VCSServer = proj.VCSServers[0].Name
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(api.mustDB(), app, proj.Key)
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
	uri := router.GetRoute("POST", api.addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.Application{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol2.RepositoryPollers[0].Name)
}

func TestUpdatePollerHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Crerouter.ate admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := testfindLinkedProject(t, db, api.Cache)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil))

	//4. Insert Application
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, nil))

	//5. Attach pipeline to application
	_, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID)
	test.NoError(t, err)

	app.VCSServer = proj.VCSServers[0].Name
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(api.mustDB(), app, proj.Key)
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
	uri := router.GetRoute("POST", api.addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

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

	uri = router.GetRoute("PUT", api.updatePollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ = http.NewRequest("PUT", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ = ioutil.ReadAll(w.Body)
	popol3 := &sdk.Application{}
	json.Unmarshal(res, &popol3)
	assert.NotEqual(t, popol.Enabled, popol3.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol3.RepositoryPollers[0].Name)
}

func TestGetApplicationPollersHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := testfindLinkedProject(t, db, api.Cache)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil))

	//4. Insert Application
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, nil))

	//5. Attach pipeline to application
	_, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID)
	test.NoError(t, err)

	app.VCSServer = proj.VCSServers[0].Name
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(api.mustDB(), app, proj.Key)
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
	uri := router.GetRoute("POST", api.addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	t.Logf("%s", res)
	popol2 := &sdk.Application{}
	test.NoError(t, json.Unmarshal(res, &popol2))
	assert.Equal(t, popol.Enabled, popol2.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol2.RepositoryPollers[0].Name)

	t.Logf("Poller : %s", string(res))

	//9. Load the pollers
	uri = router.GetRoute("GET", api.getApplicationPollersHandler, vars)
	req, _ = http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

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
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := testfindLinkedProject(t, db, api.Cache)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil))

	//4. Insert Application
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, nil))

	//5. Attach pipeline to application
	_, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID)
	test.NoError(t, err)

	app.VCSServer = proj.VCSServers[0].Name
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(api.mustDB(), app, proj.Key)
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
	uri := router.GetRoute("POST", api.addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.Application{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol2.RepositoryPollers[0].Name)

	t.Logf("Poller : %s", string(res))

	//9. Load the pollers
	uri = router.GetRoute("GET", api.getPollersHandler, vars)
	req, _ = http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

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
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := testfindLinkedProject(t, db, api.Cache)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil))

	//4. Insert Application
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, nil))

	//5. Attach pipeline to application
	_, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID)
	test.NoError(t, err)

	app.VCSServer = proj.VCSServers[0].Name
	app.RepositoryFullname = "test/" + app.Name
	repositoriesmanager.InsertForApplication(api.mustDB(), app, proj.Key)

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
	uri := router.GetRoute("POST", api.addPollerHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.Application{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.RepositoryPollers[0].Enabled)
	assert.NotZero(t, popol2.RepositoryPollers[0].Name)

	t.Logf("Poller : %s", string(res))

	//9. Load the pollers
	uri = router.GetRoute("DELETE", api.deletePollerHandler, vars)
	req, _ = http.NewRequest("DELETE", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	//9. Load the pollers
	uri = router.GetRoute("GET", api.getApplicationPollersHandler, vars)
	req, _ = http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ = ioutil.ReadAll(w.Body)
	pollers := []sdk.RepositoryPoller{}
	json.Unmarshal(res, &pollers)

	assert.Equal(t, 0, len(pollers))
}
