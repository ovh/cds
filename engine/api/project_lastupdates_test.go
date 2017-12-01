package api

/*
func deleteUser(t *testing.T, db gorp.SqlExecutor, u *sdk.User, g *sdk.Group) error {
	var err error
	u, err = user.LoadUserWithoutAuth(api.mustDB(), u.Username)
	if err != nil {
		return err
	}
	g, err = group.LoadGroup(api.mustDB(), g.Name)
	if err != nil {
		return err
	}
	t.Logf("Delete user %s(%d)\n", u.Username, u.ID)
	if err := user.DeleteUserWithDependencies(api.mustDB(), u); err != nil {
		return err
	}
	t.Logf("Delete group %s(%d)\n", g.Name, g.ID)
	if err := group.DeleteGroupAndDependencies(api.mustDB(), g); err != nil {
		return err
	}
	return nil
}

func Test_getUserLastUpdatesShouldReturns1Project1App1Pipeline(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create a user
	u := sdk.NewUser("testuser")
	u.Admin = false
	u.Email = "mail@mail.com"
	u.Origin = "local"
	u.Auth = sdk.Auth{
		EmailVerified: true,
	}

	//Create a group
	g := &sdk.Group{Name: "testgroup"}

	//Delete user and group
	deleteUser(t, db, u, g)
	//All the project
	deleteAll(t, api, "TEST_LAST_UPDATE")

	//Create a user
	t.Logf("Insert User %s", u.Username)
	test.NoError(t, user.InsertUser(api.mustDB(), u, &u.Auth))

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TEST_LAST_UPDATE", "TEST_LAST_UPDATE", u)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip, u))

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	test.NoError(t, application.Insert(api.mustDB(), api.Cache,  proj, app, u))

	//Create a group
	t.Logf("Insert Group %s", g.Name)
	test.NoError(t, group.InsertGroup(api.mustDB(), g))

	//Add user in group
	test.NoError(t, group.InsertUserInGroup(api.mustDB(), g.ID, u.ID, true))

	//All associations
	test.NoError(t, group.InsertGroupInProject(api.mustDB(), proj.ID, g.ID, 4))
	test.NoError(t, group.InsertGroupInApplication(api.mustDB(), app.ID, g.ID, 4))
	test.NoError(t, group.InsertGroupInPipeline(api.mustDB(), pip.ID, g.ID, 4))

	url := fmt.Sprintf("/project_lastupdates_test/mon/lastupdates")
	req, err := http.NewRequest("GET", url, nil)

	test.NoError(t, loadUserPermissions(api.mustDB(), u))

	c := &businesscontext.Ctx{
		User: u,
	}

	router := mux.NewRouter()
	router.HandleFunc("/project_lastupdates_test/mon/lastupdates",
		func(w http.ResponseWriter, r *http.Request) {
			getUserLastUpdates(w, r, db, c)
		})
	http.Handle("/project_lastupdates_test/", router)

	test.NoError(t, err)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	t.Logf("Code status : %d", w.Code)
	assert.Equal(t, 200, w.Code)

	resp := string(w.Body.Bytes())
	t.Logf("Response: %s", resp)
	buf := bytes.NewBuffer([]byte{})

	w.Header().Write(buf)
	t.Logf("Headers: \n%s", string(buf.Bytes()))

	lastUpdates := []sdk.ProjectLastUpdates{}
	err = json.Unmarshal(w.Body.Bytes(), &lastUpdates)
	test.NoError(t, err)

	assert.Equal(t, 1, len(lastUpdates))
	assert.Equal(t, proj.Name, lastUpdates[0].Name)
	assert.NotZero(t, lastUpdates[0].LastModified)
	assert.Equal(t, 1, len(lastUpdates[0].Applications))
	assert.Equal(t, 1, len(lastUpdates[0].Pipelines))
	assert.Equal(t, app.Name, lastUpdates[0].Applications[0].Name)
	assert.Equal(t, pip.Name, lastUpdates[0].Pipelines[0].Name)
	assert.NotZero(t, lastUpdates[0].Applications[0].LastModified)
	assert.NotZero(t, lastUpdates[0].Pipelines[0].LastModified)
}

func Test_getUserLastUpdatesShouldReturns1Project2Apps1Pipeline(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create a user
	u := sdk.NewUser("testuser")
	u.Admin = false
	u.Email = "mail@mail.com"
	u.Origin = "local"
	u.Auth = sdk.Auth{
		EmailVerified: true,
	}

	//Create a group
	g := &sdk.Group{Name: "testgroup"}

	//Delete user and group
	deleteUser(t, db, u, g)
	//All the project
	deleteAll(t, api, "TEST_LAST_UPDATE")

	//Create a user
	t.Logf("Insert User %s", u.Username)
	test.NoError(t, user.InsertUser(api.mustDB(), u, &u.Auth))

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TEST_LAST_UPDATE", "TEST_LAST_UPDATE", u)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip, u))

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	test.NoError(t, application.Insert(api.mustDB(), api.Cache,  proj, app, u))

	//Create a group
	t.Logf("Insert Group %s", g.Name)
	test.NoError(t, group.InsertGroup(api.mustDB(), g))

	//Add user in group
	test.NoError(t, group.InsertUserInGroup(api.mustDB(), g.ID, u.ID, true))

	//All associations
	test.NoError(t, group.InsertGroupInProject(api.mustDB(), proj.ID, g.ID, 4))
	test.NoError(t, group.InsertGroupInApplication(api.mustDB(), app.ID, g.ID, 4))

	time.Sleep(1 * time.Second)
	//Insert Application
	app2 := &sdk.Application{
		Name: "TEST_APP_2",
	}
	t.Logf("Insert Application %s for Project %s", app2.Name, proj.Name)
	test.NoError(t, application.Insert(api.mustDB(), api.Cache,  proj, app2, u))
	test.NoError(t, group.InsertGroupInApplication(api.mustDB(), app2.ID, g.ID, 4))
	test.NoError(t, group.InsertGroupInPipeline(api.mustDB(), pip.ID, g.ID, 4))

	url := fmt.Sprintf("/project_lastupdates_test1/mon/lastupdates")
	req, err := http.NewRequest("GET", url, nil)

	test.NoError(t, loadUserPermissions(api.mustDB(), u))

	c := &businesscontext.Ctx{
		User: u,
	}

	router := mux.NewRouter()
	router.HandleFunc("/project_lastupdates_test1/mon/lastupdates",
		func(w http.ResponseWriter, r *http.Request) {
			getUserLastUpdates(w, r, db, c)
		})
	http.Handle("/project_lastupdates_test1/", router)

	test.NoError(t, err)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	t.Logf("Code status : %d", w.Code)
	assert.Equal(t, 200, w.Code)

	resp := string(w.Body.Bytes())
	t.Logf("Response: %s", resp)
	buf := bytes.NewBuffer([]byte{})

	w.Header().Write(buf)
	t.Logf("Headers: \n%s", string(buf.Bytes()))

	lastUpdates := []sdk.ProjectLastUpdates{}
	err = json.Unmarshal(w.Body.Bytes(), &lastUpdates)
	test.NoError(t, err)

	assert.Equal(t, 1, len(lastUpdates))
	assert.Equal(t, proj.Name, lastUpdates[0].Name)
	assert.NotZero(t, lastUpdates[0].LastModified)
	assert.Equal(t, 2, len(lastUpdates[0].Applications))
	assert.Equal(t, 1, len(lastUpdates[0].Pipelines))
	assert.Equal(t, app.Name, lastUpdates[0].Applications[0].Name)
	assert.Equal(t, app2.Name, lastUpdates[0].Applications[1].Name)
	assert.Equal(t, pip.Name, lastUpdates[0].Pipelines[0].Name)
	assert.NotZero(t, lastUpdates[0].Applications[0].LastModified)
	assert.NotZero(t, lastUpdates[0].Applications[1].LastModified)
	assert.NotZero(t, lastUpdates[0].Pipelines[0].LastModified)
	assert.True(t, lastUpdates[0].Applications[0].LastModified < lastUpdates[0].Applications[1].LastModified)
}

func Test_getUserLastUpdatesShouldReturns2Project2Apps1Pipeline(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create a user
	u := sdk.NewUser("testuser")
	u.Admin = false
	u.Email = "mail@mail.com"
	u.Origin = "local"
	u.Auth = sdk.Auth{
		EmailVerified: true,
	}

	//Create a group
	g := &sdk.Group{Name: "testgroup"}

	//Delete user and group
	deleteUser(t, db, u, g)
	//All the project
	deleteAll(t, api, "TEST_LAST_UPDATE")
	deleteAll(t, api, "TEST_LAST_UPDATE_2")

	//Create a user
	t.Logf("Insert User %s", u.Username)
	test.NoError(t, user.InsertUser(api.mustDB(), u, &u.Auth))

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TEST_LAST_UPDATE", "TEST_LAST_UPDATE", u)

	proj2 := assets.InsertTestProject(t, db, api.Cache, "TEST_LAST_UPDATE_2", "TEST_LAST_UPDATE_2", u)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip, u))

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	test.NoError(t, application.Insert(api.mustDB(), api.Cache,  proj, app, u))

	//Create a group
	t.Logf("Insert Group %s", g.Name)
	test.NoError(t, group.InsertGroup(api.mustDB(), g))

	//Add user in group
	err := group.InsertUserInGroup(api.mustDB(), g.ID, u.ID, true)
	test.NoError(t, err)

	//All associations
	err = group.InsertGroupInProject(api.mustDB(), proj.ID, g.ID, 4)
	test.NoError(t, err)
	err = group.InsertGroupInProject(api.mustDB(), proj2.ID, g.ID, 4)
	test.NoError(t, err)
	err = group.InsertGroupInApplication(api.mustDB(), app.ID, g.ID, 4)
	test.NoError(t, err)

	//Insert Application
	app2 := &sdk.Application{
		Name: "TEST_APP_2",
	}
	t.Logf("Insert Application %s for Project %s", app2.Name, proj.Name)
	err = application.Insert(api.mustDB(), api.Cache,  proj, app2, u)
	test.NoError(t, err)
	err = group.InsertGroupInApplication(api.mustDB(), app2.ID, g.ID, 4)
	test.NoError(t, err)
	err = group.InsertGroupInPipeline(api.mustDB(), pip.ID, g.ID, 4)
	test.NoError(t, err)

	url := fmt.Sprintf("/project_lastupdates_test2/mon/lastupdates")
	req, err := http.NewRequest("GET", url, nil)

	test.NoError(t, loadUserPermissions(api.mustDB(), u))

	c := &businesscontext.Ctx{
		User: u,
	}

	router := mux.NewRouter()
	router.HandleFunc("/project_lastupdates_test2/mon/lastupdates",
		func(w http.ResponseWriter, r *http.Request) {
			getUserLastUpdates(w, r, db, c)
		})
	http.Handle("/project_lastupdates_test2/", router)

	test.NoError(t, err)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	t.Logf("Code status : %d", w.Code)
	assert.Equal(t, 200, w.Code)

	resp := string(w.Body.Bytes())
	t.Logf("Response: %s", resp)
	buf := bytes.NewBuffer([]byte{})

	w.Header().Write(buf)
	t.Logf("Headers: \n%s", string(buf.Bytes()))

	lastUpdates := []sdk.ProjectLastUpdates{}
	err = json.Unmarshal(w.Body.Bytes(), &lastUpdates)
	test.NoError(t, err)

	assert.Equal(t, 2, len(lastUpdates))
	for _, p := range lastUpdates {
		if p.Name == proj.Name {
			assert.Equal(t, proj.Name, p.Name)
			assert.NotZero(t, p.LastModified)
			assert.Equal(t, 2, len(p.Applications))
			assert.Equal(t, 1, len(p.Pipelines))
			assert.Equal(t, app.Name, p.Applications[0].Name)
			assert.Equal(t, app2.Name, p.Applications[1].Name)
			assert.Equal(t, pip.Name, p.Pipelines[0].Name)
			assert.NotZero(t, p.Applications[0].LastModified)
			assert.NotZero(t, p.Applications[1].LastModified)
			assert.NotZero(t, p.Pipelines[0].LastModified)
		} else {
			assert.Equal(t, proj2.Name, p.Name)
			assert.Equal(t, 0, len(p.Applications))
			assert.Equal(t, 0, len(p.Pipelines))
		}
	}
}

func Test_getUserLastUpdatesShouldReturns1Project1Apps1PipelineWithSinceHeader(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create a user
	u := sdk.NewUser("testuser")
	u.Admin = false
	u.Email = "mail@mail.com"
	u.Origin = "local"
	u.Auth = sdk.Auth{
		EmailVerified: true,
	}

	//Create a group
	g := &sdk.Group{Name: "testgroup"}

	//Delete user and group
	deleteUser(t, db, u, g)
	//All the project
	deleteAll(t, api, "TEST_LAST_UPDATE")

	//Create a user
	t.Logf("Insert User %s", u.Username)
	test.NoError(t, user.InsertUser(api.mustDB(), u, &u.Auth))

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TEST_LAST_UPDATE", "TEST_LAST_UPDATE", u)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	err := pipeline.InsertPipeline(api.mustDB(), proj, pip, u)
	test.NoError(t, err)

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	err = application.Insert(api.mustDB(), api.Cache,  proj, app, u)
	test.NoError(t, err)

	//Create a group
	t.Logf("Insert Group %s", g.Name)
	err = group.InsertGroup(api.mustDB(), g)
	test.NoError(t, err)

	//Add user in group
	err = group.InsertUserInGroup(api.mustDB(), g.ID, u.ID, true)
	test.NoError(t, err)

	//All associations
	err = group.InsertGroupInProject(api.mustDB(), proj.ID, g.ID, 4)
	test.NoError(t, err)
	err = group.InsertGroupInApplication(api.mustDB(), app.ID, g.ID, 4)
	test.NoError(t, err)

	time.Sleep(1 * time.Second)
	since := time.Now()
	time.Sleep(2 * time.Second)
	//Insert Application
	app2 := &sdk.Application{
		Name: "TEST_APP_2",
	}
	t.Logf("Insert Application %s for Project %s", app2.Name, proj.Name)
	err = application.Insert(api.mustDB(), api.Cache,  proj, app2, u)
	test.NoError(t, err)
	test.NoError(t, project.UpdateLastModified(api.mustDB(), u, proj))
	err = group.InsertGroupInApplication(api.mustDB(), app2.ID, g.ID, 4)
	test.NoError(t, err)
	err = group.InsertGroupInPipeline(api.mustDB(), pip.ID, g.ID, 4)
	test.NoError(t, err)

	url := fmt.Sprintf("/project_lastupdates_test3/mon/lastupdates")
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("If-Modified-Since", since.Format(time.RFC1123))

	test.NoError(t, loadUserPermissions(api.mustDB(), u))

	c := &businesscontext.Ctx{
		User: u,
	}

	router := mux.NewRouter()
	router.HandleFunc("/project_lastupdates_test3/mon/lastupdates",
		func(w http.ResponseWriter, r *http.Request) {
			getUserLastUpdates(w, r, db, c)
		})
	http.Handle("/project_lastupdates_test3/", router)

	test.NoError(t, err)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	t.Logf("Code status : %d", w.Code)
	assert.Equal(t, 200, w.Code)

	resp := string(w.Body.Bytes())
	t.Logf("Response: %s", resp)
	buf := bytes.NewBuffer([]byte{})

	w.Header().Write(buf)
	t.Logf("Headers: \n%s", string(buf.Bytes()))

	lastUpdates := []sdk.ProjectLastUpdates{}
	err = json.Unmarshal(w.Body.Bytes(), &lastUpdates)
	test.NoError(t, err)

	assert.Equal(t, 1, len(lastUpdates))
	assert.Equal(t, proj.Name, lastUpdates[0].Name)
	assert.NotZero(t, lastUpdates[0].LastModified)
	assert.Equal(t, 1, len(lastUpdates[0].Applications))
	assert.Equal(t, 0, len(lastUpdates[0].Pipelines))
	assert.Equal(t, app2.Name, lastUpdates[0].Applications[0].Name)
	assert.NotZero(t, lastUpdates[0].Applications[0].LastModified)

}

func Test_getUserLastUpdatesShouldReturnsNothingWithSinceHeader(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create a user
	u := sdk.NewUser("testuser")
	u.Admin = false
	u.Email = "mail@mail.com"
	u.Origin = "local"
	u.Auth = sdk.Auth{
		EmailVerified: true,
	}

	//Create a group
	g := &sdk.Group{Name: "testgroup"}

	//Delete user and group
	deleteUser(t, db, u, g)
	//All the project
	deleteAll(t, api, "TEST_LAST_UPDATE")

	//Create a user
	t.Logf("Insert User %s", u.Username)
	err := user.InsertUser(api.mustDB(), u, &u.Auth)
	test.NoError(t, err)

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TEST_LAST_UPDATE", "TEST_LAST_UPDATE", u)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.InsertPipeline(api.mustDB(), proj, pip, u)
	test.NoError(t, err)

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	err = application.Insert(api.mustDB(), api.Cache,  proj, app, u)
	test.NoError(t, err)

	//Create a group
	t.Logf("Insert Group %s", g.Name)
	err = group.InsertGroup(api.mustDB(), g)
	test.NoError(t, err)

	//Add user in group
	err = group.InsertUserInGroup(api.mustDB(), g.ID, u.ID, true)
	test.NoError(t, err)

	//All associations
	err = group.InsertGroupInProject(api.mustDB(), proj.ID, g.ID, 4)
	test.NoError(t, err)
	err = group.InsertGroupInApplication(api.mustDB(), app.ID, g.ID, 4)
	test.NoError(t, err)

	//Insert Application
	app2 := &sdk.Application{
		Name: "TEST_APP_2",
	}
	t.Logf("Insert Application %s for Project %s", app2.Name, proj.Name)
	err = application.Insert(api.mustDB(), api.Cache,  proj, app2, u)

	test.NoError(t, err)
	err = group.InsertGroupInApplication(api.mustDB(), app2.ID, g.ID, 4)
	test.NoError(t, err)
	err = group.InsertGroupInPipeline(api.mustDB(), pip.ID, g.ID, 4)
	test.NoError(t, err)

	url := fmt.Sprintf("/project_lastupdates_test4/mon/lastupdates")
	req, err := http.NewRequest("GET", url, nil)

	time.Sleep(1 * time.Second)
	since := time.Now()

	req.Header.Set("If-Modified-Since", since.Format(time.RFC1123))

	test.NoError(t, loadUserPermissions(api.mustDB(), u))

	c := &businesscontext.Ctx{
		User: u,
	}

	router := mux.NewRouter()
	router.HandleFunc("/project_lastupdates_test4/mon/lastupdates",
		func(w http.ResponseWriter, r *http.Request) {
			getUserLastUpdates(w, r, db, c)
		})
	http.Handle("/project_lastupdates_test4/", router)

	test.NoError(t, err)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	t.Logf("Code status : %d", w.Code)
	assert.Equal(t, 304, w.Code)

	resp := string(w.Body.Bytes())
	t.Logf("Response: %s", resp)
	buf := bytes.NewBuffer([]byte{})

	w.Header().Write(buf)
	t.Logf("Headers: \n%s", string(buf.Bytes()))

	assert.Empty(t, w.Body.Bytes())
}
*/
