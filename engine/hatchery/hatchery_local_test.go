package hatchery_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ovh/cds/sdk/jws"

	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

type TestRunner struct {
	t *testing.T
}

func (r *TestRunner) NewCmd(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestHatcheryLocal(t *testing.T) {
	InitMock(t)
	defer gock.Off()

	var h = local.New()

	h.LocalWorkerRunner = &TestRunner{t}

	var cfg = local.HatcheryConfiguration{
		Basedir: os.TempDir(),
	}

	cfg.Name = "lolcat-test-hatchery"
	cfg.API.HTTP.Insecure = false
	cfg.API.HTTP.URL = "http://lolcat.host"
	cfg.API.Token = "xxxxxxxx"
	cfg.API.MaxHeartbeatFailures = 0
	cfg.Provision.Frequency = 1
	cfg.Provision.RegisterFrequency = 1
	cfg.Provision.MaxWorker = 1
	privKey, _ := jws.NewRandomRSAKey()
	privKeyPEM, _ := jws.ExportPrivateKey(privKey)
	cfg.RSAPrivateKey = string(privKeyPEM)

	err := h.ApplyConfiguration(cfg)
	require.NoError(t, err)

	srvCfg, err := h.Init(cfg)
	require.NotNil(t, srvCfg)
	// srvCfg.Verbose = true
	t.Logf("service config: %+v", srvCfg)

	srvCfg.Hook = func(client cdsclient.Interface) error {
		client.HTTPSSEClient().Transport = newMockSSERoundTripper(t, context.TODO())
		gock.InterceptClient(client.HTTPSSEClient())
		gock.InterceptClient(client.HTTPClient())
		return nil
	}

	err = h.Start(context.TODO(), srvCfg)
	require.NoError(t, err)

	var srvConfig sdk.ServiceConfig
	b, _ := json.Marshal(cfg)
	json.Unmarshal(b, &srvConfig) // nolint

	err = h.Register(context.TODO(), srvConfig)
	require.NoError(t, err)

	heartbeatCtx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	err = h.Heartbeat(heartbeatCtx, h.Status)
	require.Contains(t, "context deadline exceeded", err.Error())

	serveCtx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	err = h.Serve(serveCtx)
	require.Contains(t, "context deadline exceeded", err.Error())

	// Mock assertions

	if !gock.IsDone() {
		pending := gock.Pending()
		for _, m := range pending {
			t.Errorf("PENDING %s %s", m.Request().Method, m.Request().URLStruct.String())
		}
	}
	assert.False(t, gock.HasUnmatchedRequest(), "gock should not have unmatched request")
	if gock.HasUnmatchedRequest() {
		reqs := gock.GetUnmatchedRequests()
		for _, req := range reqs {
			t.Logf("Request %s %s unmatched", req.Method, req.URL.String())
		}
	}
}

func TestHelperProcess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
}
