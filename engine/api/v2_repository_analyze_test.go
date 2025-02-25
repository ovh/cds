package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/ovh/cds/engine/api/workflow_v2"

	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/sdk"
)

func TestSortEntitiesFiles(t *testing.T) {
	filesContext := map[string][]byte{
		".cds/workflows/w1.yml":           nil,
		".cds/actions/act1.yml":           nil,
		".cds/workflow-templates/wt2.yml": nil,
		".cds/worker-models/wm1.yml":      nil,
		".cds/worker-models/wm2.yml":      nil,
		".cds/workflows/w2.yml":           nil,
		".cds/actions/act2.yml":           nil,
		".cds/workflow-templates/wt1.yml": nil,
	}
	keys := sortEntitiesFiles(filesContext)
	require.Equal(t, ".cds/worker-models/wm1.yml", keys[0])
	require.Equal(t, ".cds/worker-models/wm2.yml", keys[1])
	require.Equal(t, ".cds/actions/act1.yml", keys[2])
	require.Equal(t, ".cds/actions/act2.yml", keys[3])
	require.Equal(t, ".cds/workflow-templates/wt1.yml", keys[4])
	require.Equal(t, ".cds/workflow-templates/wt2.yml", keys[5])
	require.Equal(t, ".cds/workflows/w1.yml", keys[6])
	require.Equal(t, ".cds/workflows/w2.yml", keys[7])

}
func TestCleanAnalysis(t *testing.T) {
	api, db, _ := newTestAPI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	vcsProject := &sdk.VCSProject{
		Name:        "the-name",
		Type:        sdk.VCSTypeGithub,
		Auth:        sdk.VCSAuthProject{Username: "the-username", Token: "the-token"},
		Description: "the-username",
		ProjectID:   proj1.ID,
	}

	err := vcs.Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)
	require.NotEmpty(t, vcsProject.ID)

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		CloneURL:     "myurl",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	for i := 0; i < 60; i++ {
		a := sdk.ProjectRepositoryAnalysis{
			ProjectRepositoryID: repo.ID,
			ProjectKey:          proj1.Key,
			VCSProjectID:        vcsProject.ID,
		}
		require.NoError(t, repository.InsertAnalysis(context.TODO(), db, &a))
	}
	api.cleanRepositoryAnalysis(ctx, 1*time.Second)

	analyses, err := repository.LoadAnalysesByRepo(context.TODO(), db, repo.ID)
	require.NoError(t, err)
	require.Len(t, analyses, 50)
}

func TestAnalyzeGithubWithoutHash(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "fakesign",
					Verified:  true,
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, "unable to check the commit signature", analysisUpdated.Data.Error)
	require.Equal(t, sdk.RepositoryAnalysisStatusError, analysisUpdated.Status)
}

func TestAnalyzeGithubWrongSignature(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "fakesign",
					Verified:  true,
					Hash:      "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusError, analysisUpdated.Status)
	require.Contains(t, analysisUpdated.Data.Error, "unable to check the commit signature")
}

func TestAnalyzeGithubGPGKeyNotFound(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	userKey, err := user.LoadGPGKeyByKeyID(ctx, db, "F344BDDCE15F17D7")
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		require.NoError(t, err)
	}
	if userKey != nil {
		require.NoError(t, user.DeleteGPGKey(db, *userKey))
	}

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
		Data: sdk.ProjectRepositoryData{
			Initiator: &sdk.V2Initiator{
				VCS:         "vcs-server",
				VCSUsername: "my-githup-username",
			},
		},
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "-----BEGIN PGP SIGNATURE-----\n\niQIzBAABCAAdFiEEfYJxMHx+E0DPuqaA80S93OFfF9cFAmME7aIACgkQ80S93OFf\nF9eFWBAAq5hOcZIx/A+8J6/NwRtXMs5OW+TJxzJb5siXdRC8Mjrm+fqwpTPPHqtB\nbb7iuiRnmY/HqCegULiw4qVxDyA3sswyDHPLcyUcfG4drJGylPW9ZYg3YeRslX2B\niQykYZyd4h3R/euYAuBKA9vMGoWnaU/Vh22A11Po1pXpPq623FTkiFOSAZrD8Hql\nEvmlhw26qHSPlhsdSKsR+/FPvpLUXlNUiYB5oq7W9qy0yOOafgwZ9r3vvxshzvkt\nvW5zG+R05thQ8icCyrWfEfIWp+TTtQX3asOopnQG9dFs2LRODLXXaHTRVRB/MWPa\nNVvUD/dIzBVyNimpik+2Uqq5jWNiXavQmqoxyL9n4A372AIH7Hu78NnfmAz7VnYo\nyVHRNBryiCcYNj5g0x/WnGsDuhQr7170ODw7QfEYJdCPxGgYuhdYovHdjcMcgWpF\ncWEtayj8bhuLTjjxEsqXTv+psxwB55N5OUvyXmNAaFLhJSEI+l1VHW14L3gZFdPT\n+VgPQtT9a1+GEjPqLvZ6wLVTcSI9uogK6NHowmyM261FtFQqLVdkOdUU8RCR8qLC\nekZWQaJutqicIZTolAQyBPBw8aQz0i+uBUgdWkoiHf/zEEudu0b06IpDq2oYFFVH\nVmCuZ3/AcXrW6T3XXcE5pu+Rvsi57O7iR8i7TIP0CaDTr2FfQWc=\n=/H7t\n-----END PGP SIGNATURE-----",
					Verified:  true,
					Hash:      "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(2)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSkipped, analysisUpdated.Status)
	require.Equal(t, "unable to find commiter for commit abcdef", analysisUpdated.Data.Error)
}

func TestAnalyzeGithubUserNotEnoughPerm(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	uk, err := user.LoadGPGKeyByKeyID(ctx, db, "F344BDDCE15F17D7")
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		require.NoError(t, err)
	}
	if uk != nil {
		require.NoError(t, user.DeleteGPGKey(db, *uk))
	}

	u, _ := assets.InsertLambdaUser(t, db)
	userKey := &sdk.UserGPGKey{
		KeyID: "F344BDDCE15F17D7",
		PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
		AuthentifiedUserID: u.ID,
	}
	require.NoError(t, user.InsertGPGKey(ctx, db, userKey))

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "-----BEGIN PGP SIGNATURE-----\n\niQIzBAABCAAdFiEEfYJxMHx+E0DPuqaA80S93OFfF9cFAmME7aIACgkQ80S93OFf\nF9eFWBAAq5hOcZIx/A+8J6/NwRtXMs5OW+TJxzJb5siXdRC8Mjrm+fqwpTPPHqtB\nbb7iuiRnmY/HqCegULiw4qVxDyA3sswyDHPLcyUcfG4drJGylPW9ZYg3YeRslX2B\niQykYZyd4h3R/euYAuBKA9vMGoWnaU/Vh22A11Po1pXpPq623FTkiFOSAZrD8Hql\nEvmlhw26qHSPlhsdSKsR+/FPvpLUXlNUiYB5oq7W9qy0yOOafgwZ9r3vvxshzvkt\nvW5zG+R05thQ8icCyrWfEfIWp+TTtQX3asOopnQG9dFs2LRODLXXaHTRVRB/MWPa\nNVvUD/dIzBVyNimpik+2Uqq5jWNiXavQmqoxyL9n4A372AIH7Hu78NnfmAz7VnYo\nyVHRNBryiCcYNj5g0x/WnGsDuhQr7170ODw7QfEYJdCPxGgYuhdYovHdjcMcgWpF\ncWEtayj8bhuLTjjxEsqXTv+psxwB55N5OUvyXmNAaFLhJSEI+l1VHW14L3gZFdPT\n+VgPQtT9a1+GEjPqLvZ6wLVTcSI9uogK6NHowmyM261FtFQqLVdkOdUU8RCR8qLC\nekZWQaJutqicIZTolAQyBPBw8aQz0i+uBUgdWkoiHf/zEEudu0b06IpDq2oYFFVH\nVmCuZ3/AcXrW6T3XXcE5pu+Rvsi57O7iR8i7TIP0CaDTr2FfQWc=\n=/H7t\n-----END PGP SIGNATURE-----",
					Verified:  true,
					Hash:      "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: true,
						Name:        "worker-models",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)

	modelContent := `
    name: docker-debian
    description: my debian worker model
    osarch: linux/amd64
    type: docker
    spec:
      image: myimage:1.1
      envs:
        MYVAR: toto
  `
	content64 := base64.StdEncoding.EncodeToString([]byte(modelContent))
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/content/.cds%2Fworker-models%2Fmymodel.yml?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				content := sdk.VCSContent{
					IsFile:  true,
					Name:    "mymodel.yml",
					Content: content64,
				}
				*(out.(*sdk.VCSContent)) = content
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworker-models?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsFile: true,
						Name:   "mymodel.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=&default=true&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := sdk.VCSBranch{
					ID: "refs/heads/master",
				}
				*(out.(*sdk.VCSBranch)) = contents
				return nil, 200, nil
			},
		)
	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSkipped, analysisUpdated.Status)
	require.Contains(t, analysisUpdated.Data.Error, "User doesn't have the permission to manage WorkerModel")
}

