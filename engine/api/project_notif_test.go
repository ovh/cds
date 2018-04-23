package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	// Create project
	p := assets.InsertTestProject(t, db, api.Cache, strings.ToUpper(sdk.RandomString(4)), sdk.RandomString(10), u)
	test.NoError(t, group.InsertUserInGroup(api.mustDB(context.Background()), p.ProjectGroups[0].Group.ID, u.ID, true))

	app := &sdk.Application{Name: sdk.RandomString(10)}
	err := application.Insert(api.mustDB(context.Background()), api.Cache, p, app, u)
	test.NoError(t, err)
	test.NoError(t, group.InsertGroupInApplication(api.mustDB(context.Background()), app.ID, p.ProjectGroups[0].Group.ID, 7))

	pip := &sdk.Pipeline{
		Name:      sdk.RandomString(10),
		Type:      "build",
		ProjectID: p.ID,
	}
	err = pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, p, pip, nil)
	test.NoError(t, err)
	test.NoError(t, group.InsertGroupInPipeline(api.mustDB(context.Background()), pip.ID, p.ProjectGroups[0].Group.ID, 7))

	_, err = application.AttachPipeline(api.mustDB(context.Background()), app.ID, pip.ID)
	test.NoError(t, err)

	appPips, err := application.GetAllPipelinesByID(api.mustDB(context.Background()), app.ID)
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

	jsonBody, _ := json.Marshal(notifsToAdd)
	body := bytes.NewBuffer(jsonBody)
	uri := router.GetRoute("POST", api.addNotificationsHandler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	vars = map[string]string{
		"permProjectKey": p.Key,
	}

	uri = router.GetRoute("GET", api.getProjectNotificationsHandler, vars)
	req, err = http.NewRequest("GET", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var notifs []sdk.UserNotification
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &notifs))
}
