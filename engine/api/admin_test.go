package api

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func Test_getAdminDatabaseSignatureResume(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureResume, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("%s", w.Body.String())
}

func Test_getAdminDatabaseSignatureTuplesByPrimaryKey(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureResume, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resume = sdk.CanonicalFormUsageResume{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resume))

	for entity, data := range resume {

		for i := range data {

			vars := map[string]string{
				"entity": entity,
				"signer": data[i].Signer,
			}

			uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureTuplesBySigner, vars)
			req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

			// Do the request
			w := httptest.NewRecorder()
			api.Router.Mux.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)

			var pks []string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

			assert.Len(t, pks, int(data[i].Number))
		}
	}
}

func Test_postAdminDatabaseSignatureRollEntityByPrimaryKey(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureResume, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resume = sdk.CanonicalFormUsageResume{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resume))

	for entity, data := range resume {

		for i := range data {

			vars := map[string]string{
				"entity": entity,
				"signer": data[i].Signer,
			}

			uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureTuplesBySigner, vars)
			req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

			// Do the request
			w := httptest.NewRecorder()
			api.Router.Mux.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)

			var pks []string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

			for _, pk := range pks {
				vars := map[string]string{
					"entity": entity,
					"pk":     pk,
				}

				uri := api.Router.GetRoute("POST", api.postAdminDatabaseSignatureRollEntityByPrimaryKey, vars)
				req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)

				// Do the request
				w := httptest.NewRecorder()
				api.Router.Mux.ServeHTTP(w, req)
				assert.Equal(t, 204, w.Code)
			}
		}
	}
}