func TestAnalyzeGithubServerCommitNotSigned(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Hash: "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSkipped, analysisUpdated.Status)
	require.Equal(t, "commit abcdef is not signed", analysisUpdated.Data.Error)
}

func TestAnalyzeGithubAddWorkerModel(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	uk, err := user.LoadGPGKeyByKeyID(ctx, db, "F344BDDCE15F17D7")
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		require.NoError(t, err)
	}
	if uk != nil {
		require.NoError(t, user.DeleteGPGKey(db, *uk))
	}

	u, _ := assets.InsertLambdaUser(t, db)
	userKey := &sdk.UserGPGKey{
		KeyID: "F344BDDCE15F17D7",
		PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
		AuthentifiedUserID: u.ID,
	}
	require.NoError(t, user.InsertGPGKey(ctx, db, userKey))

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManageWorkerModel, proj1.Key, *u)

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	model := `
    name: docker-debian
    description: my debian worker model
    osarch: linux/amd64
    type: docker
    spec:
      image: myimage:1.1
      envs:
        MYVAR: toto
  `
	encodedModel := base64.StdEncoding.EncodeToString([]byte(model))

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "-----BEGIN PGP SIGNATURE-----\n\niQIzBAABCAAdFiEEfYJxMHx+E0DPuqaA80S93OFfF9cFAmME7aIACgkQ80S93OFf\nF9eFWBAAq5hOcZIx/A+8J6/NwRtXMs5OW+TJxzJb5siXdRC8Mjrm+fqwpTPPHqtB\nbb7iuiRnmY/HqCegULiw4qVxDyA3sswyDHPLcyUcfG4drJGylPW9ZYg3YeRslX2B\niQykYZyd4h3R/euYAuBKA9vMGoWnaU/Vh22A11Po1pXpPq623FTkiFOSAZrD8Hql\nEvmlhw26qHSPlhsdSKsR+/FPvpLUXlNUiYB5oq7W9qy0yOOafgwZ9r3vvxshzvkt\nvW5zG+R05thQ8icCyrWfEfIWp+TTtQX3asOopnQG9dFs2LRODLXXaHTRVRB/MWPa\nNVvUD/dIzBVyNimpik+2Uqq5jWNiXavQmqoxyL9n4A372AIH7Hu78NnfmAz7VnYo\nyVHRNBryiCcYNj5g0x/WnGsDuhQr7170ODw7QfEYJdCPxGgYuhdYovHdjcMcgWpF\ncWEtayj8bhuLTjjxEsqXTv+psxwB55N5OUvyXmNAaFLhJSEI+l1VHW14L3gZFdPT\n+VgPQtT9a1+GEjPqLvZ6wLVTcSI9uogK6NHowmyM261FtFQqLVdkOdUU8RCR8qLC\nekZWQaJutqicIZTolAQyBPBw8aQz0i+uBUgdWkoiHf/zEEudu0b06IpDq2oYFFVH\nVmCuZ3/AcXrW6T3XXcE5pu+Rvsi57O7iR8i7TIP0CaDTr2FfQWc=\n=/H7t\n-----END PGP SIGNATURE-----",
					Verified:  true,
					Hash:      "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: true,
						Name:        "worker-models",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworker-models?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: false,
						IsFile:      true,
						Name:        "mymodels.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/content/.cds%2Fworker-models%2Fmymodels.yml?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {

				content := sdk.VCSContent{
					IsDirectory: false,
					IsFile:      true,
					Name:        "mymodels.yml",
					Content:     encodedModel,
				}
				*(out.(*sdk.VCSContent)) = content
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=&default=true&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := sdk.VCSBranch{
					ID: "refs/heads/master",
				}
				*(out.(*sdk.VCSBranch)) = contents
				return nil, 200, nil
			},
		)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, analysisUpdated.Status)

	es, err := entity.LoadByTypeAndRefCommit(context.TODO(), db, repo.ID, sdk.EntityTypeWorkerModel, "refs/heads/master", "abcdef")
	require.NoError(t, err)

	require.Equal(t, 1, len(es))
	require.Equal(t, model, es[0].Data)
	t.Logf("%+v", es[0])

	e, err := entity.LoadByRefTypeNameCommit(context.TODO(), db, repo.ID, "refs/heads/master", sdk.EntityTypeWorkerModel, "docker-debian", "abcdef")
	require.NoError(t, err)
	require.Equal(t, model, e.Data)
}

func TestAnalyzeGithubMergeCommitNoLink(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	githubUsername := sdk.RandomString(10)
	api.Config.VCS.GPGKeys = map[string][]GPGKey{
		"vcs-server": {
			{
				ID: "F344BDDCE15F17D7",
				PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
			},
		},
	}

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	uk, err := user.LoadGPGKeyByKeyID(ctx, db, "F344BDDCE15F17D7")
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		require.NoError(t, err)
	}
	if uk != nil {
		require.NoError(t, user.DeleteGPGKey(db, *uk))
	}

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "-----BEGIN PGP SIGNATURE-----\n\niQIzBAABCAAdFiEEfYJxMHx+E0DPuqaA80S93OFfF9cFAmME7aIACgkQ80S93OFf\nF9eFWBAAq5hOcZIx/A+8J6/NwRtXMs5OW+TJxzJb5siXdRC8Mjrm+fqwpTPPHqtB\nbb7iuiRnmY/HqCegULiw4qVxDyA3sswyDHPLcyUcfG4drJGylPW9ZYg3YeRslX2B\niQykYZyd4h3R/euYAuBKA9vMGoWnaU/Vh22A11Po1pXpPq623FTkiFOSAZrD8Hql\nEvmlhw26qHSPlhsdSKsR+/FPvpLUXlNUiYB5oq7W9qy0yOOafgwZ9r3vvxshzvkt\nvW5zG+R05thQ8icCyrWfEfIWp+TTtQX3asOopnQG9dFs2LRODLXXaHTRVRB/MWPa\nNVvUD/dIzBVyNimpik+2Uqq5jWNiXavQmqoxyL9n4A372AIH7Hu78NnfmAz7VnYo\nyVHRNBryiCcYNj5g0x/WnGsDuhQr7170ODw7QfEYJdCPxGgYuhdYovHdjcMcgWpF\ncWEtayj8bhuLTjjxEsqXTv+psxwB55N5OUvyXmNAaFLhJSEI+l1VHW14L3gZFdPT\n+VgPQtT9a1+GEjPqLvZ6wLVTcSI9uogK6NHowmyM261FtFQqLVdkOdUU8RCR8qLC\nekZWQaJutqicIZTolAQyBPBw8aQz0i+uBUgdWkoiHf/zEEudu0b06IpDq2oYFFVH\nVmCuZ3/AcXrW6T3XXcE5pu+Rvsi57O7iR8i7TIP0CaDTr2FfQWc=\n=/H7t\n-----END PGP SIGNATURE-----",
					Verified:  true,
					Hash:      "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			commit := &sdk.VCSCommit{
				Committer: sdk.VCSAuthor{
					Name: githubUsername,
					ID:   "1234",
				},
			}
			*(out.(*sdk.VCSCommit)) = *commit
			return nil, 200, nil
		},
		).MaxTimes(1)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	t.Logf("%+v", analysisUpdated.Data)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSkipped, analysisUpdated.Status)
	require.Equal(t, fmt.Sprintf("github user %s not found in CDS", githubUsername), analysisUpdated.Data.Error)
}

