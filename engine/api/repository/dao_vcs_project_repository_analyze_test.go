package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestDeleteOldest(t *testing.T) {
	db, cache := test.SetupPG(t)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, cache, key1, key1)

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

	for i := 0; i < 50; i++ {
		a := sdk.ProjectRepositoryAnalyze{
			ProjectRepositoryID: repo.ID,
		}
		require.NoError(t, repository.InsertAnalyze(context.TODO(), db, &a))
	}

	analyzes, err := repository.ListAnalyzesByRepo(context.TODO(), db, repo.ID)
	require.NoError(t, err)
	require.Len(t, analyzes, 50)

	// Add 51
	a := sdk.ProjectRepositoryAnalyze{
		ProjectRepositoryID: repo.ID,
		Branch:              "lastbranch",
	}
	require.NoError(t, repository.InsertAnalyze(context.TODO(), db, &a))

	analyzes, err = repository.ListAnalyzesByRepo(context.TODO(), db, repo.ID)
	require.NoError(t, err)
	require.Len(t, analyzes, 50)
	require.Equal(t, "lastbranch", analyzes[0].Branch)

}
