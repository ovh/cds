package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/loopfz/gadgeto/iffy"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func deleteAll(t *testing.T, db *gorp.DbMap, key string) error {
	// Delete all apps
	t.Logf("start deleted : %s", key)
	proj, err := project.Load(db, key, &sdk.User{Admin: true})
	if err != nil {
		return err
	}

	apps, err := application.LoadAll(db, key, &sdk.User{Admin: true})
	if err != nil {
		t.Logf("Cannot list app: %s", err)
		return err
	}
	for _, app := range apps {
		tx, _ := db.Begin()
		err = application.DeleteApplication(tx, app.ID)
		if err != nil {
			t.Logf("DeleteApplication: %s", err)
			return err
		}
		_ = tx.Commit()
	}

	// Delete all pipelines
	pips, err := pipeline.LoadPipelines(db, proj.ID, false, &sdk.User{Admin: true})
	if err != nil {
		t.Logf("ListPipelines: %s", err)
		return err
	}
	for _, pip := range pips {
		err = pipeline.DeletePipeline(db, pip.ID, 1)
		if err != nil {
			t.Logf("DeletePipeline: %s", err)
			return err
		}
	}

	if err := group.LoadGroupByProject(db, proj); err != nil {
		return err
	}

	for _, g := range proj.ProjectGroups {
		if err := group.DeleteGroupAndDependencies(db, &g.Group); err != nil {
			return err
		}
	}

	// Delete project
	err = project.Delete(db, key)
	if err != nil {
		t.Logf("RemoveProject: %s", err)
		return err
	}
	t.Logf("All deleted")
	return nil
}

func testApplicationPipelineNotifBoilerPlate(t *testing.T, f func(*testing.T, *gorp.DbMap, *sdk.Project, *sdk.Pipeline, *sdk.Application, *sdk.Environment, *sdk.User)) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = deleteAll(t, db, "TEST_APP_PIPELINE_NOTIF")

	//Insert Project
	proj := assets.InsertTestProject(t, db, "TEST_APP_PIPELINE_NOTIF", "TEST_APP_PIPELINE_NOTIF")

	u, p := assets.InsertAdminUser(t, db)
	u.Auth.HashedPassword = p

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	err := pipeline.InsertPipeline(db, pip)
	test.NoError(t, err)

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	err = application.Insert(db, proj, app)

	env := &sdk.DefaultEnv

	t.Logf("Attach Pipeline %s on Application %s", pip.Name, app.Name)
	_, err = application.AttachPipeline(db, app.ID, pip.ID)
	test.NoError(t, err)

	f(t, db, proj, pip, app, env, u)

	t.Logf("Detach Pipeline %s on Application %s", pip.Name, app.Name)
	tx, err := db.Begin()
	test.NoError(t, err)
	err = application.RemovePipeline(tx, proj.Key, app.Name, pip.Name)
	test.NoError(t, err)
	err = tx.Commit()
	test.NoError(t, err)

	err = application.DeleteAllApplicationPipeline(db, app.ID)
	test.NoError(t, err)

	err = environment.DeleteAllEnvironment(db, proj.ID)
	test.NoError(t, err)

	//Delete application
	t.Logf("Delete Application %s for Project %s", app.Name, proj.Name)
	tx, err = db.Begin()
	test.NoError(t, err)
	err = application.DeleteApplication(tx, app.ID)
	test.NoError(t, err)
	err = tx.Commit()
	test.NoError(t, err)

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(db, pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = assets.DeleteTestProject(t, db, "TEST_APP_PIPELINE_NOTIF")
	test.NoError(t, err)
}