func TestAnalyzeGithubAddWorkerModelMergeCommit(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	githubUsername := sdk.RandomString(10)
	api.Config.VCS.GPGKeys = map[string][]GPGKey{
		"vcs-server": {
			{
				ID: "F344BDDCE15F17D7",
				PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
			},
		},
	}

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	uk, err := user.LoadGPGKeyByKeyID(ctx, db, "F344BDDCE15F17D7")
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		require.NoError(t, err)
	}
	if uk != nil {
		require.NoError(t, user.DeleteGPGKey(db, *uk))
	}

	u, _ := assets.InsertLambdaUser(t, db)
	ul := sdk.UserLink{
		Type:               "github",
		AuthentifiedUserID: u.ID,
		Username:           githubUsername,
		ExternalID:         sdk.RandomString(10),
	}
	require.NoError(t, link.Insert(context.TODO(), db, &ul))

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManageWorkerModel, proj1.Key, *u)

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	model := `name: docker-debian
description: my debian worker model
type: docker
osarch: linux/amd64
spec:
  image: myimage:1.1
  envs:
    MYVAR: toto

`
	encodedModel := base64.StdEncoding.EncodeToString([]byte(model))

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "-----BEGIN PGP SIGNATURE-----\n\niQIzBAABCAAdFiEEfYJxMHx+E0DPuqaA80S93OFfF9cFAmME7aIACgkQ80S93OFf\nF9eFWBAAq5hOcZIx/A+8J6/NwRtXMs5OW+TJxzJb5siXdRC8Mjrm+fqwpTPPHqtB\nbb7iuiRnmY/HqCegULiw4qVxDyA3sswyDHPLcyUcfG4drJGylPW9ZYg3YeRslX2B\niQykYZyd4h3R/euYAuBKA9vMGoWnaU/Vh22A11Po1pXpPq623FTkiFOSAZrD8Hql\nEvmlhw26qHSPlhsdSKsR+/FPvpLUXlNUiYB5oq7W9qy0yOOafgwZ9r3vvxshzvkt\nvW5zG+R05thQ8icCyrWfEfIWp+TTtQX3asOopnQG9dFs2LRODLXXaHTRVRB/MWPa\nNVvUD/dIzBVyNimpik+2Uqq5jWNiXavQmqoxyL9n4A372AIH7Hu78NnfmAz7VnYo\nyVHRNBryiCcYNj5g0x/WnGsDuhQr7170ODw7QfEYJdCPxGgYuhdYovHdjcMcgWpF\ncWEtayj8bhuLTjjxEsqXTv+psxwB55N5OUvyXmNAaFLhJSEI+l1VHW14L3gZFdPT\n+VgPQtT9a1+GEjPqLvZ6wLVTcSI9uogK6NHowmyM261FtFQqLVdkOdUU8RCR8qLC\nekZWQaJutqicIZTolAQyBPBw8aQz0i+uBUgdWkoiHf/zEEudu0b06IpDq2oYFFVH\nVmCuZ3/AcXrW6T3XXcE5pu+Rvsi57O7iR8i7TIP0CaDTr2FfQWc=\n=/H7t\n-----END PGP SIGNATURE-----",
					Verified:  true,
					Hash:      "abcdef",
					Committer: sdk.VCSAuthor{
						Name: githubUsername,
						ID:   ul.ExternalID,
					},
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(2)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: true,
						Name:        "worker-models",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworker-models?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: false,
						IsFile:      true,
						Name:        "mymodels.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/content/.cds%2Fworker-models%2Fmymodels.yml?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {

				content := sdk.VCSContent{
					IsDirectory: false,
					IsFile:      true,
					Name:        "mymodels.yml",
					Content:     encodedModel,
				}
				*(out.(*sdk.VCSContent)) = content
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=&default=true&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := sdk.VCSBranch{
					ID: "refs/heads/master",
				}
				*(out.(*sdk.VCSBranch)) = contents
				return nil, 200, nil
			},
		)
	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, analysisUpdated.Status)

	es, err := entity.LoadByTypeAndRefCommit(context.TODO(), db, repo.ID, sdk.EntityTypeWorkerModel, "refs/heads/master", "abcdef")
	require.NoError(t, err)

	require.Equal(t, 1, len(es))
	require.Equal(t, model, es[0].Data)
	t.Logf("%+v", es[0])

	e, err := entity.LoadByRefTypeNameCommit(context.TODO(), db, repo.ID, "refs/heads/master", sdk.EntityTypeWorkerModel, "docker-debian", "abcdef")
	require.NoError(t, err)
	require.Equal(t, model, e.Data)
}

func TestAnalyzeBitbucketMergeCommit(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	_, _ = db.Exec("DELETE FROM service")

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	api.Config.VCS.GPGKeys = map[string][]GPGKey{
		"vcs-server": {
			{
				ID: "F344BDDCE15F17D7",
				PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
			},
		},
	}

	u, _ := assets.InsertLambdaUser(t, db)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManageWorkerModel, proj1.Key, *u)

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", sdk.VCSTypeBitbucketServer)

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
		Data: sdk.ProjectRepositoryData{
			OperationUUID: sdk.UUID(),
		},
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	model := `
    name: docker-debian
    description: my debian worker model
    osarch: linux/amd64
    type: docker
    spec:
      image: myimage:1.1
      envs:
        MYVAR: toto
  `

	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Name: ".cds/worker-models/model.yml",
		Mode: 0755,
		Size: int64(len([]byte(model))),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte(model))
	require.NoError(t, err)
	tw.Close()
	gw.Close()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef&offset=0&limit=1", gomock.Any(), gomock.Any(), gomock.Any())

	servicesClients.EXPECT().
		StreamRequest(gomock.Any(), "POST", "/vcs/vcs-server/repos/myrepo/archive", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, _ interface{}) (io.Reader, http.Header, int, error) {
				return bytes.NewReader(buf.Bytes()), nil, 200, nil
			},
		)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Hash: "abcdef",
					Committer: sdk.VCSAuthor{
						Name:        u.Username,
						Slug:        u.Username,
						DisplayName: u.Username,
						Email:       u.GetEmail(),
					},
					Verified: true,
					KeyID:    "F344BDDCE15F17D7",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: true,
						Name:        "worker-models",
					},
					{
						IsDirectory: true,
						Name:        "worker-model-templates",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworker-models?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: false,
						IsFile:      true,
						Name:        "mymodels.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=&default=true&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := sdk.VCSBranch{
					ID: "refs/heads/master",
				}
				*(out.(*sdk.VCSBranch)) = contents
				return nil, 200, nil
			},
		)
	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, analysisUpdated.Status)

	es, err := entity.LoadByTypeAndRefCommit(context.TODO(), db, repo.ID, sdk.EntityTypeWorkerModel, "refs/heads/master", "abcdef")
	require.NoError(t, err)

	require.Equal(t, 1, len(es))
	require.Equal(t, model, es[0].Data)

	e, err := entity.LoadByRefTypeNameCommit(context.TODO(), db, repo.ID, "refs/heads/master", sdk.EntityTypeWorkerModel, "docker-debian", "abcdef")
	require.NoError(t, err)
	require.Equal(t, model, e.Data)
}

