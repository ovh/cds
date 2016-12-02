package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repositoriesmanager/polling"
	"github.com/ovh/cds/engine/api/sessionstore"
	test "github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/sdk"
)

func testfindLinkedProject(t *testing.T, db *sql.DB) (*sdk.Project, *sdk.RepositoriesManager) {
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

	if proj == nil {
		t.Fail()
		return nil, nil
	}

	rm, err := repositoriesmanager.LoadByID(db, rmID)
	if err != nil {
		t.Error(err)
		return nil, nil
	}

	return proj, rm
}

func TestAddPollerOnLinkedApplications(t *testing.T) {
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	query := `
		select 	project.ID, application.ID, repositories_manager.id
		from 	project, application, repositories_manager_project, repositories_manager
		where 	project.id = application.project_id
		and 	project.id = repositories_manager_project.id_project
		and 	repositories_manager.id = repositories_manager_project.id_repositories_manager
		and 	application.repositories_manager_id = repositories_manager_project.id_repositories_manager
		and 	application.repo_fullname is not null
	`

	rows, err := db.Query(query)
	if err != nil {
		t.Error(err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var projectID, applicationID int64
		var rmID int64
		if err := rows.Scan(&projectID, &applicationID, &rmID); err != nil {
			t.Error(err.Error())
			return
		}

		rm, err := repositoriesmanager.LoadByID(db, rmID)
		if err != nil {
			t.Error(err.Error())
			return
		}

		projs, err := project.LoadAllProjects(db)
		if err != nil {
			t.Error(err.Error())
			return
		}
		var proj *sdk.Project
		for _, p := range projs {
			if p.ID == projectID {
				proj = p
				break
			}
		}

		if proj == nil {
			t.Fail()
			return
		}

		pollers, err := poller.LoadPollersByApplication(db, applicationID)
		if err != nil {
			t.Error(err.Error())
			return
		}

		if len(pollers) == 0 {
			app, err := application.LoadApplicationByID(db, applicationID)
			if err != nil {
				t.Error(err.Error())
				return
			}

			pips, err := application.GetAllPipelinesByID(db, applicationID)
			if err != nil {
				t.Error(err.Error())
				return
			}

			if len(pips) == 0 {
				t.Fail()
				return
			}

			p := &sdk.RepositoryPoller{
				Application: *app,
				Pipeline:    pips[0].Pipeline,
				Enabled:     true,
				Name:        rm.Name,
			}

			if err := poller.InsertPoller(db, p); err != nil {
				t.Error(err)
				return
			}
		}

		if rm.PollingSupported {
			c1 := make(chan bool, 1)
			go func() {
				time.Sleep(time.Second * 120)
				c1 <- true
			}()

			t.Logf("Testing poller on %s", rm.Name)
			w := polling.NewWorker(proj.Key)
			polling.RunningPollers.Workers[proj.Key] = w
			_, quit, err := w.Poll()
			if err != nil {
				t.Error(err)
			}
			assert.NoError(t, err)
			select {
			case <-quit:
				t.Logf("Polling is over")
				t.Fail()
				return
			case <-c1:
				return
			}
		}
	}

}

func TestAddPollerHandler(t *testing.T) {
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	router = &Router{authDriver, mux.NewRouter(), "/TestAddPollerHandler"}
	router.init()

	//1. Create admin user
	u, pass, err := test.InsertAdminUser(t, db)
	assert.NoError(t, err)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	assert.NotNil(t, proj)
	if proj == nil {
		t.Fail()
		return
	}

	//3. Create Pipeline
	pipelineKey := test.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

	//4. Insert Application
	appName := test.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	err = application.InsertApplication(db, proj, app)

	//5. Attach pipeline to application
	err = application.AttachPipeline(db, app.ID, pip.ID)
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.RepositoryPoller{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.Enabled)
	assert.NotZero(t, popol2.Name)
}

func TestUpdatePollerHandler(t *testing.T) {
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	router = &Router{authDriver, mux.NewRouter(), "/TestUpdatePollerHandler"}
	router.init()

	//1. Crerouter.ate admin user
	u, pass, err := test.InsertAdminUser(t, db)
	assert.NoError(t, err)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	assert.NotNil(t, proj)
	if proj == nil {
		t.Fail()
		return
	}

	//3. Create Pipeline
	pipelineKey := test.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

	//4. Insert Application
	appName := test.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	err = application.InsertApplication(db, proj, app)

	//5. Attach pipeline to application
	err = application.AttachPipeline(db, app.ID, pip.ID)
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.RepositoryPoller{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.Enabled)
	assert.NotZero(t, popol2.Name)

	//9. Update the poller
	popol2.Enabled = false
	jsonBody, _ = json.Marshal(popol2)
	body = bytes.NewBuffer(jsonBody)

	uri = router.getRoute("PUT", updatePollerHandler, vars)
	if uri == "" {
		t.Fail()
		return
	}
	req, _ = http.NewRequest("PUT", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ = ioutil.ReadAll(w.Body)
	popol3 := &sdk.RepositoryPoller{}
	json.Unmarshal(res, &popol3)
	assert.NotEqual(t, popol.Enabled, popol3.Enabled)
	assert.NotZero(t, popol3.Name)
}

func TestGetApplicationPollersHandler(t *testing.T) {
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	router = &Router{authDriver, mux.NewRouter(), "/TestGetApplicationPollersHandler"}
	router.init()

	//1. Create admin user
	u, pass, err := test.InsertAdminUser(t, db)
	assert.NoError(t, err)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	assert.NotNil(t, proj)
	if proj == nil {
		t.Fail()
		return
	}

	//3. Create Pipeline
	pipelineKey := test.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

	//4. Insert Application
	appName := test.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	err = application.InsertApplication(db, proj, app)

	//5. Attach pipeline to application
	err = application.AttachPipeline(db, app.ID, pip.ID)
	assert.NoError(t, err)
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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.RepositoryPoller{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.Enabled)
	assert.NotZero(t, popol2.Name)

	t.Logf("Poller : %s", string(res))

	//9. Load the pollers
	uri = router.getRoute("GET", getApplicationPollersHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ = ioutil.ReadAll(w.Body)
	pollers := []sdk.RepositoryPoller{}
	json.Unmarshal(res, &pollers)

	assert.Equal(t, 1, len(pollers))
	assert.Equal(t, popol.Name, pollers[0].Name)
	assert.Equal(t, popol.Enabled, pollers[0].Enabled)
	assert.Equal(t, popol.Application.Name, pollers[0].Application.Name)
	assert.Equal(t, popol.Pipeline.Name, pollers[0].Pipeline.Name)
}

func TestGetPollersHandler(t *testing.T) {
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	router = &Router{authDriver, mux.NewRouter(), "/TestGetPollersHandler"}
	router.init()

	//1. Create admin user
	u, pass, err := test.InsertAdminUser(t, db)
	assert.NoError(t, err)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	assert.NotNil(t, proj)
	if proj == nil {
		t.Fail()
		return
	}

	//3. Create Pipeline
	pipelineKey := test.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

	//4. Insert Application
	appName := test.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	err = application.InsertApplication(db, proj, app)

	//5. Attach pipeline to application
	err = application.AttachPipeline(db, app.ID, pip.ID)
	assert.NoError(t, err)
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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.RepositoryPoller{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.Enabled)
	assert.NotZero(t, popol2.Name)

	t.Logf("Poller : %s", string(res))

	//9. Load the pollers
	uri = router.getRoute("GET", getPollersHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.AuthentifyRequest(t, req, u, pass)

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
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	router = &Router{authDriver, mux.NewRouter(), "/TestGetPollersHandler"}
	router.init()

	//1. Create admin user
	u, pass, err := test.InsertAdminUser(t, db)
	assert.NoError(t, err)

	//2. Create project
	proj, rm := testfindLinkedProject(t, db)
	assert.NotNil(t, proj)
	if proj == nil {
		t.Fail()
		return
	}

	//3. Create Pipeline
	pipelineKey := test.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

	//4. Insert Application
	appName := test.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	err = application.InsertApplication(db, proj, app)

	//5. Attach pipeline to application
	err = application.AttachPipeline(db, app.ID, pip.ID)
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	popol2 := &sdk.RepositoryPoller{}
	json.Unmarshal(res, &popol2)
	assert.Equal(t, popol.Enabled, popol2.Enabled)
	assert.NotZero(t, popol2.Name)

	t.Logf("Poller : %s", string(res))

	//9. Load the pollers
	uri = router.getRoute("DELETE", deletePollerHandler, vars)
	req, err = http.NewRequest("DELETE", uri, nil)
	test.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	//9. Load the pollers
	uri = router.getRoute("GET", getApplicationPollersHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ = ioutil.ReadAll(w.Body)
	pollers := []sdk.RepositoryPoller{}
	json.Unmarshal(res, &pollers)

	assert.Equal(t, 0, len(pollers))
}