func testCheckUserNotificationSettings(t *testing.T, n1, n2 map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings) {
	for k, v := range n1 {
		t.Logf("Checkin %s: %s", k, v)
		assert.NotNil(t, n2[k])
		if k == sdk.JabberUserNotification || k == sdk.EmailUserNotification {
			j1, ok := v.(*sdk.JabberEmailUserNotificationSettings)
			assert.True(t, ok, "Should be type JabberEmailUserNotificationSettings")
			j2, ok := n2[k].(*sdk.JabberEmailUserNotificationSettings)
			assert.True(t, ok, "Should be type JabberEmailUserNotificationSettings")
			assert.Equal(t, j1.OnFailure, j2.OnFailure)
			assert.Equal(t, j1.OnSuccess, j2.OnSuccess)
			assert.Equal(t, j1.OnStart, j2.OnStart)
			assert.Equal(t, j1.SendToAuthor, j2.SendToAuthor)
			assert.Equal(t, j1.SendToGroups, j2.SendToGroups)
			assert.Equal(t, len(j1.Recipients), len(j2.Recipients))
			if len(j1.Recipients) == len(j2.Recipients) {
				for i := range j1.Recipients {
					assert.Equal(t, j1.Recipients[i], j2.Recipients[i])
				}
			}
			assert.Equal(t, j1.Template.Subject, j2.Template.Subject)
			assert.Equal(t, j1.Template.Body, j2.Template.Body)
		}
	}
}

func Test_LoadEmptyApplicationPipelineNotif(t *testing.T) {
	testApplicationPipelineNotifBoilerPlate(t, func(t *testing.T, db *gorp.DbMap, proj *sdk.Project, pip *sdk.Pipeline, app *sdk.Application, env *sdk.Environment, u *sdk.User) {
		t.Logf("Load Application Pipeline Notif %s %s", app.Name, env.Name)
		notif, err := notification.LoadUserNotificationSettings(db, app.ID, pip.ID, env.ID)
		test.NoError(t, err)
		assert.Nil(t, notif)
	})
}

func Test_InsertAndLoadApplicationPipelineNotif(t *testing.T) {
	testApplicationPipelineNotifBoilerPlate(t, func(t *testing.T, db *gorp.DbMap, proj *sdk.Project, pip *sdk.Pipeline, app *sdk.Application, env *sdk.Environment, u *sdk.User) {
		notif := sdk.UserNotification{
			Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
				sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
					OnSuccess:    "on_success",
					OnStart:      true,
					OnFailure:    "on_failure",
					SendToAuthor: true,
					SendToGroups: true,
					Recipients:   []string{"1", "2"},
					Template: sdk.UserNotificationTemplate{
						Subject: "subject",
						Body:    "body",
					},
				},
				sdk.EmailUserNotification: &sdk.JabberEmailUserNotificationSettings{
					OnSuccess:    "on_success_",
					OnStart:      true,
					OnFailure:    "on_failure_",
					SendToAuthor: true,
					SendToGroups: true,
					Recipients:   []string{"1", "2", "3"},
					Template: sdk.UserNotificationTemplate{
						Subject: "subject_",
						Body:    "body_",
					},
				},
			},
			Environment: *env,
		}

		err := notification.InsertOrUpdateUserNotificationSettings(db, app.ID, pip.ID, env.ID, &notif)
		test.NoError(t, err)

		t.Logf("Load Application Pipeline Notif %s %s", app.Name, env.Name)
		notif1, err := notification.LoadUserNotificationSettings(db, app.ID, pip.ID, env.ID)
		test.NoError(t, err)
		assert.NotNil(t, notif1)

		testCheckUserNotificationSettings(t, notif.Notifications, notif1.Notifications)
	})
}

func Test_getUserNotificationApplicationPipelineHandlerReturnsEmptyUserNotification(t *testing.T) {
	testApplicationPipelineNotifBoilerPlate(t, func(t *testing.T, db *gorp.DbMap, proj *sdk.Project, pip *sdk.Pipeline, app *sdk.Application, env *sdk.Environment, u *sdk.User) {
		url := fmt.Sprintf("/test1/project/%s/application/%s/pipeline/%s/notification", proj.Key, app.Name, pip.Name)
		req, err := http.NewRequest("GET", url, nil)

		assets.AuthentifyRequest(t, req, u, u.Auth.HashedPassword)

		router := mux.NewRouter()
		router.HandleFunc("/test1/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/notification",
			func(w http.ResponseWriter, r *http.Request) {
				getUserNotificationApplicationPipelineHandler(w, r, db, &context.Ctx{
					User: u,
				})
			})
		http.Handle("/test1/", router)

		test.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 204, w.Code)
		assert.Equal(t, "null", w.Body.String())
	})
}

