package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/notification_v2"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_CrudProjectNotification(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManageNotification, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	// INSERT Notification
	notif := sdk.ProjectNotification{
		Name:       "cds",
		ProjectKey: proj.Key,
		WebHookURL: "http://lolcat.host",
		Auth: sdk.ProjectNotificationAuth{
			Headers: map[string]string{
				"Authorization": "Bearer aaaaa",
			},
		},
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectNotificationHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, notif)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// List notifications
	varsList := map[string]string{
		"projectKey": proj.Key,
	}
	uriList := api.Router.GetRouteV2("GET", api.getProjectNotifsHandler, varsList)
	test.NotEmpty(t, uriList)
	reqList := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriList, nil)
	wList := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wList, reqList)
	require.Equal(t, 200, wList.Code)

	var nsList []sdk.ProjectNotification
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &nsList))

	require.Equal(t, 1, len(nsList))
	require.Equal(t, nsList[0].Name, notif.Name)

	// Get notification
	varsGet1 := map[string]string{
		"projectKey":   proj.Key,
		"notification": notif.Name,
	}
	uriGet1 := api.Router.GetRouteV2("GET", api.getProjectNotificationHandler, varsGet1)
	test.NotEmpty(t, uriGet1)
	reqGet1 := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGet1, nil)
	wGet1 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet1, reqGet1)
	require.Equal(t, 200, wGet1.Code)

	var nGet sdk.ProjectNotification
	require.NoError(t, json.Unmarshal(wGet1.Body.Bytes(), &nGet))

	require.Equal(t, notif.Name, nGet.Name)

	// Update notification
	notif.Filters = sdk.ProjectNotificationFilters{
		"filter1": sdk.ProjectNotificationFilter{
			Events: []string{"RunJob.*"},
		},
	}

	varsUpdate := map[string]string{
		"projectKey":   proj.Key,
		"notification": notif.Name,
	}
	uriUpdate := api.Router.GetRouteV2("PUT", api.putProjectNotificationHandler, varsUpdate)
	test.NotEmpty(t, uriUpdate)
	reqUpdate := assets.NewAuthentifiedRequest(t, user1, pass, "PUT", uriUpdate, notif)
	wUpdate := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wUpdate, reqUpdate)
	require.Equal(t, 200, wUpdate.Code)

	var nUpdate sdk.ProjectNotification
	require.NoError(t, json.Unmarshal(wUpdate.Body.Bytes(), &nUpdate))

	require.Equal(t, 1, len(nUpdate.Filters))
	require.Equal(t, len(nUpdate.Filters["filter1"].Events), 1)
	require.Equal(t, "RunJob.*", nUpdate.Filters["filter1"].Events[0])

	// Load with decrypt
	notifDB, err := notification_v2.LoadByName(context.TODO(), db, proj.Key, notif.Name, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, notifDB.Auth.Headers["Authorization"], "Bearer aaaaa")

	notifDBs, err := notification_v2.LoadAll(context.TODO(), db, proj.Key, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, notifDBs[0].Auth.Headers["Authorization"], "Bearer aaaaa")

	// Delete notification
	varsDelete := map[string]string{
		"projectKey":   proj.Key,
		"notification": notif.Name,
	}
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteProjectNotificationHandler, varsDelete)
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, user1, pass, "DELETE", uriDelete, notif)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)
}