func TestManageWorkflowHooksAllSameRepo(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE from v2_workflow_hook")
	require.NoError(t, err)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repoDef := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "sgu/myDefRepo")

	//
	e := sdk.EntityWithObject{
		Entity: sdk.Entity{
			ProjectKey:          proj.Key,
			ProjectRepositoryID: repoDef.ID,
			Ref:                 "refs/heads/master",
			Type:                sdk.EntityTypeWorkflow,
			Commit:              "123456",
			Name:                sdk.RandomString(10),
		},
		Workflow: sdk.V2Workflow{
			Repository: &sdk.WorkflowRepository{
				VCSServer: vcsServer.Name,
				Name:      repoDef.Name,
			},
			CommitStatus: &sdk.CommitStatus{
				Title:       "foo",
				Description: "bar",
			},
			On: &sdk.WorkflowOn{
				Push:           &sdk.WorkflowOnPush{},
				WorkflowUpdate: &sdk.WorkflowOnWorkflowUpdate{},
				ModelUpdate: &sdk.WorkflowOnModelUpdate{
					Models: []string{"MyModel"},
				},
				Schedule: []sdk.WorkflowOnSchedule{
					{
						Cron: "* * * * *",
					},
				},
			},
		},
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e.Entity))

	// INSERT OLD SCHEDULER DEFINITION
	oldSche := sdk.V2WorkflowHook{
		ProjectKey:     proj.Key,
		VCSName:        "github",
		RepositoryName: "sgu/mydefrepo",
		EntityID:       e.ID,
		WorkflowName:   e.Name,
		Ref:            "refs/heads/master",
		Commit:         "123456",
		Type:           sdk.WorkflowHookTypeScheduler,
		Data: sdk.V2WorkflowHookData{
			Cron: "1 1 1 1 1",
		},
	}
	require.NoError(t, workflow_v2.InsertWorkflowHook(context.TODO(), db, &oldSche))

	srvs, err := services.LoadAllByType(context.TODO(), api.mustDB(), sdk.TypeHooks)
	require.NoError(t, err)

	s, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer t.Cleanup(func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	})
	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "DELETE", "/v2/workflow/scheduler/github/sgu%2Fmydefrepo/"+e.Name, gomock.Any(), gomock.Any()).Times(1)

	_, err = manageWorkflowHooks(context.TODO(), db, api.Cache, nil, e, "github", "sgu/mydefrepo", &sdk.VCSBranch{ID: "refs/heads/master", LatestCommit: "123456"}, srvs)
	require.NoError(t, err)

	repoWebHooks, err := workflow_v2.LoadHooksByRepositoryEvent(context.TODO(), db, vcsServer.Name, repoDef.Name, "push")
	require.NoError(t, err)
	require.Equal(t, 2, len(repoWebHooks)) // commit + HEAD

	// Local workflow so worklow update hook must not be saved
	_, err = workflow_v2.LoadHooksByWorkflowUpdated(context.TODO(), db, proj.Key, vcsServer.Name, repoDef.Name, e.Name, "123456")
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))

	// Local workflow so model update hook must not be saved
	hooks, err := workflow_v2.LoadHooksByModelUpdated(context.TODO(), db, "123456", []string{"MyModel"})
	require.NoError(t, err)
	require.Equal(t, 0, len(hooks))

	// Check scheduler config
	scheds, err := workflow_v2.LoadHookSchedulerByWorkflow(context.TODO(), db, proj.Key, vcsServer.Name, "sgu/mydefrepo", e.Name)
	require.NoError(t, err)
	require.Equal(t, 1, len(scheds))
	require.Equal(t, "sgu/mydefrepo", scheds[0].Data.RepositoryName)
	require.Equal(t, "sgu/mydefrepo", scheds[0].RepositoryName)
	require.Equal(t, "* * * * *", scheds[0].Data.Cron)
}

func TestManageWorkflowHooksAllDistantEntitiesOndefaultBranch(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE from v2_workflow_hook")
	require.NoError(t, err)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repoDef := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "sgu/myDefRepo")

	//
	e := sdk.EntityWithObject{
		Entity: sdk.Entity{
			ProjectKey:          proj.Key,
			ProjectRepositoryID: repoDef.ID,
			Ref:                 "refs/heads/main",
			Type:                sdk.EntityTypeWorkflow,
			Commit:              "123456",
			Name:                sdk.RandomString(10),
		},
		Workflow: sdk.V2Workflow{
			Repository: &sdk.WorkflowRepository{
				VCSServer: vcsServer.Name,
				Name:      "sgu/myapp",
			},
			On: &sdk.WorkflowOn{
				Push:           &sdk.WorkflowOnPush{},
				WorkflowUpdate: &sdk.WorkflowOnWorkflowUpdate{},
				ModelUpdate: &sdk.WorkflowOnModelUpdate{
					Models: []string{"MyModel"},
				},
				Schedule: []sdk.WorkflowOnSchedule{
					{
						Cron: "0 12 * * 5",
					},
				},
			},
		},
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e.Entity))

	srvs, err := services.LoadAllByType(context.TODO(), api.mustDB(), sdk.TypeHooks)
	require.NoError(t, err)

	_, err = manageWorkflowHooks(context.TODO(), db, api.Cache, nil, e, "github", "sgu/mydefrepo", &sdk.VCSBranch{ID: "refs/heads/main", LatestCommit: "123456"}, srvs)
	require.NoError(t, err)

	repoWebHooks, err := workflow_v2.LoadHooksByRepositoryEvent(context.TODO(), db, vcsServer.Name, "sgu/myapp", "push")
	require.NoError(t, err)
	require.Equal(t, 2, len(repoWebHooks))

	hookWithCommit := false
	hookWithHead := false
	for _, wh := range repoWebHooks {
		if wh.Commit == "123456" {
			hookWithCommit = true
		} else if wh.Commit == "HEAD" {
			hookWithHead = true
		}
	}
	require.True(t, hookWithCommit)
	require.True(t, hookWithHead)

	// Distant workflow so worklow update hook must be saved
	workflowUpdateHooks, err := workflow_v2.LoadHooksByWorkflowUpdated(context.TODO(), db, proj.Key, vcsServer.Name, repoDef.Name, e.Name, "123456")
	require.NoError(t, err)
	require.NotNil(t, workflowUpdateHooks)

	// Distant workflow so model update hook must be saved
	modelKey := fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsServer.Name, repoDef.Name, "MyModel")
	hooks, err := workflow_v2.LoadHooksByModelUpdated(context.TODO(), db, "123456", []string{modelKey})
	require.NoError(t, err)
	require.Equal(t, 1, len(hooks))

	scheds, err := workflow_v2.LoadHookSchedulerByWorkflow(context.TODO(), db, proj.Key, vcsServer.Name, "sgu/mydefrepo", e.Name)
	require.NoError(t, err)
	require.Equal(t, 1, len(scheds))
	require.Equal(t, "sgu/myapp", scheds[0].Data.RepositoryName)
	require.Equal(t, "sgu/mydefrepo", scheds[0].RepositoryName)
}

func TestManageWorkflowHooksAllDistantEntities(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE from v2_workflow_hook")
	require.NoError(t, err)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repoDef := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "sgu/myDefRepo")

	//
	e := sdk.EntityWithObject{
		Entity: sdk.Entity{
			ProjectKey:          proj.Key,
			ProjectRepositoryID: repoDef.ID,
			Ref:                 "refs/heads/main",
			Type:                sdk.EntityTypeWorkflow,
			Commit:              "123456",
			Name:                sdk.RandomString(10),
		},
		Workflow: sdk.V2Workflow{
			Repository: &sdk.WorkflowRepository{
				VCSServer: vcsServer.Name,
				Name:      "sgu/myapp",
			},
			On: &sdk.WorkflowOn{
				Push:           &sdk.WorkflowOnPush{},
				WorkflowUpdate: &sdk.WorkflowOnWorkflowUpdate{},
				ModelUpdate: &sdk.WorkflowOnModelUpdate{
					Models: []string{"MyModel"},
				},
			},
		},
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e.Entity))
	_, err = manageWorkflowHooks(context.TODO(), db, api.Cache, nil, e, "github", "sgu/mydefrepo", &sdk.VCSBranch{ID: "refs/heads/main"}, nil)
	require.NoError(t, err)

	repoWebHooks, err := workflow_v2.LoadHooksByRepositoryEvent(context.TODO(), db, vcsServer.Name, "sgu/myapp", "push")
	require.NoError(t, err)
	require.Equal(t, 1, len(repoWebHooks))

	// Distant workflow so worklow update hook must be saved
	workflowUpdateHooks, err := workflow_v2.LoadHooksByWorkflowUpdated(context.TODO(), db, proj.Key, vcsServer.Name, repoDef.Name, e.Name, "123456")
	require.NoError(t, err)
	require.NotNil(t, workflowUpdateHooks)

	// Distant workflow so model update hook must be saved
	modelKey := fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsServer.Name, repoDef.Name, "MyModel")
	hooks, err := workflow_v2.LoadHooksByModelUpdated(context.TODO(), db, "123456", []string{modelKey})
	require.NoError(t, err)
	require.Equal(t, 1, len(hooks))
}

