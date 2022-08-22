package api

import (
	"context"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCleanAnalyze(t *testing.T) {
	api, db, _ := newTestAPI(t)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	vcsProject := &sdk.VCSProject{
		Name:        "the-name",
		Type:        "github",
		Auth:        sdk.VCSAuthProject{Username: "the-username", Token: "the-token"},
		Description: "the-username",
		ProjectID:   proj1.ID,
	}

	err := vcs.Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)
	require.NotEmpty(t, vcsProject.ID)

	repo := sdk.ProjectRepository{
		Name: "myrepo",
		Auth: sdk.ProjectRepositoryAuth{
			Username: "myuser",
			Token:    "mytoken",
		},
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		CloneURL:     "myurl",
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	for i := 0; i < 60; i++ {
		a := sdk.ProjectRepositoryAnalyze{
			ProjectRepositoryID: repo.ID,
			ProjectKey:          proj1.Key,
			VCSProjectID:        vcsProject.ID,
		}
		require.NoError(t, repository.InsertAnalyze(context.TODO(), db, &a))
	}
	api.cleanRepositoyAnalyzis(ctx, 1*time.Second)

	analyzes, err := repository.LoadAllAnalyzesByRepo(context.TODO(), db, repo.ID)
	require.NoError(t, err)
	require.Len(t, analyzes, 50)
}
