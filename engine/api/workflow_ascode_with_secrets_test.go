package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_RunNonDefaultBranchWithSecrets(t *testing.T) {
	api, db, router := newTestAPI(t)

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	_, _ = assets.InsertService(t, db, t.Name()+"_REPO", sdk.TypeRepositories)

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// The mock has been geenrated by mockgen: go get github.com/golang/mock/mockgen
	// If you have to regenerate thi mock you just have to run, from directory $GOPATH/src/github.com/ovh/cds/engine/api/services:
	// mockgen -source=http.go -destination=mock_services/services_mock.go Client
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	// Create a project with a repository manager
	prjKey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, prjKey, prjKey)
	u, pass := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	appVariable := sdk.Variable{
		Name:  "app-password",
		Value: "my application secret",
	}
	encryptedAppVariable, errA := project.EncryptWithBuiltinKey(api.mustDB(), proj.ID, appVariable.Name, appVariable.Value)
	require.NoError(t, errA)

	k, errK := keys.GenerateSSHKey("app-key")
	require.NoError(t, errK)
	appKey := sdk.ApplicationKey{
		Name:    "app-key",
		Private: k.Private,
	}
	encryptedAppKey, errB := project.EncryptWithBuiltinKey(api.mustDB(), proj.ID, appKey.Name, appKey.Private)
	require.NoError(t, errB)

	encryptedAppIntegration, errC := project.EncryptWithBuiltinKey(api.mustDB(), proj.ID, "token", "my application integration token")
	require.NoError(t, errC)

	envVariable := sdk.Variable{
		Name:  "env-password",
		Value: "my environment secret",
	}
	encryptedEnvVariable, errD := project.EncryptWithBuiltinKey(api.mustDB(), proj.ID, envVariable.Name, envVariable.Value)
	require.NoError(t, errD)

	ke, errKe := keys.GenerateSSHKey("env-key")
	require.NoError(t, errKe)
	envKey := sdk.EnvironmentKey{
		Name:    "env-key",
		Private: ke.Private,
	}
	encryptedEnvKey, errE := project.EncryptWithBuiltinKey(api.mustDB(), proj.ID, envKey.Name, envKey.Private)
	require.NoError(t, errE)

	UUID := sdk.UUID()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/myrepo/branches", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				bs := []sdk.VCSBranch{
					{
						DisplayID: "master",
						Default:   true,
					},
					{
						DisplayID: "devbranch",
					},
				}
				out = bs
				return nil, 200, nil
			},
		).MaxTimes(3)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/operations", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				ope := sdk.Operation{
					UUID: UUID,
				}
				*(out.(*sdk.Operation)) = ope
				return nil, 200, nil
			},
		).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/operations/"+UUID, gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				ope := new(sdk.Operation)
				ope.URL = "https://github.com/myrepo.git"
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusDone
				ope.VCSServer = "github"
				ope.RepoFullName = "myrepo"
				ope.RepositoryStrategy.Branch = "devbranch"
				ope.Setup.Checkout.Branch = "devbranch"
				ope.RepositoryInfo = new(sdk.OperationRepositoryInfo)
				ope.RepositoryInfo.Name = "myrepo"
				ope.RepositoryInfo.DefaultBranch = "master"
				ope.RepositoryInfo.FetchURL = "ssh://myrepo.git"
				ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
				ope.LoadFiles.Results = map[string][]byte{
					"myworkflowascode.yml": []byte(`name: myworkflowascode
version: v2.0
workflow:
  root:
    pipeline: root
    application: app-ascode
    environment: env-ascode
    integration: myintegration
`),
					"app-ascode.app.yml": []byte(`name: app-ascode
version: v1.0
repo: myrepo
vcs_server: github
variables:
  app-password:
    type: password
    value: ` + encryptedAppVariable + `
keys:
  app-key:
    type: ssh
    value: ` + encryptedAppKey + `
deployments:
  myintegration:
    token:
      type: password
      value: ` + encryptedAppIntegration),
					"env-ascode.env.yml": []byte(`name: env-ascode
version: v1.0
values:
  env-password:
    type: password
    value: ` + encryptedEnvVariable + `
keys:
  env-key:
    type: ssh
    value: ` + encryptedEnvKey),
					"go-repo.pip.yml": []byte(`name: root
version: v1.0`),
				}
				*(out.(*sdk.Operation)) = *ope
				return nil, 200, nil
			},
		)

	// RUN WORKFLOW MOCK
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/myrepo", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	// Get vcsInfos +  Get Commits
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/myrepo/branches/?branch=devbranch", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := sdk.VCSBranch{
					DisplayID:    "devbranch",
					LatestCommit: "abcdef123456",
				}
				*(out.(*sdk.VCSBranch)) = b
				return nil, 200, nil
			},
		).MinTimes(0)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/myrepo/commits/abcdef123456", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	assert.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

	modelIntegration := sdk.IntegrationModel{
		Name:       sdk.RandomString(10),
		Deployment: true,
	}
	require.NoError(t, integration.InsertModel(db, &modelIntegration))
	projInt := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"test": sdk.IntegrationConfigValue{
				Description: "here is a test",
				Type:        sdk.IntegrationConfigTypeString,
				Value:       "test",
			},
			"mypassword": sdk.IntegrationConfigValue{
				Description: "here isa password",
				Type:        sdk.IntegrationConfigTypePassword,
				Value:       "mypassword",
			},
		},
		Name:               "myintegration",
		ProjectID:          proj.ID,
		Model:              modelIntegration,
		IntegrationModelID: modelIntegration.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &projInt))
	t.Logf("### Integration %s created with id: %d\n", projInt.Name, projInt.ID)

	p := sdk.GRPCPlugin{
		Author:             "unitTest",
		Description:        "desc",
		Name:               sdk.RandomString(10),
		Type:               sdk.GRPCPluginDeploymentIntegration,
		IntegrationModelID: &modelIntegration.ID,
		Integration:        modelIntegration.Name,
		Binaries: []sdk.GRPCPluginBinary{
			{
				OS:   "linux",
				Arch: "adm64",
				Name: "blabla",
			},
		},
	}

	app := sdk.Application{
		ProjectID:          proj.ID,
		Name:               "app-ascode",
		VCSServer:          "github",
		RepositoryFullname: "myrepo",
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))
	require.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "env-ascode",
	}
	require.NoError(t, environment.InsertEnvironment(db, &env))
	require.NoError(t, plugin.Insert(db, &p))
	assert.NotEqual(t, 0, p.ID)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "root",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)
	pip.Stages = append(pip.Stages, *s)

	var err error
	proj, err = project.Load(context.TODO(), api.mustDB(), proj.Key,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
		project.LoadOptions.WithIntegrations,
	)
	require.NoError(t, err)

	w := sdk.Workflow{
		Name:       "myworkflowascode",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &w))

	// Make workflow as code
	require.NoError(t, workflow.UpdateFromRepository(db, w.ID, "ssh://myrepo.git"))

	// RUN IT
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Payload: map[string]string{
				"git.branch": "devbranch",
			},
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)
	var wr sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wr))

	require.NoError(t, waitCraftinWorkflow(t, api, db, wr.ID))

	wrDB, errDB := workflow.LoadRunByID(db, wr.ID, workflow.LoadRunOptions{})
	require.NoError(t, errDB)

	t.Logf("%d %+v", wrDB.Workflow.WorkflowData.Node.ID, wrDB.WorkflowNodeRuns)
	require.NotNil(t, wrDB.WorkflowNodeRuns[wrDB.Workflow.WorkflowData.Node.ID])
	require.Len(t, wrDB.WorkflowNodeRuns[wrDB.Workflow.WorkflowData.Node.ID], 1)
	secrets, errS := workflow.LoadDecryptSecrets(context.TODO(), db, wrDB, &wrDB.WorkflowNodeRuns[wrDB.Workflow.WorkflowData.Node.ID][0])
	require.NoError(t, errS)

	t.Logf("%+v", secrets)
	require.Len(t, secrets, 6)
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.app-key.priv"))
	require.NotNil(t, sdk.VariableFind(secrets, "cds.app.app-password"))
	require.NotNil(t, sdk.VariableFind(secrets, "cds.integration.token"))
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.env-key.priv"))
	require.NotNil(t, sdk.VariableFind(secrets, "cds.env.env-password"))

	// from project
	require.NotNil(t, sdk.VariableFind(secrets, "cds.integration.mypassword"))
}
