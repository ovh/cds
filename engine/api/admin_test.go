package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/notification_v2"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/gorpmapper"
	engine_test "github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func Test_getAdminOrganizationCRUD(t *testing.T) {
	api, db, _ := newTestAPI(t)
	_, jwt := assets.InsertAdminUser(t, db)

	orga := sdk.Organization{Name: sdk.RandomString(10)}
	uri := api.Router.GetRoute("POST", api.postAdminOrganizationHandler, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, &orga))
	require.Equal(t, 201, w.Code)

	uriList := api.Router.GetRoute("GET", api.getAdminOrganizationsHandler, nil)
	wList := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wList, assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uriList, nil))

	var orgs []sdk.Organization
	require.Equal(t, 200, wList.Code)
	body := wList.Body.Bytes()
	require.NoError(t, json.Unmarshal(body, &orgs))

	require.Equal(t, 2, len(orgs))

	uriDelete := api.Router.GetRoute("DELETE", api.deleteAdminOrganizationsHandler, map[string]string{"organizationIdentifier": orgs[1].Name})
	reqDelete := assets.NewJWTAuthentifiedRequest(t, jwt, "DELETE", uriDelete, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)

	orgsDb, err := organization.LoadOrganizations(context.TODO(), db)
	require.NoError(t, err)
	require.Equal(t, 1, len(orgsDb))
}

func Test_getAdminDatabaseEntityList(t *testing.T) {
	api, db, _ := newTestAPI(t)
	_, jwt := assets.InsertAdminUser(t, db)
	gorpmapping.Register(gorpmapping.New(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	var d1 = gorpmapper.TestEncryptedData{
		Data:                 "data-1",
		SensitiveData:        "sensitive-data-1",
		AnotherSensitiveData: "another-sensitive-data-1",
		SensitiveJsonData:    gorpmapper.SensitiveJsonData{Data: "json-sentitive-data-1"},
	}
	require.NoError(t, gorpmapping.InsertAndSign(context.TODO(), &engine_test.FakeTransaction{
		DbMap: api.mustDB(),
	}, &d1))

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseEntityList, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil))
	require.Equal(t, 200, w.Code)

	var res []sdk.DatabaseEntity
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

	var found bool
	for _, entity := range res {
		if entity.Name == "gorpmapper.TestEncryptedData" {
			found = true
			require.True(t, entity.Encrypted, "gorpmapper.TestEncryptedData entity should be encrypted")
			require.True(t, entity.Signed, "gorpmapper.TestEncryptedData entity should be signed")
			require.True(t, len(entity.CanonicalForms) >= 1)
			break
		}
	}
	require.True(t, found, "gorpmapper.TestEncryptedData entity should be listed")
}

func Test_getAdminDatabaseEntity(t *testing.T) {
	api, db, _ := newTestAPI(t)
	_, jwt := assets.InsertAdminUser(t, db)
	gorpmapping.Register(gorpmapping.New(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	var d1 = gorpmapper.TestEncryptedData{
		Data:                 "data-1",
		SensitiveData:        "sensitive-data-1",
		AnotherSensitiveData: "another-sensitive-data-1",
		SensitiveJsonData:    gorpmapper.SensitiveJsonData{Data: "json-sentitive-data-1"},
	}
	require.NoError(t, gorpmapping.InsertAndSign(context.TODO(), &engine_test.FakeTransaction{
		DbMap: api.mustDB(),
	}, &d1))

	var d2 = gorpmapper.TestEncryptedData{
		Data:                 "canonical-variant-data-2",
		SensitiveData:        "sensitive-data-2",
		AnotherSensitiveData: "another-sensitive-data-2",
		SensitiveJsonData:    gorpmapper.SensitiveJsonData{Data: "json-sentitive-data-2"},
	}
	require.NoError(t, gorpmapping.InsertAndSign(context.TODO(), &engine_test.FakeTransaction{
		DbMap: api.mustDB(),
	}, &d2))

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil))
	require.Equal(t, 200, w.Code)
	var pks []string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

	var d1Found, d2Found bool
	for _, pk := range pks {
		if pk == strconv.FormatInt(d1.ID, 10) {
			d1Found = true
		}
		if pk == strconv.FormatInt(d2.ID, 10) {
			d2Found = true
		}
	}
	require.True(t, d1Found, "gorpmapper.TestEncryptedData d1 entity pk should be listed")
	require.True(t, d2Found, "gorpmapper.TestEncryptedData d2 entity pk should be listed")

	lastestCanonicalForm, _ := d2.Canonical().Latest()
	sha := gorpmapper.GetSigner(lastestCanonicalForm)
	uri = api.Router.GetRoute("GET", api.getAdminDatabaseEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil, cdsclient.Signer(sha)))
	require.Equal(t, 200, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

	d1Found, d2Found = false, false
	for _, pk := range pks {
		if pk == strconv.FormatInt(d1.ID, 10) {
			d1Found = true
		}
		if pk == strconv.FormatInt(d2.ID, 10) {
			d2Found = true
		}
	}
	require.False(t, d1Found, "gorpmapper.TestEncryptedData d1 entity pk should not be listed")
	require.True(t, d2Found, "gorpmapper.TestEncryptedData d2 entity pk should be listed")
}

