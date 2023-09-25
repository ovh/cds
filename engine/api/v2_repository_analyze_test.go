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

	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

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

func TestCleanAnalysis(t *testing.T) {
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
		Branch:              "master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

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
		Branch:              "master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

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
		Branch:              "master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

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

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSkipped, analysisUpdated.Status)
	require.Equal(t, "gpgkey F344BDDCE15F17D7 not found", analysisUpdated.Data.Error)
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
		Branch:              "master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

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
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
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
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds%2Fworker-models?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
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
		Branch:              "master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

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
		Branch:              "master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

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
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
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

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, analysisUpdated.Status)

	es, err := entity.LoadByRepositoryAndType(context.TODO(), db, repo.ID, sdk.EntityTypeWorkerModel)
	require.NoError(t, err)

	require.Equal(t, 1, len(es))
	require.Equal(t, model, es[0].Data)
	t.Logf("%+v", es[0])

	e, err := entity.LoadByBranchTypeName(context.TODO(), db, repo.ID, "master", sdk.EntityTypeWorkerModel, "docker-debian")
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
		Branch:              "master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

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

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/search/pullrequest?commit=abcdef&state=closed", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			pr := &sdk.VCSPullRequest{
				MergeBy: sdk.VCSAuthor{
					Slug: githubUsername,
				},
			}
			*(out.(*sdk.VCSPullRequest)) = *pr
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
		Branch:              "master",
		VCSProjectID:        vcsProject.ID,
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

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
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/contents/.cds?commit=abcdef", gomock.Any(), gomock.Any(), gomock.Any()).
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
	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-server/repos/myrepo/search/pullrequest?commit=abcdef&state=closed", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			pr := &sdk.VCSPullRequest{
				MergeBy: sdk.VCSAuthor{
					Slug: githubUsername,
					ID:   ul.ExternalID,
				},
			}
			*(out.(*sdk.VCSPullRequest)) = *pr
			return nil, 200, nil
		},
		).MaxTimes(1)

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, analysisUpdated.Status)

	es, err := entity.LoadByRepositoryAndType(context.TODO(), db, repo.ID, sdk.EntityTypeWorkerModel)
	require.NoError(t, err)

	require.Equal(t, 1, len(es))
	require.Equal(t, model, es[0].Data)
	t.Logf("%+v", es[0])

	e, err := entity.LoadByBranchTypeName(context.TODO(), db, repo.ID, "master", sdk.EntityTypeWorkerModel, "docker-debian")
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
		Branch:              "master",
		VCSProjectID:        vcsProject.ID,
		Data: sdk.ProjectRepositoryData{
			OperationUUID: sdk.UUID(),
		},
	}
	require.NoError(t, repository.InsertAnalysis(ctx, db, &analysis))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sRepo, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeRepositories)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sRepo)
		services.NewClient = services.NewDefaultClient
	}()

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
				}
				*(out.(*sdk.VCSCommit)) = *commit
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/operations/"+analysis.Data.OperationUUID, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ ...interface{}) (http.Header, int, error) {
				op := &sdk.Operation{
					Status: sdk.OperationStatusDone,
					Setup: sdk.OperationSetup{
						Checkout: sdk.OperationCheckout{},
					},
				}
				op.Setup.Checkout.Result.SignKeyID = "F344BDDCE15F17D7"
				op.Setup.Checkout.Result.CommitVerified = true
				*(out.(*sdk.Operation)) = *op
				return nil, 200, nil
			})

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

	require.NoError(t, api.analyzeRepository(ctx, repo.ID, analysis.ID))

	analysisUpdated, err := repository.LoadRepositoryAnalysisById(ctx, db, repo.ID, analysis.ID)
	require.NoError(t, err)
	t.Logf("%+v", analysisUpdated)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, analysisUpdated.Status)

	es, err := entity.LoadByRepositoryAndType(context.TODO(), db, repo.ID, sdk.EntityTypeWorkerModel)
	require.NoError(t, err)

	require.Equal(t, 1, len(es))
	require.Equal(t, model, es[0].Data)
	t.Logf("%+v", es[0])

	e, err := entity.LoadByBranchTypeName(context.TODO(), db, repo.ID, "master", sdk.EntityTypeWorkerModel, "docker-debian")
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
			Branch:              "master",
			Type:                sdk.EntityTypeWorkflow,
			Commit:              "123456",
			Name:                sdk.RandomString(10),
		},
		Workflow: sdk.V2Workflow{
			Repository: &sdk.WorkflowRepository{
				VCSServer: vcsServer.Name,
				Name:      repoDef.Name,
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
	require.NoError(t, manageWorkflowHooks(context.TODO(), db, e, "github", "sgu/myDefRepo", "main"))

	repoWebHooks, err := workflow_v2.LoadHooksByRepositoryEvent(context.TODO(), db, vcsServer.Name, repoDef.Name, "push")
	require.NoError(t, err)
	require.Equal(t, 1, len(repoWebHooks))

	// Local workflow so worklow update hook must not be saved
	_, err = workflow_v2.LoadHooksByWorkflowUpdated(context.TODO(), db, proj.Key, vcsServer.Name, repoDef.Name, e.Name)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))

	// Local workflow so model update hook must not be saved
	hooks, err := workflow_v2.LoadHooksByModelUpdated(context.TODO(), db, []string{"MyModel"})
	require.NoError(t, err)
	require.Equal(t, 0, len(hooks))
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
			Branch:              "main",
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
	require.NoError(t, manageWorkflowHooks(context.TODO(), db, e, "github", "sgu/myDefRepo", "main"))

	repoWebHooks, err := workflow_v2.LoadHooksByRepositoryEvent(context.TODO(), db, vcsServer.Name, "sgu/myapp", "push")
	require.NoError(t, err)
	require.Equal(t, 1, len(repoWebHooks))

	// Local workflow so worklow update hook must not be saved
	workflowUpdateHooks, err := workflow_v2.LoadHooksByWorkflowUpdated(context.TODO(), db, proj.Key, vcsServer.Name, repoDef.Name, e.Name)
	require.NoError(t, err)
	require.NotNil(t, workflowUpdateHooks)

	// Local workflow so model update hook must not be saved
	modelKey := fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsServer.Name, repoDef.Name, "MyModel")
	hooks, err := workflow_v2.LoadHooksByModelUpdated(context.TODO(), db, []string{modelKey})
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
			Branch:              "main",
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
	require.NoError(t, manageWorkflowHooks(context.TODO(), db, e, "github", "sgu/myDefRepo", "main"))

	repoWebHooks, err := workflow_v2.LoadHooksByRepositoryEvent(context.TODO(), db, vcsServer.Name, "sgu/myapp", "push")
	require.NoError(t, err)
	require.Equal(t, 1, len(repoWebHooks))

	// Local workflow so worklow update hook must not be saved
	workflowUpdateHooks, err := workflow_v2.LoadHooksByWorkflowUpdated(context.TODO(), db, proj.Key, vcsServer.Name, repoDef.Name, e.Name)
	require.NoError(t, err)
	require.NotNil(t, workflowUpdateHooks)

	// Local workflow so model update hook must not be saved
	modelKey := fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsServer.Name, repoDef.Name, "MyModel")
	hooks, err := workflow_v2.LoadHooksByModelUpdated(context.TODO(), db, []string{modelKey})
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
			Branch:              "test",
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
	require.NoError(t, manageWorkflowHooks(context.TODO(), db, e, "github", "sgu/myDefRepo", "main"))

	repoWebHooks, err := workflow_v2.LoadHooksByRepositoryEvent(context.TODO(), db, vcsServer.Name, "sgu/myapp", "push")
	require.NoError(t, err)
	require.Equal(t, 0, len(repoWebHooks))

	// Local workflow so worklow update hook must not be saved
	_, err = workflow_v2.LoadHooksByWorkflowUpdated(context.TODO(), db, proj.Key, vcsServer.Name, repoDef.Name, e.Name)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))

	// Local workflow so model update hook must not be saved
	modelKey := fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsServer.Name, repoDef.Name, "MyModel")
	hooks, err := workflow_v2.LoadHooksByModelUpdated(context.TODO(), db, []string{modelKey})
	require.NoError(t, err)
	require.Equal(t, 0, len(hooks))
}
