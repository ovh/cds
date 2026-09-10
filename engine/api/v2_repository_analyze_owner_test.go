package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

// mockAnalysisServices registers a VCS and a hooks service and routes every service call to a mock.
func mockAnalysisServices(t *testing.T, db *test.FakeTransaction) (*mock_services.MockClient, func()) {
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	ctrl := gomock.NewController(t)
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client { return servicesClients }
	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()
	return servicesClients, func() {
		ctrl.Finish()
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}
}

// mockRepositoryWithOneWorkerModel serves, on vcs-server/myrepo at commit abcdef, a repository whose
// only entity is the worker model .cds/worker-models/mymodels.yml.
func mockRepositoryWithOneWorkerModel(servicesClients *mock_services.MockClient, commit sdk.VCSCommit) string {
	model := `
    name: docker-debian
    description: my debian worker model
    osarch: linux/amd64
    type: docker
    spec:
      image: myimage:1.1
  `
	encodedModel := base64.StdEncoding.EncodeToString([]byte(model))

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			*(out.(*sdk.VCSCommit)) = commit
			return nil, 200, nil
		}).AnyTimes()
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			*(out.(*[]sdk.VCSContent)) = []sdk.VCSContent{{IsDirectory: true, Name: "worker-models"}}
			return nil, 200, nil
		}).AnyTimes()
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworker-models?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			*(out.(*[]sdk.VCSContent)) = []sdk.VCSContent{{IsFile: true, Name: "mymodels.yml"}}
			return nil, 200, nil
		}).AnyTimes()
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/content/.cds%2Fworker-models%2Fmymodels.yml?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			*(out.(*sdk.VCSContent)) = sdk.VCSContent{IsFile: true, Name: "mymodels.yml", Content: encodedModel}
			return nil, 200, nil
		}).AnyTimes()
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=&default=true&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			*(out.(*sdk.VCSBranch)) = sdk.VCSBranch{ID: "refs/heads/master", Default: true, LatestCommit: "abcdef"}
			return nil, 200, nil
		}).AnyTimes()
	return model
}

func newAnalysisOfMyRepo(t *testing.T, db *test.FakeTransaction, proj *sdk.Project, vcsProject *sdk.VCSProject, repo *sdk.ProjectRepository, initiator *sdk.V2Initiator) sdk.ProjectRepositoryAnalysis {
	analysis := sdk.ProjectRepositoryAnalysis{
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
		Data:                sdk.ProjectRepositoryData{Initiator: initiator},
	}
	require.NoError(t, repository.InsertAnalysis(context.TODO(), db, &analysis))
	return analysis
}