func Test_postAdminDatabaseRollEncryptedEntityForApplication(t *testing.T) {
	api, db, _ := newTestAPI(t)
	admin, jwt := assets.InsertAdminUser(t, db)

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
	t.Cleanup(func() { _ = integration.DeleteModel(context.TODO(), db, pf.ID) })

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

	// application.dbApplication
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

	uri := api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "application.dbApplication"})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{fmt.Sprintf("%d", app.ID)}))
	require.Equal(t, 200, w.Code)

	app, err = application.LoadByNameWithClearVCSStrategyPassword(context.Background(), db, proj.Key, app.Name)
	require.NoError(t, err)
	require.Equal(t, "bar", app.RepositoryStrategy.Password)

	// application.dbApplicationDeploymentStrategy
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
	require.NoError(t, application.SetDeploymentStrategy(db, proj.ID, app.ID, pp.Model.ID, pp.Name, pfConfig))

	app, err = application.LoadByName(context.Background(), db, proj.Key, app.Name, application.LoadOptions.WithClearDeploymentStrategies)
	require.NoError(t, err)

	adsID, err := db.SelectInt("select id from application_deployment_strategy where application_id = $1 and project_integration_id = $2", app.ID, pp.ID)
	require.NoError(t, err)

	t.Logf("deployment strategies before rollover: %+v", app.DeploymentStrategies)

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "application.dbApplicationDeploymentStrategy"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{fmt.Sprintf("%d", adsID)}))
	require.Equal(t, 200, w.Code)

	app2, err := application.LoadByName(context.Background(), db, proj.Key, app.Name, application.LoadOptions.WithClearDeploymentStrategies)
	require.NoError(t, err)
	t.Logf("deployment strategies after rollover: %+v", app2.DeploymentStrategies)

	require.Equal(t, app.DeploymentStrategies, app2.DeploymentStrategies)
	tBefore, err := app.DeploymentStrategies["token"].Value()
	require.NoError(t, err)
	tAfter, err := app2.DeploymentStrategies["token"].Value()
	require.NoError(t, err)
	require.Equal(t, tBefore, tAfter)

	// application.dbApplicationVariable
	vari := sdk.ApplicationVariable{
		Name:  "secret",
		Type:  "string",
		Value: "bar",
	}
	require.NoError(t, application.InsertVariable(db, app.ID, &vari, admin))

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "application.dbApplicationVariable"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{fmt.Sprintf("%d", vari.ID)}))
	require.Equal(t, 200, w.Code)

	vari2, err := application.LoadVariableWithDecryption(context.TODO(), db, app.ID, vari.ID, "secret")
	require.NoError(t, err)
	require.Equal(t, "bar", vari2.Value)

	// application.dbApplicationKey
	k := &sdk.ApplicationKey{
		Name:          "mykey",
		Type:          "pgp",
		ApplicationID: app.ID,
	}
	pgpK, err := keys.GeneratePGPKeyPair(k.Name, "", "test@cds")
	if err != nil {
		t.Fatal(err)
	}
	k.Public = pgpK.Public
	k.Private = pgpK.Private
	k.KeyID = pgpK.KeyID
	if err := application.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "application.dbApplicationKey"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{fmt.Sprintf("%d", k.ID)}))
	require.Equal(t, 200, w.Code)

	keys, err := application.LoadAllKeysWithPrivateContent(context.TODO(), db, app.ID)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, pgpK.Private, keys[0].Private)
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKeyForWorkerModelSecret(t *testing.T) {
	api, db, _ := newTestAPI(t)
	_, jwt := assets.InsertAdminUser(t, db)

	models, err := workermodel.LoadAll(context.Background(), api.mustDB(), nil)
	require.NoError(t, err)
	for _, m := range models {
		require.NoError(t, workermodel.DeleteByID(api.mustDB(), m.ID))
	}

	g, err := group.LoadByName(context.TODO(), api.mustDB(), "shared.infra")
	require.NoError(t, err)

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

	uri := api.Router.GetRoute("POST", api.postWorkerModelHandler, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model))
	require.Equal(t, 200, w.Code)

	var newModel sdk.Model
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	require.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
	require.Equal(t, "THIS IS A TEST", newModel.ModelDocker.Envs["CDS_TEST"], "Worker model envs are not good")
	require.Equal(t, "{{.secrets.registry_password}}", newModel.ModelDocker.Password)

	secrets, err := workermodel.LoadSecretsByModelID(context.TODO(), api.mustDB(), newModel.ID)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	require.Equal(t, "secrets.registry_password", secrets[0].Name)
	require.Equal(t, "pwtest", secrets[0].Value)

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "workermodel.workerModelSecret"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{secrets[0].ID}))
	require.Equal(t, 200, w.Code)

	secrets, err = workermodel.LoadSecretsByModelID(context.TODO(), api.mustDB(), newModel.ID)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	require.Equal(t, "secrets.registry_password", secrets[0].Name)
	require.Equal(t, "pwtest", secrets[0].Value)
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKeyForProjectIntegration(t *testing.T) {
	api, db, router := newTestAPI(t)
	u, jwt := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(6), sdk.RandomString(6))

	integrationModel, err := integration.LoadModelByName(context.TODO(), db, sdk.KafkaIntegration.Name)
	if err != nil {
		require.NoError(t, integration.CreateBuiltinModels(context.TODO(), api.mustDB()))
		models, _ := integration.LoadModels(db)
		require.True(t, len(models) > 0)
	}

	integrationModel, err = integration.LoadModelByName(context.TODO(), db, sdk.AWSIntegration.Name)
	require.NoError(t, err)

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

	uri := router.GetRoute("POST", api.postProjectIntegrationHandler, map[string]string{permProjectKey: proj.Key})
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, pp))
	require.Equal(t, 200, w.Code)

	integ, err := integration.LoadIntegrationsByProjectIDWithClearPassword(context.TODO(), db, proj.ID)
	require.NoError(t, err)
	t.Logf("%+v", integ[0].Config)

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "integration.dbProjectIntegration"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{fmt.Sprintf("%d", integ[0].ID)}))
	require.Equal(t, 200, w.Code)

	integ2, err := integration.LoadIntegrationsByProjectIDWithClearPassword(context.TODO(), db, proj.ID)
	require.NoError(t, err)
	t.Logf("%+v", integ2[0].Config)

	require.Len(t, integ2[0].Config, len(pp.Config))
	for k, v := range pp.Config {
		require.Equal(t, integ2[0].Config[k], v)
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
	require.NotEqual(t, 0, p.ID)

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

	uri := router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":                      proj.Key,
		"permWorkflowNameAdvanced": w1.Name,
	})
	require.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Payload: map[string]string{
				"test": "hereismytest",
			},
		},
	}
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts))
	require.Equal(t, 202, rec.Code)

	wr := &sdk.WorkflowRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	require.Equal(t, int64(1), wr.Number)
	require.Equal(t, int64(0), wr.LastSubNumber)

	// wait for the workflow to finish crafting
	require.NoError(t, waitCraftinWorkflow(t, api, db, wr.ID))

	lastRun, err := workflow.LoadLastRun(context.Background(), api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{})
	require.NoError(t, err)
	require.NotNil(t, lastRun.RootRun())
	payloadCount := 0
	testFound := false
	for _, param := range lastRun.RootRun().BuildParameters {
		if param.Name == "payload" {
			payloadCount++
		} else if param.Name == "test" {
			testFound = true
		}
	}
	require.Equal(t, 1, payloadCount)
	require.True(t, testFound, "should find 'test' in build parameters")

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
	uri = api.Router.GetRoute("GET", api.getAdminDatabaseEntity, map[string]string{"entity": "workflow.dbWorkflowRunSecret"})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, pass, "GET", uri, nil))
	require.Equal(t, 200, w.Code)

	var res []string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

	for _, s := range res {
		uri := api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "workflow.dbWorkflowRunSecret"})
		req := assets.NewJWTAuthentifiedRequest(t, pass, "POST", uri, []string{s})
		w := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code)
	}

	// Rerun
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":                      proj.Key,
		"permWorkflowNameAdvanced": w1.Name,
	})
	opts = &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Payload: map[string]string{
				"test": "hereismytest",
			},
		},
		Number:      &lastRun.Number,
		FromNodeIDs: []int64{lastRun.RootRun().WorkflowNodeID},
	}
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts))
	require.Equal(t, 202, rec.Code)

	wr = &sdk.WorkflowRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	require.Equal(t, int64(1), wr.Number)

	// wait for the workflow to finish crafting
	require.NoError(t, waitCraftinWorkflow(t, api, db, wr.ID))

	lastRun, err = workflow.LoadLastRun(context.Background(), api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(1), lastRun.LastSubNumber)

	require.NotNil(t, lastRun.RootRun())
	payloadCount = 0
	testFound = false
	for _, param := range lastRun.RootRun().BuildParameters {
		if param.Name == "payload" {
			payloadCount++
		} else if param.Name == "test" {
			testFound = true
		}
	}
	require.Equal(t, 1, payloadCount)
	require.True(t, testFound, "should find 'test' in build parameters")

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