func TestManageWorkflowHooksAllDistantEntitiesWithModelOnDifferentRepo(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE from v2_workflow_hook")
	require.NoError(t, err)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repoDef := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "sgu/myDefRepo")

	//
	e := sdk.EntityWithObject{
		Entity: sdk.Entity{
			ProjectKey:          proj.Key,
			ProjectRepositoryID: repoDef.ID,
			Ref:                 "refs/heads/main",
			Type:                sdk.EntityTypeWorkflow,
			Commit:              "123456",
			Name:                sdk.RandomString(10),
		},
		Workflow: sdk.V2Workflow{
			Repository: &sdk.WorkflowRepository{
				VCSServer: vcsServer.Name,
				Name:      "sgu/myapp",
			},
			On: &sdk.WorkflowOn{
				Push:           &sdk.WorkflowOnPush{},
				WorkflowUpdate: &sdk.WorkflowOnWorkflowUpdate{},
				ModelUpdate: &sdk.WorkflowOnModelUpdate{
					Models: []string{"sgu/myresources/MyModel"},
				},
			},
		},
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e.Entity))
	_, err = manageWorkflowHooks(context.TODO(), db, api.Cache, nil, e, "github", "sgu/mydefrepo", &sdk.VCSBranch{ID: "refs/heads/main"}, nil)
	require.NoError(t, err)

	repoWebHooks, err := workflow_v2.LoadHooksByRepositoryEvent(context.TODO(), db, vcsServer.Name, "sgu/myapp", "push")
	require.NoError(t, err)
	require.Equal(t, 1, len(repoWebHooks))

	// Distant workflow so worklow update hook must be saved
	workflowUpdateHooks, err := workflow_v2.LoadHooksByWorkflowUpdated(context.TODO(), db, proj.Key, vcsServer.Name, repoDef.Name, e.Name, "123456")
	require.NoError(t, err)
	require.NotNil(t, workflowUpdateHooks)

	// Model and workflow on different repo,  hook must not be saved
	modelKey := fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsServer.Name, repoDef.Name, "MyModel")
	hooks, err := workflow_v2.LoadHooksByModelUpdated(context.TODO(), db, "123456", []string{modelKey})
	require.NoError(t, err)
	require.Equal(t, 0, len(hooks))
}

func TestManageWorkflowHooksAllDistantEntitiesNonDefaultBranch(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE from v2_workflow_hook")
	require.NoError(t, err)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repoDef := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "sgu/myDefRepo")

	//
	e := sdk.EntityWithObject{
		Entity: sdk.Entity{
			ProjectKey:          proj.Key,
			ProjectRepositoryID: repoDef.ID,
			Ref:                 "refs/heads/test",
			Type:                sdk.EntityTypeWorkflow,
			Commit:              "123456",
			Name:                sdk.RandomString(10),
		},
		Workflow: sdk.V2Workflow{
			Repository: &sdk.WorkflowRepository{
				VCSServer: vcsServer.Name,
				Name:      "sgu/myapp",
			},
			On: &sdk.WorkflowOn{
				Push:           &sdk.WorkflowOnPush{},
				WorkflowUpdate: &sdk.WorkflowOnWorkflowUpdate{},
				ModelUpdate: &sdk.WorkflowOnModelUpdate{
					Models: []string{"MyModel"},
				},
				Schedule: []sdk.WorkflowOnSchedule{
					{
						Cron: "* * * * *",
					},
				},
			},
		},
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e.Entity))
	_, err = manageWorkflowHooks(context.TODO(), db, api.Cache, nil, e, "github", "sgu/mydefrepo", &sdk.VCSBranch{ID: "refs/heads/main"}, nil)
	require.NoError(t, err)

	repoWebHooks, err := workflow_v2.LoadHooksByRepositoryEvent(context.TODO(), db, vcsServer.Name, "sgu/myapp", "push")
	require.NoError(t, err)
	require.Equal(t, 0, len(repoWebHooks))

	// Non default branch, hook must not be saved
	_, err = workflow_v2.LoadHooksByWorkflowUpdated(context.TODO(), db, proj.Key, vcsServer.Name, repoDef.Name, e.Name, "123456")
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))

	// Non default branch, hook must not be saved
	modelKey := fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsServer.Name, repoDef.Name, "MyModel")
	hooks, err := workflow_v2.LoadHooksByModelUpdated(context.TODO(), db, "123456", []string{modelKey})
	require.NoError(t, err)
	require.Equal(t, 0, len(hooks))
}

