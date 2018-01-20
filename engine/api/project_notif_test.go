package api

import (
	"strings"
	"testing"

	"github.com/yesnault/gadgeto/iffy"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_getProjectNotificationsHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	// Create project
	p := assets.InsertTestProject(t, db, api.Cache, strings.ToUpper(sdk.RandomString(4)), sdk.RandomString(10), u)
	test.NoError(t, group.InsertUserInGroup(api.mustDB(), p.ProjectGroups[0].Group.ID, u.ID, true))

	app := &sdk.Application{Name: sdk.RandomString(10)}
	err := application.Insert(api.mustDB(), api.Cache, p, app, u)
	test.NoError(t, err)
	test.NoError(t, group.InsertGroupInApplication(api.mustDB(), app.ID, p.ProjectGroups[0].Group.ID, 7))

	pip := &sdk.Pipeline{
		Name:      sdk.RandomString(10),
		Type:      "build",
		ProjectID: p.ID,
	}
	err = pipeline.InsertPipeline(api.mustDB(), api.Cache, p, pip, nil)
	test.NoError(t, err)
	test.NoError(t, group.InsertGroupInPipeline(api.mustDB(), pip.ID, p.ProjectGroups[0].Group.ID, 7))

	_, err = application.AttachPipeline(api.mustDB(), app.ID, pip.ID)
	test.NoError(t, err)

	appPips, err := application.GetAllPipelinesByID(api.mustDB(), app.ID)
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
	route := router.GetRoute("POST", api.addNotificationsHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)
	tester.AddCall("Test_getProjectNotificationsHandler", "POST", route, notifsToAdd).Headers(headers).Checkers(iffy.ExpectStatus(200))
	tester.Run()
	tester.Reset()

	vars = map[string]string{
		"permProjectKey": p.Key,
	}
	route = router.GetRoute("GET", api.getProjectNotificationsHandler, vars)
	tester.AddCall("Test_getProjectNotificationsHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t))
	tester.Run()

}