func Test_postAdminDatabaseRollEncryptedEntityForProject(t *testing.T) {
	api, db, _ := newTestAPI(t)
	admin, jwt := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(6), sdk.RandomString(6))

	// vcs.dbVCSProject
	vcsProject := &sdk.VCSProject{
		Name:      "myserver",
		Created:   time.Now(),
		Type:      sdk.VCSTypeGitea,
		URL:       "http://localhost:3000",
		CreatedBy: "sgu",
		ProjectID: proj.ID,
		Auth: sdk.VCSAuthProject{
			Username: "myuser",
			Token:    "mytoken",
		},
	}
	require.NoError(t, vcs.Insert(context.TODO(), db, vcsProject))

	vcsProject, err := vcs.LoadVCSByIDAndProjectKey(context.TODO(), db, proj.Key, vcsProject.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	uri := api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "vcs.dbVCSProject"})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{vcsProject.ID}))
	require.Equal(t, 200, w.Code)

	vcsProject2, err := vcs.LoadVCSByIDAndProjectKey(context.TODO(), db, proj.Key, vcsProject.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	require.Equal(t, vcsProject.Auth.Token, vcsProject2.Auth.Token)
	require.Equal(t, vcsProject.Auth.Username, vcsProject2.Auth.Username)

	// project.dbProjectVariable
	v := &sdk.ProjectVariable{
		Name:  "secret",
		Type:  sdk.StringVariable,
		Value: "myvalue",
	}
	require.NoError(t, project.InsertVariable(db, proj.ID, v, admin))

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "project.dbProjectVariable"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{fmt.Sprintf("%d", v.ID)}))
	require.Equal(t, 200, w.Code)

	vAfter, err := project.LoadVariableWithDecryption(db, proj.ID, v.ID, "secret")
	require.NoError(t, err)
	require.Equal(t, "myvalue", vAfter.Value)

	// project.dbProjectKey
	k := &sdk.ProjectKey{
		Name:      "mykey",
		Type:      "pgp",
		ProjectID: proj.ID,
	}
	pgpK, err := keys.GeneratePGPKeyPair(k.Name, "", "test@cds")
	require.NoError(t, err)
	k.Public = pgpK.Public
	k.Private = pgpK.Private
	k.KeyID = pgpK.KeyID
	require.NoError(t, project.InsertKey(db, k))

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "project.dbProjectKey"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{fmt.Sprintf("%d", k.ID)}))
	require.Equal(t, 200, w.Code)

	keys, err := project.LoadAllKeysWithPrivateContent(context.TODO(), db, proj.ID)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, pgpK.Private, keys[0].Private)

	// notification_v2.dbProjectNotification
	n := &sdk.ProjectNotification{
		Name:       "mynotif",
		ProjectKey: proj.Key,
		Auth: sdk.ProjectNotificationAuth{
			Headers: map[string]string{
				"secret": "value",
			},
		},
	}
	require.NoError(t, notification_v2.Insert(context.TODO(), db, n))

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "notification_v2.dbProjectNotification"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{n.ID}))
	require.Equal(t, 200, w.Code)

	n, err = notification_v2.LoadByName(context.TODO(), db, proj.Key, "mynotif", gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, "value", n.Auth.Headers["secret"])

	// project.dbProjectVariableSetItemSecret
	vs := sdk.ProjectVariableSet{
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	its := &sdk.ProjectVariableSetItem{
		ProjectVariableSetID: vs.ID,
		Name:                 sdk.RandomString(10),
		Type:                 sdk.ProjectVariableTypeSecret,
		Value:                "mySecretValue",
	}
	require.NoError(t, project.InsertVariableSetItemSecret(context.TODO(), db, its))

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "project.dbProjectVariableSetItemSecret"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{its.ID}))
	require.Equal(t, 200, w.Code)

	its, err = project.LoadVariableSetItem(context.TODO(), db, vs.ID, its.Name, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, "mySecretValue", its.Value)
}

