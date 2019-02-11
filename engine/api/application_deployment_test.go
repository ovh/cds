package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func Test_getApplicationDeploymentStrategiesConfigHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()
	u, pass := assets.InsertAdminUser(api.mustDB())
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, u))

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
	}

	uri := router.GetRoute("GET", api.getApplicationDeploymentStrategiesConfigHandler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_postApplicationDeploymentStrategyConfigHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()
	u, pass := assets.InsertAdminUser(api.mustDB())
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, u))

	pf := sdk.IntegrationModel{
		Name:       "test-deploy-2",
		Deployment: true,
	}
	test.NoError(t, integration.InsertModel(db, &pf))
	defer func() { _ = integration.DeleteModel(db, pf.ID) }()

	pp := sdk.ProjectIntegration{
		Model:              pf,
		Name:               pf.Name,
		IntegrationModelID: pf.ID,
		ProjectID:          proj.ID,
		Config: sdk.IntegrationConfig{
			"token": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypePassword,
				Value: "my-secret-token",
			},
			"url": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypeString,
				Value: "my-url",
			},
		},
	}
	test.NoError(t, integration.InsertIntegration(db, &pp))

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
		"integration":     pf.Name,
	}

	uri := router.GetRoute("POST", api.postApplicationDeploymentStrategyConfigHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token",
		},
		"url": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypeString,
			Value: "my-url",
		},
	})

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	//Then we try to update
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token-2",
		},
		"url": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypeString,
			Value: "my-url-2",
		},
	})

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	uri = router.GetRoute("GET", api.getApplicationDeploymentStrategyConfigHandler, vars)
	//Then we try to update
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	cfg := sdk.IntegrationConfig{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &cfg))
	assert.Equal(t, sdk.PasswordPlaceholder, cfg["token"].Value)

	// with clear paswword
	uri = router.GetRoute("GET", api.getApplicationDeploymentStrategyConfigHandler, vars)
	// Then we try to update
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	q := req.URL.Query()
	q.Set("withClearPassword", "true")
	req.URL.RawQuery = q.Encode()

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	cfg2 := sdk.IntegrationConfig{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &cfg2))
	assert.Equal(t, "my-secret-token-2", cfg2["token"].Value)

	// with clear paswword
	uri = router.GetRoute("DELETE", api.deleteApplicationDeploymentStrategyConfigHandler, vars)
	// Then we try to update
	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)
	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	uri = router.GetRoute("GET", api.getApplicationDeploymentStrategyConfigHandler, vars)
	//Then we try to update
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func Test_postApplicationDeploymentStrategyConfigHandler_InsertTwoDifferentIntegrations(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()
	u, pass := assets.InsertAdminUser(api.mustDB())
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, u))

	pf := sdk.IntegrationModel{
		Name:       "test-deploy-2",
		Deployment: true,
	}
	test.NoError(t, integration.InsertModel(db, &pf))
	defer func() { _ = integration.DeleteModel(db, pf.ID) }()

	pp := sdk.ProjectIntegration{
		Model:              pf,
		Name:               pf.Name,
		IntegrationModelID: pf.ID,
		ProjectID:          proj.ID,
		Config: sdk.IntegrationConfig{
			"token": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypePassword,
				Value: "my-secret-token",
			},
			"url": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypeString,
				Value: "my-url",
			},
		},
	}
	test.NoError(t, integration.InsertIntegration(db, &pp))

	pp2 := sdk.ProjectIntegration{
		Model:              pf,
		Name:               pf.Name + "-2",
		IntegrationModelID: pf.ID,
		ProjectID:          proj.ID,
		Config: sdk.IntegrationConfig{
			"token": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypePassword,
				Value: "my-secret-token",
			},
			"url": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypeString,
				Value: "my-url",
			},
		},
	}
	test.NoError(t, integration.InsertIntegration(db, &pp2))

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
		"integration":     pf.Name,
	}

	uri := router.GetRoute("POST", api.postApplicationDeploymentStrategyConfigHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token",
		},
		"url": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypeString,
			Value: "my-url",
		},
	})

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	//Now add a new
	vars = map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
		"integration":     pp2.Name,
	}

	uri = router.GetRoute("POST", api.postApplicationDeploymentStrategyConfigHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token",
		},
		"url": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypeString,
			Value: "my-url",
		},
	})

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	vars = map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
	}
	uri = router.GetRoute("GET", api.getApplicationDeploymentStrategiesConfigHandler, vars)
	//Then we try to update
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	cfg := map[string]sdk.IntegrationConfig{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &cfg))
	assert.Len(t, cfg, 2)

}

func Test_deleteApplicationDeploymentStrategyConfigHandler(t *testing.T) {
	//see Test_postApplicationDeploymentStrategyConfigHandler
}

func Test_postApplicationDeploymentStrategyConfigHandlerAsProvider(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	api.Config.Providers = append(api.Config.Providers, ProviderConfiguration{
		Name:  "test-provider",
		Token: "my-token",
	})

	u, _ := assets.InsertAdminUser(api.mustDB())
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey, u)
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, u))

	pf := sdk.IntegrationModel{
		Name:       "test-deploy-3",
		Deployment: true,
		DeploymentDefaultConfig: sdk.IntegrationConfig{
			"token": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypePassword,
				Value: "my-secret-token",
			},
			"url": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypeString,
				Value: "my-url",
			},
		},
	}
	test.NoError(t, integration.InsertModel(api.mustDB(), &pf))
	defer func() { _ = integration.DeleteModel(api.mustDB(), pf.ID) }()

	pp := sdk.ProjectIntegration{
		Model:              pf,
		Name:               pf.Name,
		IntegrationModelID: pf.ID,
		ProjectID:          proj.ID,
	}
	test.NoError(t, integration.InsertIntegration(api.mustDB(), &pp))

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Name:  "test-provider",
		Token: "my-token",
	})

	err := sdkclient.ApplicationDeploymentStrategyUpdate(proj.Key, app.Name, pf.Name, sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token-2",
		},
	})
	test.NoError(t, err)

	cfg, err := application.LoadDeploymentStrategies(api.mustDB(), app.ID, true)
	test.NoError(t, err)

	var assertCfg = func(key string, cfg sdk.IntegrationConfig, expected sdk.IntegrationConfigValue) {
		actual, has := cfg[key]
		assert.True(t, has, "%s not found", key)
		assert.Equal(t, expected.Value, actual.Value)
		assert.Equal(t, expected.Type, actual.Type)
	}

	pfcfg, has := cfg[pf.Name]
	assert.True(t, has, "%s not found", pf.Name)
	assertCfg("token", pfcfg,
		sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token-2",
		})

	assertCfg("url", pfcfg,
		sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypeString,
			Value: "my-url",
		})

}
