package swarm

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/sdk"
)

func Test_pullImage(t *testing.T) {
	t.Cleanup(gock.Off)

	h := InitTestHatcherySwarm(t)

	gock.New("https://lolcat.local").Post("/images/create").Times(5).AddMatcher(func(r *http.Request, rr *gock.Request) (bool, error) {
		values := r.URL.Query()

		// Call 1
		if values.Get("fromImage") == "my-registry.lolcat.local/my-image-1" && values.Get("tag") == "my-tag" {
			return true, nil
		}
		// Call 2
		if values.Get("fromImage") == "my-image-2" && values.Get("tag") == "my-tag" {
			return true, nil
		}

		buf, err := base64.StdEncoding.DecodeString(r.Header.Get("X-Registry-Auth"))
		require.NoError(t, err)
		var auth types.AuthConfig
		require.NoError(t, json.Unmarshal(buf, &auth))

		t.Log("Auth config", auth)

		// Call 3
		if values.Get("fromImage") == "my-first-registry.lolcat.local/my-image-3" && values.Get("tag") == "my-tag" &&
			auth.Username == "my-user" && auth.Password == "my-pass-1" && auth.ServerAddress == "my-first-registry.lolcat.local" {
			return true, nil
		}
		// Call 4
		if values.Get("fromImage") == "my-second-registry.lolcat.local/my-image-4" && values.Get("tag") == "my-tag" &&
			auth.Username == "my-user" && auth.Password == "my-pass-2" && auth.ServerAddress == "my-second-registry.lolcat.local" {
			return true, nil
		}
		// Call 5
		if values.Get("fromImage") == "my-image-5" && values.Get("tag") == "my-tag" &&
			auth.Username == "my-user" && auth.Password == "my-pass" && auth.ServerAddress == "docker.io" {
			return true, nil
		}
		return false, nil
	}).Reply(http.StatusOK)

	require.NoError(t, h.pullImage(h.dockerClients["default"], "my-registry.lolcat.local/my-image-1:my-tag", time.Minute, sdk.WorkerStarterWorkerModel{ModelV1: &sdk.Model{}}))
	require.NoError(t, h.pullImage(h.dockerClients["default"], "my-image-2:my-tag", time.Minute, sdk.WorkerStarterWorkerModel{ModelV1: &sdk.Model{}}))

	h.Config.RegistryCredentials = []RegistryCredential{
		{
			Domain:   "docker.io",
			Username: "my-user",
			Password: "my-pass",
		},
		{
			Domain:   "my-first-registry.lolcat.local",
			Username: "my-user",
			Password: "my-pass-1",
		},
		{
			Domain:   "^*.lolcat.local$",
			Username: "my-user",
			Password: "my-pass-2",
		},
	}

	require.NoError(t, h.pullImage(h.dockerClients["default"], "my-first-registry.lolcat.local/my-image-3:my-tag", time.Minute, sdk.WorkerStarterWorkerModel{ModelV1: &sdk.Model{}}))
	require.NoError(t, h.pullImage(h.dockerClients["default"], "my-second-registry.lolcat.local/my-image-4:my-tag", time.Minute, sdk.WorkerStarterWorkerModel{ModelV1: &sdk.Model{}}))
	require.NoError(t, h.pullImage(h.dockerClients["default"], "my-image-5:my-tag", time.Minute, sdk.WorkerStarterWorkerModel{ModelV1: &sdk.Model{}}))

	require.True(t, gock.IsDone())
}