func Test_postWorkflowMaxRunHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	_, jwt := assets.InsertAdminUser(t, db)
	workflow.SetMaxRuns(15)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	w := assets.InsertTestWorkflow(t, db, api.Cache, p, sdk.RandomString(10))

	require.Equal(t, int64(15), w.MaxRuns)

	uri := api.Router.GetRoute("POST", api.postWorkflowMaxRunHandler, map[string]string{"key": p.Key, "permWorkflowName": w.Name})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, sdk.UpdateMaxRunRequest{MaxRuns: 5})

	// Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

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

func Test_postAdminDatabaseRollEncryptedEntityForEnvironment(t *testing.T) {
	api, db, _ := newTestAPI(t)
	admin, jwt := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(6), sdk.RandomString(6))

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "my-env",
	}
	require.NoError(t, environment.InsertEnvironment(api.mustDB(), &env))

	// environment.dbEnvironmentVariable
	v := &sdk.EnvironmentVariable{
		Name:  "secret",
		Type:  sdk.StringVariable,
		Value: "myvalue",
	}
	require.NoError(t, environment.InsertVariable(db, env.ID, v, admin))

	uri := api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "environment.dbEnvironmentVariable"})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{fmt.Sprintf("%d", v.ID)}))
	require.Equal(t, 200, w.Code)

	vAfter, err := environment.LoadVariableWithDecryption(db, env.ID, v.ID, "secret")
	require.NoError(t, err)
	require.Equal(t, "myvalue", vAfter.Value)

	// environment.dbEnvironmentKey
	k := &sdk.EnvironmentKey{
		Name:          "mykey",
		Type:          "pgp",
		EnvironmentID: env.ID,
	}
	pgpK, err := keys.GeneratePGPKeyPair(k.Name, "", "test@cds")
	require.NoError(t, err)
	k.Public = pgpK.Public
	k.Private = pgpK.Private
	k.KeyID = pgpK.KeyID
	require.NoError(t, environment.InsertKey(db, k))

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseEntityRoll, map[string]string{"entity": "environment.dbEnvironmentKey"})
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, []string{fmt.Sprintf("%d", k.ID)}))
	require.Equal(t, 200, w.Code)

	keys, err := environment.LoadAllKeysWithPrivateContent(db, env.ID)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, pgpK.Private, keys[0].Private)
}
