package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getAllRepositoriesHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	// Clean db
	_, err := db.Exec("delete from project_repository")
	require.NoError(t, err)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertAdminUser(t, db)

	vcsProj := assets.InsertTestVCSProject(t, db, proj.ID, "vcs-github", "github")

	require.NoError(t, repository.Insert(context.TODO(), db, &sdk.ProjectRepository{
		Name:         "my/repo",
		ID:           sdk.UUID(),
		CreatedBy:    "me",
		Created:      time.Now(),
		VCSProjectID: vcsProj.ID,
	}))

	vars := map[string]string{}
	uri := api.Router.GetRouteV2("GET", api.getAllRepositoriesHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var repositories []sdk.ProjectRepository
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &repositories))
	require.Len(t, repositories, 1)

}
