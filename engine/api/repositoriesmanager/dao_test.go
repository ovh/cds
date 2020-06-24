package repositoriesmanager_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func Test_CRUDProjectVCSServerLink(t *testing.T) {
	db, _ := test.SetupPG(t)

	pk := sdk.RandomString(8)

	proj := sdk.Project{
		Key:  pk,
		Name: pk,
	}
	require.NoError(t, project.Insert(db, &proj))

	vcsServerForProject := &sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "test_github",
		Username:  "test",
	}
	vcsServerForProject.Set("token", "token")
	vcsServerForProject.Set("secret", "secret")
	vcsServerForProject.Set("created", strconv.FormatInt(time.Now().Unix(), 10))

	err := repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, vcsServerForProject)
	require.NoError(t, err, "InsertProjectVCSServerLink should succeed but failed with %v", err)
	encryptedToken, _ := vcsServerForProject.Get("token")
	assert.Equal(t, sdk.PasswordPlaceholder, encryptedToken)

	vcsServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(context.TODO(), db, pk, "test_github")
	require.NoError(t, err, "LoadProjectVCSServerLinkByProjectKeyAndVCSServerName should succeed but failed with %v", err)

	vcsServer.Set("token", "token2")
	err = repositoriesmanager.UpdateProjectVCSServerLink(context.TODO(), db, &vcsServer)
	require.NoError(t, err, "UpdateProjectVCSServerLink should succeed but failed with %v", err)

	repoData, err := repositoriesmanager.LoadProjectVCSServerLinksData(context.TODO(), db, vcsServer.ID, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err, "LoadProjectVCSServerLinksData should succeed but failed with %v", err)
	vcsServer.ProjectVCSServerLinkData = repoData
	decryptedToken, found := vcsServer.Get("token")
	assert.True(t, found)
	assert.Equal(t, "token2", decryptedToken)

	vcsservers, err := repositoriesmanager.LoadAllProjectVCSServerLinksByProjectID(context.TODO(), db, proj.ID)
	require.NoError(t, err, "LoadAllProjectVCSServerLinksByProjectID should succeed but failed with %v", err)
	assert.Len(t, vcsservers, 1, "LoadAllProjectVCSServerLinksByProjectID should return one vcsservers")

	vcsservers, err = repositoriesmanager.LoadAllProjectVCSServerLinksByProjectKey(context.TODO(), db, proj.Key)
	require.NoError(t, err, "LoadAllProjectVCSServerLinksByProjectKey should succeed but failed with %v", err)
	assert.Len(t, vcsservers, 1, "LoadAllProjectVCSServerLinksByProjectKey should return one vcsservers")

	err = repositoriesmanager.DeleteProjectVCSServerLink(context.TODO(), db, &vcsservers[0])
	require.NoError(t, err, "DeleteProjectVCSServerLink should succeed but failed with %v", err)

}