func TestAnalyzeGithubDeleteEntity(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	uk, err := user.LoadGPGKeyByKeyID(ctx, db, "F344BDDCE15F17D7")
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		require.NoError(t, err)
	}
	if uk != nil {
		require.NoError(t, user.DeleteGPGKey(db, *uk))
	}

	u, _ := assets.InsertLambdaUser(t, db)
	userKey := &sdk.UserGPGKey{
		KeyID: "F344BDDCE15F17D7",
		PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQINBFXv+IMBEADYp5xTZ0YKvUgXvvE0SSeXg+bo8mPTTq5clIYWfdmfVjS6NL8T
IYhnjj5MXXIoGs/Lyx+B0VUC9Jo5ObSVCViJRXGVwfHpMIW2+n4i251pGO4bUPPw
o7SpEbvEc1tqE4P3OU26BZhZoIv3AaslMXi+v2eZjJe5Qr4BSc6FLOo5pdAm9HAZ
7vkj7M/WKbbpoXKpfZF+DLmJsrWU/2/TVD2ZdLANAwiXSVLmLeJr0z/zVX+9o6b9
Rz7HV3euPDCWb/t2fEI4yT8+e92QlxCtVcMpG7ZpxftQbl4z0U8kHASr38UqjTL5
VtCHKUFD5KyrxHUxFEUingI+M8NstzObho65oK2yxzcoufHTQBo2sfL4xWqPmFj8
hZeNSz3P6XPLQ+wdIganRGweEv+LSpbSMXIaWpiE2GjwFVRRTaffCgWvth1JRBti
deJI5rxe7UztytDTg8Ekt5MAqTBIoxqZ24zOdbxEef4EpEiYnaa5GXMg8EHH1bJr
aIc2nuY7Zfoz7uvqS8F5ohh69q/LbSv+gxw7aU36oogd13+8/MYPE29vfb+tIIwz
xen0PUcPkt83EQ0RdTbG7AnrvNMXDINp+ZGz3Oks3OXehezX/syPAe7BunPU/Zfy
wK/GDhpjsS9R+y/ZWDXX/LyQfHiHw5nIoX0m6I43BdshrQH5fyrTvJA02wARAQAB
tCxTdGV2ZW4gR3VpaGV1eCA8c3RldmVuLmd1aWhldXhAY29ycC5vdmguY29tPokC
OAQTAQIAIgUCVe/4gwIbAwYLCQgHAwIGFQgCCQoLBBYCAwECHgECF4AACgkQ80S9
3OFfF9dDYw//VuE85jnUS6bFwdvkFtdbXPZxOsFDMX9tiCjYDdXfT+98AoGgZboC
Ya/E8T5NhFjG8yGC8WOsiZZhQ/DyFr7TT+CwLvZ2JmLarEKHpL//YNr5ACp7Q8lo
7PSAACEJx2J3s2qpEbpMrvXVOJkAbwiFUnSz8R14RMJZLCmgbA5CDKpYqCSM/1B1
ED/WY8phhV6GknsqvG/cQiyQNQBg8PEdsyiNn79QWRGD8q5ZvWsxAuMMY7j/WSLy
VHZJ9wR9lBM9Lf3NJ+vDoVq56WaAH30vuVJ2LzGwHOULDKSFkQZ1JPodsu+7tDAZ
QDENAMaD1940GzmBANH/FOHD5T2VrOYMtPHMcyXJRSUOgw3MtvSuKJJliLMO0DNa
EZG14nCcdDP7xoS9da2JddMxDmqhzuCpsPk0IVH+JSjrAKOJ7r5YE3/vWcI2dQaU
nOYBhqST73RN2g6wF5xLt9Oi1DXYFBfdhz+oXJ1ck34MB3oPx5yzlY9Rp7N5F9a+
gDiuE1Y1iqRX0uuoDq8b2EsZrQ4dSvpjZwWYRsDghjSATjiAcrhC70NjpG22Avwt
0x3SPG+HQYgzYs9idQMI6lpKqoFU9QUHMsWQKuBFE0ZXJs9Q9d+zjjUCebFZ7LjN
twZyhn8QXg5FUhLygfF6Pq8jnYMXMzAbKXm3NEC8X1/VGaZjB1Lszcq5Ag0EVe/4
gwEQAMGVA4T9qs/a8zy10Tc8nSGAMdNzI26D0fhH2rRtjeNJs5BqGNMPu2Eg5DKR
7rStsw58fDvdKeB116ZPXq4Hoe66H+Pw83QIwDQk/vN965fPwqz9BIgDE/xTx09w
wVLvfKAHIFQF7znqqUYrES2gYpvirVD7knGKjVMMkB4Hil7TMcya6MTD2a9L32be
nMfZ5sA4311TJPS+kIEeEuG+SU2w3i6YRho+atUvsxkMNzmx92ow6JDznX8Kpbr/
PVExZObUW0+379yMKlgaZLhrgqbcwm+IOCgsM5XSs/zGb2AFACADnOdqOYToRtIt
bdvH2Y/2fq3t3upuzbpM3fiUu0Vs2rVRe5w4luHt6ZpKdZo43blEL9MN/ZbQVYE0
N/5/9SAizfyyOGmrNvB4EwPLpyImBre9MRcZJRvg22tFxcbnM2+SJGwfmD0FnPGe
gIRihPgsQxrx6BOCB1JzCUCOUqZ12gy2ul2RuopGEEX8YKLWNryNN8v0ooS+PU8D
Ii2biB9O9UYecXPVhxVP64gl48lN8psIFL+YSJ+svAErsQYGASApRF240Nor98+L
zgHm1+60JNU1i5gYQV6RzDMUML43XYWxsVqA21mTZZSJFwC/TcmLDl9yGyIOTNG4
kFPT/c1xibi5MGBQE8gIxdwEwfrj9iqohMt8afJfIMhcfwdzABEBAAGJAh8EGAEC
AAkFAlXv+IMCGwwACgkQ80S93OFfF9ceWxAAprlvofJ8qkREkhNznF9YacuDru8n
8BfWINLHKMI8zmOaijcdZVjC/+5FxC7rIx/Bc+vJCmMTTAkud0RfF4zDBPAqEv0q
I+4lR/ATThkRmX3XJSBDeI62MJTOPHqZ13mPnof5fAdy9HFclc1vwMoBjOofJpq4
DiQqchzR8eg0YXFDfaKptDrjvBGeffb14RjI7MeNwp5YIrEc4zZfQGZ3p3Q8oH84
vMbWjiWp/OZH+ZBVixLWQVMrTu1jSE7Hj7FgbBJzaXGoH/NyYqTTWany06Mpltu7
+71v/gJGgav+VxGcPoEzI83SCKdWdlLdtK5HjzpmqMixX1NaO5gfQblatmi7qLIT
f42j7Ul9tumMOLPtKQmiuloMJHO7mUmqOZDxmbrNmb47rAmIU3KRx5oNID9rLhxe
4tuAIsY8Lu2mU+PR5XQlgjG1J0aCunxUOZ4HhLUqJ6U+QWLUpRAq74zjPGocIv1e
GAH2qkfaNTarBQKytsA7k6vnzHmY7KYup3c9qQjMC8XzjuKBF5oJXl3yBU2VCPaw
qVWF89Lpz5nHVxmY2ejU/DvV7zUUAiqlVyzFmiOed5O66jVtPG4YM5x2EMwNvejk
e9rMe4DS8qoQg4er1Z3WNcb4JOAc33HDOol1LFOH1buNN5V+KrkUo0fPWMf4nQ97
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
		AuthentifiedUserID: u.ID,
	}
	require.NoError(t, user.InsertGPGKey(ctx, db, userKey))

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManageWorkerModel, proj1.Key, *u)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManageWorkflow, proj1.Key, *u)

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	workflowEntity := sdk.Entity{
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflow,
		Commit:              "HEAD",
		Ref:                 "refs/heads/master",
		Name:                "workflow1",
	}
	require.NoError(t, entity.Insert(ctx, db, &workflowEntity))

	actionEntity := sdk.Entity{
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeAction,
		Commit:              "HEAD",
		Ref:                 "refs/heads/master",
		Name:                "action1",
	}
	require.NoError(t, entity.Insert(ctx, db, &actionEntity))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	model := `
    name: docker-debian
    description: my debian worker model
    osarch: linux/amd64
    type: docker
    spec:
      image: myimage:1.1
      envs:
        MYVAR: toto
  `
	encodedModel := base64.StdEncoding.EncodeToString([]byte(model))

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "-----BEGIN PGP SIGNATURE-----\n\niQIzBAABCAAdFiEEfYJxMHx+E0DPuqaA80S93OFfF9cFAmME7aIACgkQ80S93OFf\nF9eFWBAAq5hOcZIx/A+8J6/NwRtXMs5OW+TJxzJb5siXdRC8Mjrm+fqwpTPPHqtB\nbb7iuiRnmY/HqCegULiw4qVxDyA3sswyDHPLcyUcfG4drJGylPW9ZYg3YeRslX2B\niQykYZyd4h3R/euYAuBKA9vMGoWnaU/Vh22A11Po1pXpPq623FTkiFOSAZrD8Hql\nEvmlhw26qHSPlhsdSKsR+/FPvpLUXlNUiYB5oq7W9qy0yOOafgwZ9r3vvxshzvkt\nvW5zG+R05thQ8icCyrWfEfIWp+TTtQX3asOopnQG9dFs2LRODLXXaHTRVRB/MWPa\nNVvUD/dIzBVyNimpik+2Uqq5jWNiXavQmqoxyL9n4A372AIH7Hu78NnfmAz7VnYo\nyVHRNBryiCcYNj5g0x/WnGsDuhQr7170ODw7QfEYJdCPxGgYuhdYovHdjcMcgWpF\ncWEtayj8bhuLTjjxEsqXTv+psxwB55N5OUvyXmNAaFLhJSEI+l1VHW14L3gZFdPT\n+VgPQtT9a1+GEjPqLvZ6wLVTcSI9uogK6NHowmyM261FtFQqLVdkOdUU8RCR8qLC\nekZWQaJutqicIZTolAQyBPBw8aQz0i+uBUgdWkoiHf/zEEudu0b06IpDq2oYFFVH\nVmCuZ3/AcXrW6T3XXcE5pu+Rvsi57O7iR8i7TIP0CaDTr2FfQWc=\n=/H7t\n-----END PGP SIGNATURE-----",
					Verified:  true,
					Hash:      "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: true,
						Name:        "worker-models",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworker-models?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: false,
						IsFile:      true,
						Name:        "mymodels.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/content/.cds%2Fworker-models%2Fmymodels.yml?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {

				content := sdk.VCSContent{
					IsDirectory: false,
					IsFile:      true,
					Name:        "mymodels.yml",
					Content:     encodedModel,
				}
				*(out.(*sdk.VCSContent)) = content
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=&default=true&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := sdk.VCSBranch{
					ID: "refs/heads/master",
				}
				*(out.(*sdk.VCSBranch)) = contents
				return nil, 200, nil
			},
		)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, analysisUpdated.Status)

	es, err := entity.LoadByTypeAndRefCommit(context.TODO(), db, repo.ID, sdk.EntityTypeWorkerModel, "refs/heads/master", "abcdef")
	require.NoError(t, err)

	require.Equal(t, 1, len(es))
	require.Equal(t, model, es[0].Data)

	e, err := entity.LoadByRefTypeNameCommit(context.TODO(), db, repo.ID, "refs/heads/master", sdk.EntityTypeWorkerModel, "docker-debian", "abcdef")
	require.NoError(t, err)
	require.Equal(t, model, e.Data)

	// Check workflow deletion
	_, err = entity.LoadByRefTypeNameCommit(ctx, db, repo.ID, "refs/heads/master", sdk.EntityTypeWorkflow, "workflow1", "HEAD")
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))

	// Check action
	_, err = entity.LoadByRefTypeNameCommit(ctx, db, repo.ID, "refs/heads/master", sdk.EntityTypeAction, "action1", "HEAD")
	require.NoError(t, err)
}

