package api

import (
	"context"
	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
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
		Name: "myrepo",
		Auth: sdk.ProjectRepositoryAuth{
			Username: "myuser",
			Token:    "mytoken",
		},
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
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
	require.Equal(t, "commit abcdef not found", analysisUpdated.Data.Error)
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
		Name: "myrepo",
		Auth: sdk.ProjectRepositoryAuth{
			Username: "myuser",
			Token:    "mytoken",
		},
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
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
	require.Contains(t, analysisUpdated.Data.Error, "unable to extract keyID from signature")
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
		Name: "myrepo",
		Auth: sdk.ProjectRepositoryAuth{
			Username: "myuser",
			Token:    "mytoken",
		},
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
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

	// Create VCS
	vcsProject := assets.InsertTestVCSProject(t, db, proj1.ID, "vcs-server", "github")

	repo := sdk.ProjectRepository{
		Name: "myrepo",
		Auth: sdk.ProjectRepositoryAuth{
			Username: "myuser",
			Token:    "mytoken",
		},
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
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
	require.Contains(t, analysisUpdated.Data.Error, "doesn't have enough right on project")
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
		Name: "myrepo",
		Auth: sdk.ProjectRepositoryAuth{
			Username: "myuser",
			Token:    "mytoken",
		},
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
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