func Test_getUserNotificationApplicationPipelineHandlerReturnsNonEmptyUserNotification(t *testing.T) {
	testApplicationPipelineNotifBoilerPlate(t, func(t *testing.T, db *gorp.DbMap, proj *sdk.Project, pip *sdk.Pipeline, app *sdk.Application, env *sdk.Environment, u *sdk.User) {
		notif := sdk.UserNotification{
			Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
				sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
					OnSuccess:    "on_success",
					OnStart:      true,
					OnFailure:    "on_failure",
					SendToAuthor: true,
					SendToGroups: true,
					Recipients:   []string{"1", "2"},
					Template: sdk.UserNotificationTemplate{
						Subject: "subject",
						Body:    "body",
					},
				},
				sdk.EmailUserNotification: &sdk.JabberEmailUserNotificationSettings{
					OnSuccess:    "on_success_",
					OnStart:      true,
					OnFailure:    "on_failure_",
					SendToAuthor: true,
					SendToGroups: true,
					Recipients:   []string{"1", "2", "3"},
					Template: sdk.UserNotificationTemplate{
						Subject: "subject_",
						Body:    "body_",
					},
				},
			},
		}

		err := notification.InsertOrUpdateUserNotificationSettings(db, app.ID, pip.ID, env.ID, &notif)
		test.NoError(t, err)

		url := fmt.Sprintf("/test2/project/%s/application/%s/pipeline/%s/notification", proj.Key, app.Name, pip.Name)
		req, err := http.NewRequest("GET", url, nil)
		assets.AuthentifyRequest(t, req, u, u.Auth.HashedPassword)
		router := mux.NewRouter()
		router.HandleFunc("/test2/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/notification",
			func(w http.ResponseWriter, r *http.Request) {
				getUserNotificationApplicationPipelineHandler(w, r, db, &context.Ctx{
					User: u,
				})
			})
		http.Handle("/test2/", router)

		test.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		notif1 := &sdk.UserNotification{}
		test.NoError(t, json.Unmarshal(w.Body.Bytes(), notif1))
		assert.Equal(t, notif.ApplicationPipelineID, notif1.ApplicationPipelineID)
		assert.Equal(t, notif.Environment.ID, notif1.Environment.ID)

		testCheckUserNotificationSettings(t, notif.Notifications, notif1.Notifications)
	})
}

func Test_getNotificationTypeHandler(t *testing.T) {
	url := fmt.Sprintf("/test3/notification/type")
	req, err := http.NewRequest("GET", url, nil)
	router := mux.NewRouter()
	router.HandleFunc("/test3/notification/type",
		func(w http.ResponseWriter, r *http.Request) {
			getUserNotificationTypeHandler(w, r, nil, nil)
		})
	http.Handle("/test3/", router)

	test.NoError(t, err)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var s = []string{}
	err = json.Unmarshal(w.Body.Bytes(), &s)
	test.NoError(t, err)
	assert.Equal(t, 200, w.Code)
}