func TestAnalyzeGithubUpdateWorkerModelNoRight(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	uk, err := user.LoadGPGKeyByKeyID(ctx, db, "F344BDDCE15F17D7")
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		require.NoError(t, err)
	}
	if uk != nil {
		require.NoError(t, user.DeleteGPGKey(db, *uk))
	}

	u, _ := assets.InsertLambdaUser(t, db)
	userKey := &sdk.UserGPGKey{
		KeyID: "F344BDDCE15F17D7",
		PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
		AuthentifiedUserID: u.ID,
	}
	require.NoError(t, user.InsertGPGKey(ctx, db, userKey))

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	existingEntity := sdk.Entity{
		ID:                  sdk.UUID(),
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/mymodels.yml",
		Name:                "docker-debian",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		LastUpdate:          time.Now(),
		Data:                "blabla",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &existingEntity))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	model := `
    name: docker-debian
    description: my debian worker model
    osarch: linux/amd64
    type: docker
    spec:
      image: myimage:1.1
      envs:
        MYVAR: toto
  `
	encodedModel := base64.StdEncoding.EncodeToString([]byte(model))

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "-----BEGIN PGP SIGNATURE-----\n\niQIzBAABCAAdFiEEfYJxMHx+E0DPuqaA80S93OFfF9cFAmME7aIACgkQ80S93OFf\nF9eFWBAAq5hOcZIx/A+8J6/NwRtXMs5OW+TJxzJb5siXdRC8Mjrm+fqwpTPPHqtB\nbb7iuiRnmY/HqCegULiw4qVxDyA3sswyDHPLcyUcfG4drJGylPW9ZYg3YeRslX2B\niQykYZyd4h3R/euYAuBKA9vMGoWnaU/Vh22A11Po1pXpPq623FTkiFOSAZrD8Hql\nEvmlhw26qHSPlhsdSKsR+/FPvpLUXlNUiYB5oq7W9qy0yOOafgwZ9r3vvxshzvkt\nvW5zG+R05thQ8icCyrWfEfIWp+TTtQX3asOopnQG9dFs2LRODLXXaHTRVRB/MWPa\nNVvUD/dIzBVyNimpik+2Uqq5jWNiXavQmqoxyL9n4A372AIH7Hu78NnfmAz7VnYo\nyVHRNBryiCcYNj5g0x/WnGsDuhQr7170ODw7QfEYJdCPxGgYuhdYovHdjcMcgWpF\ncWEtayj8bhuLTjjxEsqXTv+psxwB55N5OUvyXmNAaFLhJSEI+l1VHW14L3gZFdPT\n+VgPQtT9a1+GEjPqLvZ6wLVTcSI9uogK6NHowmyM261FtFQqLVdkOdUU8RCR8qLC\nekZWQaJutqicIZTolAQyBPBw8aQz0i+uBUgdWkoiHf/zEEudu0b06IpDq2oYFFVH\nVmCuZ3/AcXrW6T3XXcE5pu+Rvsi57O7iR8i7TIP0CaDTr2FfQWc=\n=/H7t\n-----END PGP SIGNATURE-----",
					Verified:  true,
					Hash:      "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: true,
						Name:        "worker-models",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworker-models?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: false,
						IsFile:      true,
						Name:        "mymodels.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/content/.cds%2Fworker-models%2Fmymodels.yml?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {

				content := sdk.VCSContent{
					IsDirectory: false,
					IsFile:      true,
					Name:        "mymodels.yml",
					Content:     encodedModel,
				}
				*(out.(*sdk.VCSContent)) = content
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=&default=true&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := sdk.VCSBranch{
					ID: "refs/heads/master",
				}
				*(out.(*sdk.VCSBranch)) = contents
				return nil, 200, nil
			},
		)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSkipped, analysisUpdated.Status)

	es, err := entity.LoadByTypeAndRefCommit(context.TODO(), db, repo.ID, sdk.EntityTypeWorkerModel, "refs/heads/master", "abcdef")
	require.NoError(t, err)

	require.Equal(t, 1, len(es))
	require.Equal(t, "blabla", es[0].Data)

	e, err := entity.LoadByRefTypeNameCommit(context.TODO(), db, repo.ID, "refs/heads/master", sdk.EntityTypeWorkerModel, "docker-debian", "abcdef")
	require.NoError(t, err)
	require.Equal(t, "blabla", e.Data)
}

func TestAnalyzeGithubAddWorkflowWithWrongTemplate(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	uk, err := user.LoadGPGKeyByKeyID(ctx, db, "F344BDDCE15F17D7")
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		require.NoError(t, err)
	}
	if uk != nil {
		require.NoError(t, user.DeleteGPGKey(db, *uk))
	}

	u, _ := assets.InsertLambdaUser(t, db)
	userKey := &sdk.UserGPGKey{
		KeyID: "F344BDDCE15F17D7",
		PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
		AuthentifiedUserID: u.ID,
	}
	require.NoError(t, user.InsertGPGKey(ctx, db, userKey))

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManageWorkflow, proj1.Key, *u)

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	workflow := `name: myworkflow
from: mytemplate
parameters:
  toto: tata`
	encodedWorkflow := base64.StdEncoding.EncodeToString([]byte(workflow))

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "-----BEGIN PGP SIGNATURE-----\n\niQIzBAABCAAdFiEEfYJxMHx+E0DPuqaA80S93OFfF9cFAmME7aIACgkQ80S93OFf\nF9eFWBAAq5hOcZIx/A+8J6/NwRtXMs5OW+TJxzJb5siXdRC8Mjrm+fqwpTPPHqtB\nbb7iuiRnmY/HqCegULiw4qVxDyA3sswyDHPLcyUcfG4drJGylPW9ZYg3YeRslX2B\niQykYZyd4h3R/euYAuBKA9vMGoWnaU/Vh22A11Po1pXpPq623FTkiFOSAZrD8Hql\nEvmlhw26qHSPlhsdSKsR+/FPvpLUXlNUiYB5oq7W9qy0yOOafgwZ9r3vvxshzvkt\nvW5zG+R05thQ8icCyrWfEfIWp+TTtQX3asOopnQG9dFs2LRODLXXaHTRVRB/MWPa\nNVvUD/dIzBVyNimpik+2Uqq5jWNiXavQmqoxyL9n4A372AIH7Hu78NnfmAz7VnYo\nyVHRNBryiCcYNj5g0x/WnGsDuhQr7170ODw7QfEYJdCPxGgYuhdYovHdjcMcgWpF\ncWEtayj8bhuLTjjxEsqXTv+psxwB55N5OUvyXmNAaFLhJSEI+l1VHW14L3gZFdPT\n+VgPQtT9a1+GEjPqLvZ6wLVTcSI9uogK6NHowmyM261FtFQqLVdkOdUU8RCR8qLC\nekZWQaJutqicIZTolAQyBPBw8aQz0i+uBUgdWkoiHf/zEEudu0b06IpDq2oYFFVH\nVmCuZ3/AcXrW6T3XXcE5pu+Rvsi57O7iR8i7TIP0CaDTr2FfQWc=\n=/H7t\n-----END PGP SIGNATURE-----",
					Verified:  true,
					Hash:      "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: true,
						Name:        "workflows",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworkflows?commit=abcdef&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: false,
						IsFile:      true,
						Name:        "myworkflow.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/content/.cds%2Fworkflows%2Fmyworkflow.yml?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {

				content := sdk.VCSContent{
					IsDirectory: false,
					IsFile:      true,
					Name:        "2Fmyworkflow.yml",
					Content:     encodedWorkflow,
				}
				*(out.(*sdk.VCSContent)) = content
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=&default=true&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := sdk.VCSBranch{
					ID: "refs/heads/master",
				}
				*(out.(*sdk.VCSBranch)) = contents
				return nil, 200, nil
			},
		).AnyTimes()

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))
	anal, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusError, anal.Status)
	require.Equal(t, "workflow myworkflow: unable to find workflow dependency: mytemplate", anal.Data.Error)
}

func TestAnalyzeGithubUpdateWorkflowNoRight(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	uk, err := user.LoadGPGKeyByKeyID(ctx, db, "F344BDDCE15F17D7")
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		require.NoError(t, err)
	}
	if uk != nil {
		require.NoError(t, user.DeleteGPGKey(db, *uk))
	}

	u, _ := assets.InsertLambdaUser(t, db)
	userKey := &sdk.UserGPGKey{
		KeyID: "F344BDDCE15F17D7",
		PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
		AuthentifiedUserID: u.ID,
	}
	require.NoError(t, user.InsertGPGKey(ctx, db, userKey))

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	existingEntity := sdk.Entity{
		ID:                  sdk.UUID(),
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflow,
		FilePath:            ".cds/workflows/myworkflow.yml",
		Name:                "myworkflow",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		LastUpdate:          time.Now(),
		Data:                "blabla",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &existingEntity))

	existingHeadEntity := existingEntity
	existingHeadEntity.ID = ""
	existingHeadEntity.Commit = "HEAD"
	require.NoError(t, entity.Insert(context.TODO(), db, &existingHeadEntity))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "zyxwv",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/devBranch",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	wkf := `name: myworkflow
