package api

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

// A plugin is deployed by importing its definition, then uploading its binaries one by one. Jobs
// keep starting during that sequence: the binaries already published must stay served, with a
// descriptor matching the content, until they are replaced.
func Test_pluginDeploymentKeepsBinariesAvailable(t *testing.T) {
	api, db, _ := newTestAPI(t)

	storage, err := objectstore.Init(context.TODO(), objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{Basedir: t.TempDir()},
		},
	})
	require.NoError(t, err)
	api.SharedStorage = storage

	_, jwt := assets.InsertAdminUser(t, db)

	p := sdk.GRPCPlugin{
		Name:        sdk.RandomString(10),
		Type:        sdk.GRPCPluginAction,
		Author:      "me",
		Description: "the description",
	}
	uriCreate := api.Router.GetRoute("POST", api.postGRPCluginHandler, nil)
	test.NotEmpty(t, uriCreate)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uriCreate, &p))
	require.Equal(t, 200, w.Code)

	uploadPluginBinary(t, api, jwt, p.Name, "the binary of the previous release")
	requirePluginBinaryServed(t, api, jwt, p.Name, "the binary of the previous release")

	// the deployment starts by importing the definition again
	imported := p
	imported.Description = "an updated description"
	uriImport := api.Router.GetRouteV2("POST", api.postImportPluginHandler, nil) + "?force=true"
	test.NotEmpty(t, uriImport)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uriImport, &imported))
	require.Equal(t, 200, w.Code)

	requirePluginBinaryServed(t, api, jwt, p.Name, "the binary of the previous release")

	// same for a plugin imported through the v1 route
	uriUpdate := api.Router.GetRoute("PUT", api.putGRPCluginHandler, map[string]string{"name": p.Name})
	test.NotEmpty(t, uriUpdate)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "PUT", uriUpdate, &imported))
	require.Equal(t, 200, w.Code)

	requirePluginBinaryServed(t, api, jwt, p.Name, "the binary of the previous release")

	// then the new binaries are uploaded
	uploadPluginBinary(t, api, jwt, p.Name, "the binary being released")
	requirePluginBinaryServed(t, api, jwt, p.Name, "the binary being released")

	reloaded, err := plugin.LoadByName(context.TODO(), db, p.Name)
	require.NoError(t, err)
	require.Equal(t, "an updated description", reloaded.Description)
	require.Len(t, reloaded.Binaries, 1)
}

func uploadPluginBinary(t *testing.T, api *API, jwt, pluginName, content string) {
	t.Helper()

	sum := sha512.Sum512([]byte(content))
	b := sdk.GRPCPluginBinary{
		OS:          "linux",
		Arch:        "amd64",
		Name:        pluginName + "-linux-amd64",
		Perm:        0700,
		Size:        int64(len(content)),
		SHA512sum:   hex.EncodeToString(sum[:]),
		FileContent: []byte(content),
	}

	uri := api.Router.GetRoute("POST", api.postGRPCluginBinaryHandler, map[string]string{"name": pluginName})
	test.NotEmpty(t, uri)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, &b))
	require.Equal(t, 200, w.Code)
}

// requirePluginBinaryServed checks what a worker gets: the published descriptor, and the content
// served for it.
func requirePluginBinaryServed(t *testing.T, api *API, jwt, pluginName, content string) {
	t.Helper()

	vars := map[string]string{"name": pluginName, "os": "linux", "arch": "amd64"}

	uriInfos := api.Router.GetRoute("GET", api.getGRPCluginBinaryInfosHandler, vars)
	test.NotEmpty(t, uriInfos)
	wInfos := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wInfos, assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uriInfos, nil))
	require.Equal(t, 200, wInfos.Code, "the plugin binary must stay published")

	var b sdk.GRPCPluginBinary
	require.NoError(t, json.Unmarshal(wInfos.Body.Bytes(), &b))

	uriDownload := api.Router.GetRoute("GET", api.getGRPCluginBinaryHandler, vars)
	test.NotEmpty(t, uriDownload)
	wDownload := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDownload, assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uriDownload, nil))
	require.Equal(t, 200, wDownload.Code, "the plugin binary must stay downloadable")
	require.Equal(t, content, wDownload.Body.String())

	sum := sha512.Sum512(wDownload.Body.Bytes())
	require.Equal(t, hex.EncodeToString(sum[:]), b.SHA512sum, "the served content must match the published checksum")
}