func Test_updateUserNotificationApplicationPipelineHandler(t *testing.T) {
	testApplicationPipelineNotifBoilerPlate(t, func(t *testing.T, db *gorp.DbMap, proj *sdk.Project, pip *sdk.Pipeline, app *sdk.Application, env *sdk.Environment, u *sdk.User) {
		notif := sdk.UserNotification{
			Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
				sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
					OnSuccess:    "on_success",
					OnStart:      true,
					OnFailure:    "on_failure",
					SendToAuthor: true,
					SendToGroups: true,
					Recipients:   []string{"1", "2"},
					Template: sdk.UserNotificationTemplate{
						Subject: "subject",
						Body:    "body",
					},
				},
				sdk.EmailUserNotification: &sdk.JabberEmailUserNotificationSettings{
					OnSuccess:    "on_success_",
					OnStart:      true,
					OnFailure:    "on_failure_",
					SendToAuthor: true,
					SendToGroups: true,
					Recipients:   []string{"1", "2", "3"},
					Template: sdk.UserNotificationTemplate{
						Subject: "subject_",
						Body:    "body_",
					},
				},
			},
		}

		err := notification.InsertOrUpdateUserNotificationSettings(db, app.ID, pip.ID, env.ID, &notif)
		test.NoError(t, err)

		notif = sdk.UserNotification{
			Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
				sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
					OnSuccess:    "on_success",
					OnStart:      true,
					OnFailure:    "on_failure",
					SendToAuthor: true,
					SendToGroups: true,
					Recipients:   []string{"1", "2"},
					Template: sdk.UserNotificationTemplate{
						Subject: "subject",
						Body:    "body",
					},
				},
			},
			Environment: *env,
		}

		b, err := json.Marshal(notif)
		test.NoError(t, err)
		body := bytes.NewBuffer(b)

		url := fmt.Sprintf("/test4/project/%s/application/%s/pipeline/%s/notification", proj.Key, app.Name, pip.Name)
		req, err := http.NewRequest("POST", url, body)
		test.NoError(t, err)
		assets.AuthentifyRequest(t, req, u, u.Auth.HashedPassword)
		router := mux.NewRouter()
		router.HandleFunc("/test4/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/notification",
			func(w http.ResponseWriter, r *http.Request) {
				updateUserNotificationApplicationPipelineHandler(w, r, db, &context.Ctx{
					User: u,
				})
			})

		http.Handle("/test4/", router)

		test.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var appResponse sdk.Application
		err = json.Unmarshal(w.Body.Bytes(), &appResponse)
		test.NoError(t, err)

		assert.Equal(t, len(appResponse.Notifications), 1)

		assert.Equal(t, notif.Environment.ID, appResponse.Notifications[0].Environment.ID)

		testCheckUserNotificationSettings(t, notif.Notifications, appResponse.Notifications[0].Notifications)

	})
}

func Test_ShouldSendUserNotificationOnStartTrue(t *testing.T) {
	notif := sdk.UserNotification{
		Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
			sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
				OnSuccess:    "always",
				OnStart:      true,
				OnFailure:    "always",
				SendToAuthor: true,
				SendToGroups: true,
				Recipients:   []string{"1", "2"},
				Template: sdk.UserNotificationTemplate{
					Subject: "subject",
					Body:    "body",
				},
			},
		},
	}

	current := sdk.PipelineBuild{
		Status: sdk.StatusBuilding,
	}

	assert.True(t, notification.ShouldSendUserNotification(notif.Notifications[sdk.JabberUserNotification], &current, nil))
}

func Test_ShouldNotSendUserNotificationOnStartFalse(t *testing.T) {
	notif := sdk.UserNotification{
		Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
			sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
				OnSuccess:    "always",
				OnStart:      false,
				OnFailure:    "always",
				SendToAuthor: true,
				SendToGroups: true,
				Recipients:   []string{"1", "2"},
				Template: sdk.UserNotificationTemplate{
					Subject: "subject",
					Body:    "body",
				},
			},
		},
	}

	current := sdk.PipelineBuild{
		Status: sdk.StatusBuilding,
	}

	assert.False(t, notification.ShouldSendUserNotification(notif.Notifications[sdk.JabberUserNotification], &current, nil))
}

func Test_ShouldSendUserNotificationOnSuccessAlways(t *testing.T) {
	notif := sdk.UserNotification{
		Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
			sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
				OnSuccess:    "always",
				OnStart:      true,
				OnFailure:    "always",
				SendToAuthor: true,
				SendToGroups: true,
				Recipients:   []string{"1", "2"},
				Template: sdk.UserNotificationTemplate{
					Subject: "subject",
					Body:    "body",
				},
			},
		},
	}

	current := sdk.PipelineBuild{
		Status: sdk.StatusSuccess,
	}

	assert.True(t, notification.ShouldSendUserNotification(notif.Notifications[sdk.JabberUserNotification], &current, nil))
}

func Test_ShouldNotSendUserNotificationOnSuccessNever(t *testing.T) {
	notif := sdk.UserNotification{
		Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
			sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
				OnSuccess:    "never",
				OnStart:      true,
				OnFailure:    "always",
				SendToAuthor: true,
				SendToGroups: true,
				Recipients:   []string{"1", "2"},
				Template: sdk.UserNotificationTemplate{
					Subject: "subject",
					Body:    "body",
				},
			},
		},
	}

	current := sdk.PipelineBuild{
		Status: sdk.StatusSuccess,
	}

	assert.False(t, notification.ShouldSendUserNotification(notif.Notifications[sdk.JabberUserNotification], &current, nil))
}