// A commit signed with the GPG key of the VCS account configured on the project, and committed by that
// account, gives the entity a VCS owner: no CDS user nor user link is involved.
func TestAnalyzeGithubVCSUserOwnsTheEntity(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// The signing key must be unknown as a user key and unique as a project key; other tests expect
	// it to be unknown as a project key too, so it is removed once this one is done
	if uk, err := user.LoadGPGKeyByKeyID(ctx, db, testGPGKeyID); err == nil && uk != nil {
		require.NoError(t, user.DeleteGPGKey(db, *uk))
	}
	deleteProjectKey := func() {
		_, err := db.Exec("DELETE FROM project_key WHERE long_key_id = $1", testGPGKeyID)
		require.NoError(t, err)
	}
	deleteProjectKey()
	t.Cleanup(deleteProjectKey)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)
	bot := "bot-" + sdk.RandomString(6)
	botID := sdk.RandomString(10)

	require.NoError(t, project.InsertKey(db, &sdk.ProjectKey{
		Name:      "vcs-bot-gpg",
		Type:      sdk.KeyTypePGP,
		ProjectID: proj1.ID,
		KeyID:     testGPGKeyID,
		LongKeyID: testGPGKeyID,
		Public:    "public",
		Private:   "private",
	}))
	vcsProject := &sdk.VCSProject{
		Name:         "vcs-server",
		Type:         sdk.VCSTypeGithub,
		ProjectID:    proj1.ID,
		Created:      time.Now(),
		LastModified: time.Now(),
		CreatedBy:    "test",
		Auth:         sdk.VCSAuthProject{Username: bot, Token: "token", GPGKeyName: "vcs-bot-gpg"},
	}
	require.NoError(t, vcs.Insert(ctx, db, vcsProject))
	repo := sdk.ProjectRepository{Name: "myrepo", Created: time.Now(), VCSProjectID: vcsProject.ID, CreatedBy: "me", ProjectKey: proj1.Key}
	require.NoError(t, repository.Insert(ctx, db, &repo))

	// The bot account may manage worker models
	require.NoError(t, rbac.Insert(ctx, db, &sdk.RBAC{
		Name: sdk.RandomString(10),
		Projects: []sdk.RBACProject{{
			Role:            sdk.ProjectRoleManageWorkerModel,
			RBACProjectKeys: []string{proj1.Key},
			RBACVCSUsers:    sdk.RBACVCSUsers{{VCSServer: "vcs-server", VCSUsername: bot}},
		}},
	}))

	analysis := newAnalysisOfMyRepo(t, db, proj1, vcsProject, &repo, nil)
	servicesClients, cleanup := mockAnalysisServices(t, db)
	defer cleanup()
	model := mockRepositoryWithOneWorkerModel(servicesClients, sdk.VCSCommit{
		Hash:      "abcdef",
		Signature: testGPGSignature,
		Verified:  true,
		Author:    sdk.VCSAuthor{Name: bot, ID: botID},
		Committer: sdk.VCSAuthor{Name: bot, ID: botID},
	})

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, analysisUpdated.Status, analysisUpdated.Data.Error)
	require.Equal(t, bot, analysisUpdated.Data.Initiator.VCSUsername)

	es, err := entity.LoadByTypeAndRefCommit(ctx, db, repo.ID, sdk.EntityTypeWorkerModel, "refs/heads/master", "abcdef")
	require.NoError(t, err)
	require.Len(t, es, 1)
	require.Equal(t, model, es[0].Data)
	require.Equal(t, "vcs-server", es[0].Initiator.VCS)
	require.Equal(t, bot, es[0].Initiator.VCSUsername)
	require.Empty(t, es[0].Initiator.UserID)
	require.False(t, es[0].Initiator.IsAdminWithMFA)
	require.Nil(t, es[0].DeprecatedUserID)
}

// An analysis run by an admin with MFA bypasses the role checks but the entity owner keeps no privilege.
func TestAnalyzeGithubAdminSudoOwnsTheEntityWithoutPrivileges(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", sdk.VCSTypeGithub)
	repo := sdk.ProjectRepository{Name: "myrepo", Created: time.Now(), VCSProjectID: vcsProject.ID, CreatedBy: "me", ProjectKey: proj1.Key}
	require.NoError(t, repository.Insert(ctx, db, &repo))
	admin, _ := assets.InsertAdminUser(t, db)

	analysis := newAnalysisOfMyRepo(t, db, proj1, vcsProject, &repo, &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true})
	servicesClients, cleanup := mockAnalysisServices(t, db)
	defer cleanup()
	mockRepositoryWithOneWorkerModel(servicesClients, sdk.VCSCommit{Hash: "abcdef"})

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, analysisUpdated.Status, analysisUpdated.Data.Error)

	es, err := entity.LoadByTypeAndRefCommit(ctx, db, repo.ID, sdk.EntityTypeWorkerModel, "refs/heads/master", "abcdef")
	require.NoError(t, err)
	require.Len(t, es, 1)
	require.Equal(t, admin.ID, es[0].Initiator.UserID)
	require.Equal(t, admin.Username, es[0].Initiator.User.Username)
	require.False(t, es[0].Initiator.IsAdminWithMFA)
	require.Equal(t, admin.ID, *es[0].DeprecatedUserID)
}