jobs:
  root:
    runs-on: .cds/worker-models/mymodel.yml 
    steps:
    - run: echo toto`
	encodedWorkflow := base64.StdEncoding.EncodeToString([]byte(wkf))

	wm := `name: mymodel
osarch: linux/amd64
type: docker
spec:
  image: myimage`

	encodedWorkerModel := base64.StdEncoding.EncodeToString([]byte(wm))

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=devBranch&default=false&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				branch := &sdk.VCSBranch{
					ID:           sdk.GitRefBranchPrefix + "devBranch",
					DisplayID:    "devBranch",
					Default:      false,
					LatestCommit: "zyxwv",
				}
				*(out.(*sdk.VCSBranch)) = *branch
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/commits/zyxwv", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Signature: "-----BEGIN PGP SIGNATURE-----\n\niQIzBAABCAAdFiEEfYJxMHx+E0DPuqaA80S93OFfF9cFAmME7aIACgkQ80S93OFf\nF9eFWBAAq5hOcZIx/A+8J6/NwRtXMs5OW+TJxzJb5siXdRC8Mjrm+fqwpTPPHqtB\nbb7iuiRnmY/HqCegULiw4qVxDyA3sswyDHPLcyUcfG4drJGylPW9ZYg3YeRslX2B\niQykYZyd4h3R/euYAuBKA9vMGoWnaU/Vh22A11Po1pXpPq623FTkiFOSAZrD8Hql\nEvmlhw26qHSPlhsdSKsR+/FPvpLUXlNUiYB5oq7W9qy0yOOafgwZ9r3vvxshzvkt\nvW5zG+R05thQ8icCyrWfEfIWp+TTtQX3asOopnQG9dFs2LRODLXXaHTRVRB/MWPa\nNVvUD/dIzBVyNimpik+2Uqq5jWNiXavQmqoxyL9n4A372AIH7Hu78NnfmAz7VnYo\nyVHRNBryiCcYNj5g0x/WnGsDuhQr7170ODw7QfEYJdCPxGgYuhdYovHdjcMcgWpF\ncWEtayj8bhuLTjjxEsqXTv+psxwB55N5OUvyXmNAaFLhJSEI+l1VHW14L3gZFdPT\n+VgPQtT9a1+GEjPqLvZ6wLVTcSI9uogK6NHowmyM261FtFQqLVdkOdUU8RCR8qLC\nekZWQaJutqicIZTolAQyBPBw8aQz0i+uBUgdWkoiHf/zEEudu0b06IpDq2oYFFVH\nVmCuZ3/AcXrW6T3XXcE5pu+Rvsi57O7iR8i7TIP0CaDTr2FfQWc=\n=/H7t\n-----END PGP SIGNATURE-----",
					Verified:  true,
					Hash:      "abcdef",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=zyxwv&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: true,
						Name:        "workflows",
					},
					{
						IsDirectory: true,
						Name:        "worker-models",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworkflows?commit=zyxwv&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: false,
						IsFile:      true,
						Name:        "myworkflow.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworker-models?commit=zyxwv&offset=0&limit=100", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: false,
						IsFile:      true,
						Name:        "mymodel.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/content/.cds%2Fworkflows%2Fmyworkflow.yml?commit=zyxwv", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {

				content := sdk.VCSContent{
					IsDirectory: false,
					IsFile:      true,
					Name:        "myworkflow.yml",
					Content:     encodedWorkflow,
				}
				*(out.(*sdk.VCSContent)) = content
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/content/.cds%2Fworker-models%2Fmymodel.yml?commit=zyxwv", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {

				content := sdk.VCSContent{
					IsDirectory: false,
					IsFile:      true,
					Name:        "mymodel.yml",
					Content:     encodedWorkerModel,
				}
				*(out.(*sdk.VCSContent)) = content
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/branches/?branch=&default=true&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := sdk.VCSBranch{
					ID:           "refs/heads/master",
					LatestCommit: "abcdef",
				}
				*(out.(*sdk.VCSBranch)) = contents
				return nil, 200, nil
			},
		)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	t.Logf(">>>>%+v", analysisUpdated.Data.Error)
	require.Equal(t, sdk.RepositoryAnalysisStatusSkipped, analysisUpdated.Status)

	entitiesNonHEad, err := entity.LoadByTypeAndRefCommit(context.TODO(), db, repo.ID, sdk.EntityTypeWorkflow, "refs/heads/devBranch", "zyxwv")
	require.NoError(t, err)
	entitiesHEad, err := entity.LoadByTypeAndRefCommit(context.TODO(), db, repo.ID, sdk.EntityTypeWorkflow, "refs/heads/devBranch", "HEAD")
	require.NoError(t, err)

	require.Equal(t, 1, len(entitiesNonHEad))
	require.Equal(t, 1, len(entitiesHEad))
	require.Equal(t, entitiesNonHEad[0].Data, entitiesHEad[0].Data)
}

func TestAnalyzeWrongWorkflow(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	_, _ = db.Exec("DELETE FROM service")

	// Create project
	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, api.Cache, key1, key1)

	api.Config.VCS.GPGKeys = map[string][]GPGKey{
		"vcs-server": {
			{
				ID: "F344BDDCE15F17D7",
				PublicKey: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
GDFkaTe3nUJdYV4=
=SNcy
-----END PGP PUBLIC KEY BLOCK-----`,
			},
		},
	}

	u, _ := assets.InsertLambdaUser(t, db)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManageWorkerModel, proj1.Key, *u)

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", sdk.VCSTypeBitbucketServer)

	repo := sdk.ProjectRepository{
		Name:         "my/repo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		ProjectKey:   proj1.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	analysis := sdk.ProjectRepositoryAnalysis{
		ID:                  "",
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		Commit:              "abcdef",
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Created:             time.Now(),
		LastModified:        time.Now(),
		Ref:                 "refs/heads/master",
		VCSProjectID:        vcsProject.ID,
		Data: sdk.ProjectRepositoryData{
			OperationUUID: sdk.UUID(),
		},
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).AnyTimes()

	entityWM := sdk.Entity{
		ProjectKey:          proj1.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/mymodel.yml",
		Name:                "mymodel",
		Commit:              "HEAD",
		Ref:                 "refs/heads/main",
		UserID:              &u.ID,
		Data:                "name: mymodel",
	}
	require.NoError(t, entity.Insert(ctx, db, &entityWM))

	modelPath := fmt.Sprintf("%s/%s/%s/%s@refs/heads/main", proj1.Key, vcsProject.Name, repo.Name, entityWM.Name)

	wkf := fmt.Sprintf(`name: myworkflow
jobs:
  root:
    concurrency: toto
    runs-on: %s
    steps:
    - run: echo toto`, modelPath)

	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Name: ".cds/workflows/myworkflow.yml",
		Mode: 0755,
		Size: int64(len([]byte(wkf))),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte(wkf))
	require.NoError(t, err)
	tw.Close()
	gw.Close()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/my/repo/contents/.cds?commit=abcdef&offset=0&limit=1", gomock.Any(), gomock.Any(), gomock.Any())

	servicesClients.EXPECT().
		StreamRequest(gomock.Any(), "POST", "/vcs/vcs-server/repos/my/repo/archive", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, _ interface{}) (io.Reader, http.Header, int, error) {
				return bytes.NewReader(buf.Bytes()), nil, 200, nil
			},
		)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/my/repo/commits/abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				commit := &sdk.VCSCommit{
					Hash: "abcdef",
					Committer: sdk.VCSAuthor{
						Name:        u.Username,
						Slug:        u.Username,
						DisplayName: u.Username,
						Email:       u.GetEmail(),
					},
					Verified: true,
					KeyID:    "F344BDDCE15F17D7",
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/my/repo/contents/.cds?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: true,
						Name:        "workflows",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/my/repo/contents/.cds%2Fworkflows?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				contents := []sdk.VCSContent{
					{
						IsDirectory: false,
						IsFile:      true,
						Name:        "myworkflow.yml",
					},
				}
				*(out.(*[]sdk.VCSContent)) = contents
				return nil, 200, nil
			},
		).MaxTimes(1)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)

	require.Equal(t, sdk.RepositoryAnalysisStatusError, analysisUpdated.Status)
	require.Contains(t, analysisUpdated.Data.Error, "workflow myworkflow job root: concurrency toto doesn't exist")
}