func Test_getAdminDatabaseEncryptedEntities(t *testing.T) {
	gorpmapping.Register(gorpmapping.New(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseEncryptedEntities, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("%s", w.Body.String())
}

func Test_getAdminDatabaseEncryptedTuplesByEntity(t *testing.T) {
	gorpmapping.Register(gorpmapping.New(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseEncryptedTuplesByEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("%s", w.Body.String())
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKey(t *testing.T) {
	gorpmapping.Register(gorpmapping.New(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseEncryptedTuplesByEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var res []string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

	for _, s := range res {
		uri := api.Router.GetRoute("POST", api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "gorpmapper.TestEncryptedData", "pk": s})
		req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)

		// Do the request
		w := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(w, req)
		assert.Equal(t, 204, w.Code)
	}
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKeyForApplication(t *testing.T) {
	api, db, _ := newTestAPI(t)
	_, jwt := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(6), sdk.RandomString(6))

	pf := sdk.IntegrationModel{
		Name:       "test-deploy-post-2" + proj.Key,
		Deployment: true,
		DefaultConfig: sdk.IntegrationConfig{
			"token": sdk.IntegrationConfigValue{
				Type: sdk.IntegrationConfigTypePassword,
			},
			"url": sdk.IntegrationConfigValue{
				Type: sdk.IntegrationConfigTypeString,
			},
		},
	}
	require.NoError(t, integration.InsertModel(db, &pf))
	defer func() { _ = integration.DeleteModel(context.TODO(), db, pf.ID) }()

	pp := sdk.ProjectIntegration{
		Model:              pf,
		Name:               pf.Name,
		IntegrationModelID: pf.ID,
		ProjectID:          proj.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &pp))

	var err error
	proj, err = project.Load(context.Background(), db, proj.Key, project.LoadOptions.WithIntegrations)
	require.NoError(t, err)

	for i := range proj.Integrations {
		if proj.Integrations[i].Name == pf.Name {
			pp = proj.Integrations[i]
			break
		}
	}

	app := &sdk.Application{
		Name:               "my-amm",
		RepositoryFullname: "ovh/cds",
		RepositoryStrategy: sdk.RepositoryStrategy{
			ConnectionType: "https",
			User:           "foo",
			Password:       "bar",
		},
	}
	require.NoError(t, application.Insert(db, *proj, app))

	var pfConfig = sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token",
		},
		"url": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypeString,
			Value: "my-url",
		},
	}
	err = application.SetDeploymentStrategy(db, proj.ID, app.ID, pp.Model.ID, pp.Name, pfConfig)
	require.NoError(t, err)

	app, err = application.LoadByName(context.Background(), db, proj.Key, app.Name, application.LoadOptions.WithClearDeploymentStrategies)
	require.NoError(t, err)

	adsID, err := db.SelectInt("select id from application_deployment_strategy where application_id = $1 and project_integration_id = $2", app.ID, pp.ID)
	require.NoError(t, err)

	t.Logf("deployment strategies before rollover: %+v", app.DeploymentStrategies)

	uri := api.Router.GetRoute("POST", api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "application.dbApplicationDeploymentStrategy", "pk": fmt.Sprintf("%d", adsID)})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)
	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

	app2, err := application.LoadByName(context.Background(), db, proj.Key, app.Name, application.LoadOptions.WithClearDeploymentStrategies)
	require.NoError(t, err)
	t.Logf("deployment strategies after rollover: %+v", app2.DeploymentStrategies)

	require.Equal(t, app.DeploymentStrategies, app2.DeploymentStrategies)
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKeyForWorkerModelSecret(t *testing.T) {
	api, db, _ := newTestAPI(t)

	//Loading all models
	models, errlw := workermodel.LoadAll(context.Background(), api.mustDB(), nil)
	if errlw != nil {
		t.Fatalf("Error getting models : %s", errlw)
	}

	//Delete all of them
	for _, m := range models {
		if err := workermodel.DeleteByID(api.mustDB(), m.ID); err != nil {
			t.Fatalf("Error deleting model : %s", err)
		}
	}

	//Create admin user
	u, jwt := assets.InsertAdminUser(t, db)
	assert.NotZero(t, u)
	assert.NotZero(t, jwt)

	g, err := group.LoadByName(context.TODO(), api.mustDB(), "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Shell: "sh -c",
			Cmd:   "worker --api={{.API}}",
			Envs: map[string]string{
				"CDS_TEST": "THIS IS A TEST",
			},
			Private:  true,
			Username: "test",
			Password: "pwtest",
		},
		RegisteredCapabilities: sdk.RequirementList{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := api.Router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	require.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
	require.Equal(t, "THIS IS A TEST", newModel.ModelDocker.Envs["CDS_TEST"], "Worker model envs are not good")
	require.Equal(t, "{{.secrets.registry_password}}", newModel.ModelDocker.Password)

	secrets, err := workermodel.LoadSecretsByModelID(context.TODO(), api.mustDB(), newModel.ID)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "secrets.registry_password", secrets[0].Name)
	assert.Equal(t, "pwtest", secrets[0].Value)

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "workermodel.workerModelSecret", "pk": secrets[0].ID})
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)
	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

	secrets, err = workermodel.LoadSecretsByModelID(context.TODO(), api.mustDB(), newModel.ID)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "secrets.registry_password", secrets[0].Name)
	assert.Equal(t, "pwtest", secrets[0].Value)

}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKeyForProjectIntegration(t *testing.T) {
	api, db, router := newTestAPI(t)
	u, jwt := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(6), sdk.RandomString(6))

	integrationModel, err := integration.LoadModelByName(context.TODO(), db, sdk.KafkaIntegration.Name)
	if err != nil {
		assert.NoError(t, integration.CreateBuiltinModels(context.TODO(), api.mustDB()))
		models, _ := integration.LoadModels(db)
		assert.True(t, len(models) > 0)
	}

	integrationModel, err = integration.LoadModelByName(context.TODO(), db, sdk.AWSIntegration.Name)
	test.NoError(t, err)

	pp := sdk.ProjectIntegration{
		Name:               "test",
		Config:             sdk.AWSIntegration.DefaultConfig.Clone(),
		IntegrationModelID: integrationModel.ID,
	}

	for k, v := range pp.Config {
		v.Value = sdk.RandomString(5)
		pp.Config[k] = v
	}

	t.Logf("%+v", pp.Config)

	// ADD integration
	vars := map[string]string{}
	vars[permProjectKey] = proj.Key
	uri := router.GetRoute("POST", api.postProjectIntegrationHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, pp)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	require.NoError(t, err)

	integ, err := integration.LoadIntegrationsByProjectIDWithClearPassword(context.TODO(), db, proj.ID)
	t.Logf("%+v", integ[0].Config)
	require.NoError(t, err)

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "integration.dbProjectIntegration", "pk": fmt.Sprintf("%d", integ[0].ID)})
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)
	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

	integ2, err := integration.LoadIntegrationsByProjectIDWithClearPassword(context.TODO(), db, proj.ID)
	require.NoError(t, err)

	t.Logf("%+v", integ2[0].Config)

	require.Len(t, integ2[0].Config, len(pp.Config))
	for k, v := range pp.Config {
		assert.Equal(t, integ2[0].Config[k], v)
	}
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKeyForWorkflowRunSecrets(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	projKey := sdk.ProjectKey{
		Name:      "proj-sshkey",
		Type:      sdk.KeySSHParameter,
		Public:    "publicssh-proj",
		Private:   "privatessh-proj",
		Builtin:   false,
		ProjectID: proj.ID,
		KeyID:     "key-id-proj",
	}
	require.NoError(t, project.InsertKey(db, &projKey))

	pwdProject := sdk.ProjectVariable{
		Name:  "projvar",
		Type:  sdk.SecretVariable,
		Value: "myprojpassword",
	}
	require.NoError(t, project.InsertVariable(db, proj.ID, &pwdProject, u))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
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

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(api.mustDB(), s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2)
	s.Jobs = append(s.Jobs, *j)

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
		Name:               sdk.RandomString(10),
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

	require.NoError(t, plugin.Insert(db, &p))
	assert.NotEqual(t, 0, p.ID)

	app := sdk.Application{
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
		Name:       sdk.RandomString(10),
		Variables: []sdk.ApplicationVariable{
			{
				Name:  "app-password",
				Type:  sdk.SecretVariable,
				Value: "apppassword",
			},
			{
				Name:  "app-clear",
				Type:  sdk.StringVariable,
				Value: "apppassword",
			},
		},
		Keys: []sdk.ApplicationKey{
			{
				Type:    sdk.KeySSHParameter,
				Name:    "app-sshkey",
				Private: "private-key",
				Public:  "public-key",
				KeyID:   "id",
			},
		},
		DeploymentStrategies: map[string]sdk.IntegrationConfig{
			projInt.Name: map[string]sdk.IntegrationConfigValue{
				"token": {
					Type:        "password",
					Value:       "app-token",
					Description: "token",
				},
				"notoken": {
					Type:        "string",
					Value:       "app-token",
					Description: "token",
				},
			},
		},
	}
	require.NoError(t, application.Insert(db, *proj, &app))
	require.NoError(t, application.InsertVariable(db, app.ID, &app.Variables[0], u))
	app.Keys[0].ApplicationID = app.ID
	require.NoError(t, application.InsertKey(db, &app.Keys[0]))
	require.NoError(t, application.SetDeploymentStrategy(db, proj.ID, app.ID, modelIntegration.ID, projInt.Name, app.DeploymentStrategies[projInt.Name]))

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		Variables: []sdk.EnvironmentVariable{
			{
				Name:  "env-password",
				Type:  sdk.SecretVariable,
				Value: "envpassword",
			},
			{
				Name:  "env-data",
				Type:  sdk.StringVariable,
				Value: "coucou",
			},
		},
		Keys: []sdk.EnvironmentKey{
			{
				Type:    sdk.KeySSHParameter,
				Name:    "env-sshkey",
				Private: "private-key-env",
				Public:  "public-key-env",
				KeyID:   "id-env",
			},
		},
	}
	require.NoError(t, environment.InsertEnvironment(db, &env))
	require.NoError(t, environment.InsertVariable(db, env.ID, &env.Variables[0], u))
	env.Keys[0].EnvironmentID = env.ID
	require.NoError(t, environment.InsertKey(db, &env.Keys[0]))

	proj2, errP := project.Load(context.TODO(), api.mustDB(), key,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
		project.LoadOptions.WithIntegrations,
	)
	require.NoError(t, errP)

	wf := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:           pip.ID,
					ApplicationID:        app.ID,
					EnvironmentID:        env.ID,
					ProjectIntegrationID: proj2.Integrations[0].ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "child",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
	}

	proj2, errP = project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &wf))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj2, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":                      proj.Key,
		"permWorkflowNameAdvanced": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Payload: map[string]string{
				"test": "hereismytest",
			},
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	wr := &sdk.WorkflowRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)
	require.Equal(t, int64(0), wr.LastSubNumber)

	// wait for the workflow to finish crafting
	assert.NoError(t, waitCraftinWorkflow(t, api, db, wr.ID))

	lastRun, err := workflow.LoadLastRun(context.Background(), api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	assert.NotNil(t, lastRun.RootRun())
	payloadCount := 0
	testFound := false
	for _, param := range lastRun.RootRun().BuildParameters {
		if param.Name == "payload" {
			payloadCount++
		} else if param.Name == "test" {
			testFound = true
		}
	}
	assert.Equal(t, 1, payloadCount)
	assert.True(t, testFound, "should find 'test' in build parameters")

	secretsRaw, err := workflow.LoadDecryptSecrets(context.TODO(), db, lastRun, lastRun.RootRun())
	require.NoError(t, err)

	secrets := secretsRaw.ToVariables()

	t.Logf("%+v", secrets)

	// Proj key
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.proj-sshkey.priv"))
	sshkeypriv := sdk.VariableFind(secrets, "cds.key.proj-sshkey.priv")
	// Project password
	require.NotNil(t, sdk.VariableFind(secrets, "cds.proj.projvar"))
	projvar := sdk.VariableFind(secrets, "cds.proj.projvar")

	// Proj Integration
	require.NotNil(t, sdk.VariableFind(secrets, "cds.integration.deployment.mypassword"))

	// Application variable
	require.Nil(t, sdk.VariableFind(secrets, "cds.app.app-clear"))
	require.NotNil(t, sdk.VariableFind(secrets, "cds.app.app-password"))
	// Application key
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.app-sshkey.priv"))
	// Application integration
	require.NotNil(t, sdk.VariableFind(secrets, "cds.integration.deployment.token"))
	require.Nil(t, sdk.VariableFind(secrets, "cds.integration.deployment.notoken"))

	// Env variable
	require.NotNil(t, sdk.VariableFind(secrets, "cds.env.env-password"))
	require.Nil(t, sdk.VariableFind(secrets, "cds.env.env-data"))
	// En  key
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.env-sshkey.priv"))

	// Check public and id key in node run param
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.proj-sshkey.pub"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.proj-sshkey.id"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.app-sshkey.pub"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.app-sshkey.id"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.env-sshkey.pub"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.env-sshkey.id"))

	// Rollover
	uri = api.Router.GetRoute("GET", api.getAdminDatabaseEncryptedTuplesByEntity, map[string]string{"entity": "workflow.dbWorkflowRunSecret"})
	req = assets.NewJWTAuthentifiedRequest(t, pass, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var res []string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

	for _, s := range res {
		uri := api.Router.GetRoute("POST", api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "workflow.dbWorkflowRunSecret", "pk": s})
		req := assets.NewJWTAuthentifiedRequest(t, pass, "POST", uri, nil)
		w := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(w, req)
		assert.Equal(t, 204, w.Code)
	}

	// Rerun
	//Prepare request
	vars = map[string]string{
		"key":                      proj.Key,
		"permWorkflowNameAdvanced": w1.Name,
	}
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts = &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Payload: map[string]string{
				"test": "hereismytest",
			},
		},
		Number:      &lastRun.Number,
		FromNodeIDs: []int64{lastRun.RootRun().WorkflowNodeID},
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	wr = &sdk.WorkflowRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	require.Equal(t, int64(1), wr.Number)

	// wait for the workflow to finish crafting
	assert.NoError(t, waitCraftinWorkflow(t, api, db, wr.ID))

	lastRun, err = workflow.LoadLastRun(context.Background(), api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	require.Equal(t, int64(1), lastRun.LastSubNumber)

	assert.NotNil(t, lastRun.RootRun())
	payloadCount = 0
	testFound = false
	for _, param := range lastRun.RootRun().BuildParameters {
		if param.Name == "payload" {
			payloadCount++
		} else if param.Name == "test" {
			testFound = true
		}
	}
	assert.Equal(t, 1, payloadCount)
	assert.True(t, testFound, "should find 'test' in build parameters")

	secretsRaw, err = workflow.LoadDecryptSecrets(context.TODO(), db, lastRun, lastRun.RootRun())
	require.NoError(t, err)

	secrets = secretsRaw.ToVariables()

	t.Logf("%+v", secrets)

	// Proj key
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.proj-sshkey.priv"))
	require.NotEmpty(t, sdk.VariableFind(secrets, "cds.key.proj-sshkey.priv").Value)

	// Project password
	require.NotNil(t, sdk.VariableFind(secrets, "cds.proj.projvar"))
	require.NotEmpty(t, sdk.VariableFind(secrets, "cds.proj.projvar").Value)

	// Proj Integration
	require.NotNil(t, sdk.VariableFind(secrets, "cds.integration.deployment.mypassword"))
	require.NotEmpty(t, sdk.VariableFind(secrets, "cds.integration.deployment.mypassword").Value)

	// Application variable
	require.Nil(t, sdk.VariableFind(secrets, "cds.app.app-clear"))
	require.NotNil(t, sdk.VariableFind(secrets, "cds.app.app-password"))
	require.NotEmpty(t, sdk.VariableFind(secrets, "cds.app.app-password").Value)

	// Application key
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.app-sshkey.priv"))
	require.NotEmpty(t, sdk.VariableFind(secrets, "cds.key.app-sshkey.priv").Value)

	// Application integration
	require.NotNil(t, sdk.VariableFind(secrets, "cds.integration.deployment.token"))
	require.NotEmpty(t, sdk.VariableFind(secrets, "cds.integration.deployment.token").Value)

	require.Nil(t, sdk.VariableFind(secrets, "cds.integration.deployment.notoken"))

	// Env variable
	require.NotNil(t, sdk.VariableFind(secrets, "cds.env.env-password"))
	require.NotEmpty(t, sdk.VariableFind(secrets, "cds.env.env-password").Value)

	require.Nil(t, sdk.VariableFind(secrets, "cds.env.env-data"))
	// Env  key
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.env-sshkey.priv"))
	require.NotEmpty(t, sdk.VariableFind(secrets, "cds.key.env-sshkey.priv").Value)

	// Check public and id key in node run param
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.proj-sshkey.pub"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.proj-sshkey.id"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.app-sshkey.pub"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.app-sshkey.id"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.env-sshkey.pub"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.env-sshkey.id"))

	require.Equal(t, sshkeypriv, sdk.VariableFind(secrets, "cds.key.proj-sshkey.priv"))
	require.Equal(t, projvar, sdk.VariableFind(secrets, "cds.proj.projvar"))

}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKeyForProjectVCS(t *testing.T) {
	api, db, _ := newTestAPI(t)
	_, jwt := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(6), sdk.RandomString(6))

	vcsProject := &sdk.VCSProject{
		Name:      "myserver",
		Created:   time.Now(),
		Type:      sdk.VCSTypeGitea,
		URL:       "http://localhost:3000",
		CreatedBy: "sgu",
		ProjectID: proj.ID,
		Auth: sdk.VCSAuthProject{
			Username:      "myuser",
			Token:         "mytoken",
			SSHPrivateKey: "myprivatekey",
		},
	}
	require.NoError(t, vcs.Insert(context.TODO(), db, vcsProject))

	vcsProject, err := vcs.LoadVCSByID(context.TODO(), db, proj.Key, vcsProject.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	uri := api.Router.GetRoute("POST", api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "vcs.dbVCSProject", "pk": fmt.Sprintf("%s", vcsProject.ID)})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)
	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

	vcsProject2, err := vcs.LoadVCSByID(context.TODO(), db, proj.Key, vcsProject.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	require.Equal(t, vcsProject.Auth.SSHPrivateKey, vcsProject2.Auth.SSHPrivateKey)
	require.Equal(t, vcsProject.Auth.Token, vcsProject2.Auth.Token)
	require.Equal(t, vcsProject.Auth.Username, vcsProject2.Auth.Username)
}

func Test_postWorkflowMaxRunHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	workflow.SetMaxRuns(15)

	_, jwt := assets.InsertAdminUser(t, db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	w := assets.InsertTestWorkflow(t, db, api.Cache, p, sdk.RandomString(10))

	require.Equal(t, int64(15), w.MaxRuns)

	uri := api.Router.GetRoute("POST", api.postWorkflowMaxRunHandler, map[string]string{"key": p.Key, "permWorkflowName": w.Name})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, sdk.UpdateMaxRunRequest{MaxRuns: 5})

	// Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)

	wfDb, err := workflow.Load(context.TODO(), db, api.Cache, *p, w.Name, workflow.LoadOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(5), wfDb.MaxRuns)

	wfDb.MaxRuns = 20
	require.NoError(t, workflow.Update(context.TODO(), db, api.Cache, *p, wfDb, workflow.UpdateOptions{}))

	// Max runs must not be updated
	wfDb2, err := workflow.Load(context.TODO(), db, api.Cache, *p, w.Name, workflow.LoadOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(5), wfDb2.MaxRuns)
}
