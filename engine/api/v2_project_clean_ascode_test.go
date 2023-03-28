package api

import (
	"context"
	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func Test_cleanAsCodeEntities(t *testing.T) {
	api, db, _ := newTestAPI(t)

	// Create project
	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	// Create VCS
	vcsProject := &sdk.VCSProject{
		Name:        "the-name",
		Type:        "github",
		Auth:        sdk.VCSAuthProject{Username: "the-username", Token: "the-token"},
		Description: "the-username",
		ProjectID:   p.ID,
	}
	err := vcs.Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)
	require.NotEmpty(t, vcsProject.ID)

	// Create repository
	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		CloneURL:     "myurl",
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	etoDelete := sdk.Entity{
		Name:                "model1",
		Commit:              "123456",
		Branch:              "temp",
		Type:                sdk.EntityTypeWorkerModel,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data: `name: model1
type: docker
osarch: linux/amd64
spec:
  image: monimage`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &etoDelete))

	etoKeep := sdk.Entity{
		Name:                "model2",
		Commit:              "987654",
		Branch:              "master",
		Type:                sdk.EntityTypeWorkerModel,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data: `name: model1
type: docker
osarch: linux/amd64
spec:
  image: monimage`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &etoKeep))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/the-name/repos/myrepo/branches?limit=50", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				branches := []sdk.VCSBranch{
					{
						DisplayID: "master",
					},
				}
				*(out.(*[]sdk.VCSBranch)) = branches
				return nil, 200, nil
			},
		).MaxTimes(1)

	err = workerCleanProject(context.TODO(), db.DbMap, api.Cache, p.Key)
	require.NoError(t, err)

	_, err = entity.LoadByBranchTypeName(context.TODO(), db, repo.ID, "temp", sdk.EntityTypeWorkerModel, "model1")
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))

	e, err := entity.LoadByBranchTypeName(context.TODO(), db, repo.ID, "master", sdk.EntityTypeWorkerModel, "model2")
	require.NoError(t, err)
	require.Equal(t, etoKeep.ID, e.ID)
}
