package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func Test_getApplicationDeploymentStrategiesConfigHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, *proj, app))

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
	}

	uri := router.GetRoute("GET", api.getApplicationDeploymentStrategiesConfigHandler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	require.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_postApplicationDeploymentStrategyConfigHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, *proj, app))

	pf := sdk.IntegrationModel{
		Name:       "test-deploy-post-2" + pkey,
		Deployment: true,
	}
	require.NoError(t, integration.InsertModel(db, &pf))
	defer func() { _ = integration.DeleteModel(context.TODO(), db, pf.ID) }()

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
	require.NoError(t, integration.InsertIntegration(db, &pp))

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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cfg))
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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cfg2))
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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, *proj, app))

	pf := sdk.IntegrationModel{
		Name:       "test-deploy-TwoDifferentIntegrations-2" + pkey,
		Deployment: true,
	}
	require.NoError(t, integration.InsertModel(db, &pf))
	defer func() { _ = integration.DeleteModel(context.TODO(), db, pf.ID) }()

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
	require.NoError(t, integration.InsertIntegration(db, &pp))

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
	require.NoError(t, integration.InsertIntegration(db, &pp2))

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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cfg))
	assert.Len(t, cfg, 2)

}

func Test_postApplicationDeploymentStrategyConfigHandlerAsProvider(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	u, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOpts := builtin.NewConsumerOptions{
		Name:        sdk.RandomString(10),
		Description: sdk.RandomString(10),
		Duration:    0,
		GroupIDs:    u.GetGroupIDs(),
		Scopes:      sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject),
	}

	_, jws, err := builtin.NewConsumer(context.TODO(), db, consumerOpts, localConsumer)
	require.NoError(t, err)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, *proj, app))

	pf := sdk.IntegrationModel{
		Name:       "test-deploy-3" + pkey,
		Deployment: true,
		AdditionalDefaultConfig: sdk.IntegrationConfig{
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
	require.NoError(t, integration.InsertModel(db, &pf))
	defer func() { _ = integration.DeleteModel(context.TODO(), api.mustDB(), pf.ID) }()

	pp := sdk.ProjectIntegration{
		Model:              pf,
		Name:               pf.Name,
		IntegrationModelID: pf.ID,
		ProjectID:          proj.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &pp))

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Token: jws,
	})

	err = sdkclient.ApplicationDeploymentStrategyUpdate(proj.Key, app.Name, pf.Name, sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token-2",
		},
	})
	require.NoError(t, err)

	cfg, err := application.LoadDeploymentStrategies(context.TODO(), api.mustDB(), app.ID, true)
	require.NoError(t, err)

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
