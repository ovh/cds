package main

import (
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_getProjectNotificationsHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getProjectNotificationsHandler"}
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	// Create project
	p := assets.InsertTestProject(t, db, strings.ToUpper(assets.RandomString(t, 4)), assets.RandomString(t, 10))
	test.NoError(t, group.InsertUserInGroup(db, p.ProjectGroups[0].Group.ID, u.ID, true))

	app := &sdk.Application{Name: assets.RandomString(t, 10)}
	err := application.InsertApplication(db, p, app)
	test.NoError(t, err)
	test.NoError(t, group.InsertGroupInApplication(db, app.ID, p.ProjectGroups[0].Group.ID, 7))

	pip := &sdk.Pipeline{
		Name:      assets.RandomString(t, 10),
		Type:      "build",
		ProjectID: p.ID,
	}
	err = pipeline.InsertPipeline(db, pip)
	test.NoError(t, err)
	test.NoError(t, group.InsertGroupInPipeline(db, pip.ID, p.ProjectGroups[0].Group.ID, 7))

	err = application.AttachPipeline(db, app.ID, pip.ID)
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
	tester.AddCall("Test_getProjectNotificationsHandler", "POST", route, notifsToAdd).Headers(headers).Checkers(iffy.ExpectStatus(200))
	tester.Run()
	tester.Reset()

	vars = map[string]string{
		"permProjectKey": p.Key,
	}
	route = router.getRoute("GET", getProjectNotificationsHandler, vars)
	tester.AddCall("Test_getProjectNotificationsHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t))
	tester.Run()

}
