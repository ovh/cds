package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestDecryption(t *testing.T) {
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

	repo1, err := repository.LoadRepositoryByName(context.TODO(), db, vcsProject.ID, repo.Name)
	require.NoError(t, err)
	require.Equal(t, "", repo1.Auth.Token)

	repo2, err := repository.LoadRepositoryByName(context.TODO(), db, vcsProject.ID, repo.Name, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, "mytoken", repo2.Auth.Token)

}