func Test_SendPipeline(t *testing.T) {
	testApplicationPipelineNotifBoilerPlate(t, func(t *testing.T, db *gorp.DbMap, proj *sdk.Project, pip *sdk.Pipeline, app *sdk.Application, env *sdk.Environment, u *sdk.User) {
		notif := sdk.UserNotification{
			Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
				sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
					OnSuccess:    "on_success",
					OnStart:      true,
					OnFailure:    "on_failure",
					SendToAuthor: true,
					SendToGroups: true,
					Recipients:   []string{"1", "2"},
					Template: sdk.UserNotificationTemplate{
						Subject: "CDS {{.cds.project}}/{{.cds.application}} {{.cds.pipeline}} {{.cds.status}}",
						Body:    "\nDetails : {{.cds.buildURL}}",
					},
				},
			},
		}
		cache.Initialize("local", "", "", 5)
		err := notification.InsertOrUpdateUserNotificationSettings(db, app.ID, pip.ID, env.ID, &notif)
		test.NoError(t, err)

		tx, err := db.Begin()
		test.NoError(t, err)

		params := []sdk.Parameter{}
		trigger := sdk.PipelineBuildTrigger{}

		pb, err := pipeline.InsertPipelineBuild(tx, proj, pip, app, params, params, env, -1, trigger)
		test.NoError(t, err)

		err = tx.Commit()
		test.NoError(t, err)

		var event *sdk.Event
		cache.Dequeue("events", &event)
		assert.Equal(t, event.EventType, "sdk.EventNotif", nil)
		cache.Dequeue("events", &event)
		assert.Equal(t, event.EventType, "sdk.EventPipelineBuild", nil)

		err = pipeline.DeletePipelineBuildByID(db, pb.ID)
		test.NoError(t, err)

	})
}

func Test_addNotificationsHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getSchedulerApplicationPipelineHandler"}
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	// Create project
	p := assets.InsertTestProject(t, db, strings.ToUpper(assets.RandomString(t, 4)), assets.RandomString(t, 10))

	app := &sdk.Application{Name: assets.RandomString(t, 10)}

	err := application.Insert(db, p, app)
	test.NoError(t, err)

	pip := &sdk.Pipeline{
		Name:      assets.RandomString(t, 10),
		Type:      "build",
		ProjectID: p.ID,
	}
	err = pipeline.InsertPipeline(db, pip)
	test.NoError(t, err)

	_, err = application.AttachPipeline(db, app.ID, pip.ID)
	test.NoError(t, err)

	appPips, err := application.GetAllPipelinesByID(db, app.ID)
	test.NoError(t, err)

	notifsToAdd := []sdk.UserNotification{}
	notifsToAdd = append(notifsToAdd, sdk.UserNotification{
		ApplicationPipelineID: appPips[0].ID,
		Pipeline:              *pip,
		Notifications: map[sdk.UserNotificationSettingsType]sdk.UserNotificationSettings{
			sdk.JabberUserNotification: &sdk.JabberEmailUserNotificationSettings{
				OnSuccess:    "on_success",
				OnStart:      true,
				OnFailure:    "on_failure",
				SendToAuthor: true,
				SendToGroups: true,
				Recipients:   []string{"1", "2"},
				Template: sdk.UserNotificationTemplate{
					Subject: "subject",
					Body:    "body",
				},
			},
		},
	})

	vars := map[string]string{
		"key": p.Key,
		"permApplicationName": app.Name,
	}
	route := router.getRoute("POST", addNotificationsHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)
	tester.AddCall("Test_addNotificationsHandler", "POST", route, notifsToAdd).Headers(headers).Checkers(iffy.ExpectStatus(200))
	tester.Run()

	notifications, errN := notification.LoadAllUserNotificationSettings(db, app.ID)
	test.NoError(t, errN)

	assert.Equal(t, len(notifications), 1)
}
