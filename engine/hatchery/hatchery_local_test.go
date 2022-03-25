package hatchery_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

type TestRunner struct {
	t *testing.T
}

func (r *TestRunner) NewCmd(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestHatcheryLocal(t *testing.T) {
	s := InitWebsocketTestServer(t)
	InitMock(t, s.URL)
	defer s.Close()

	defer gock.Off()
	var h = local.New()

	h.LocalWorkerRunner = &TestRunner{t}
	var cfg = local.HatcheryConfiguration{
		Basedir: os.TempDir(),
	}

	cfg.Name = "lolcat-test-hatchery"
	cfg.API.HTTP.Insecure = false
	cfg.API.HTTP.URL = s.URL
	cfg.API.Token = "xxxxxxxx"
	cfg.API.MaxHeartbeatFailures = 0
	cfg.Provision.RegisterFrequency = 1
	cfg.Provision.MaxWorker = 1
	cfg.Provision.InjectEnvVars = []string{"AAA=AAA"}
	privKey, _ := jws.NewRandomRSAKey()
	privKeyPEM, _ := jws.ExportPrivateKey(privKey)
	cfg.RSAPrivateKey = string(privKeyPEM)

	require.NoError(t, h.ApplyConfiguration(cfg))

	srvCfg, err := h.Init(cfg)
	require.NoError(t, err)
	require.NotNil(t, srvCfg)
	t.Logf("service config: %+v", srvCfg)

	gock.New(s.URL).Get("/config/cdn").Times(-1).Reply(200).JSON(sdk.CDNConfig{TCPURL: "tcphost:8090"})

	srvCfg.Hook = func(client cdsclient.Interface) error {
		gock.InterceptClient(client.HTTPClient())
		return nil
	}

	require.NoError(t, h.Signin(context.TODO(), srvCfg, cfg))
	require.NoError(t, h.Start(context.TODO()))

	// Wait 30 sec to let the queue polling exec run one time
	serveCtx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	err = h.Serve(serveCtx)
	require.Contains(t, err.Error(), "Server closed")

	heartbeatCtx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	err = h.Heartbeat(heartbeatCtx, h.Status)
	require.Contains(t, err.Error(), "context deadline exceeded")

	// Mock assertions

	t.Logf("Checking mock assertions")

	if !gock.IsDone() {
		pending := gock.Pending()
		for _, m := range pending {
			if m.Request().URLStruct.String() != s.URL+"/services/heartbeat" &&
				!strings.HasPrefix(m.Request().URLStruct.String(), s.URL+"/download/worker") &&
				!strings.HasPrefix(m.Request().URLStruct.String(), s.URL+"/config/cdn") &&
				!strings.HasPrefix(m.Request().URLStruct.String(), s.URL+"/worker") {
				t.Errorf("PENDING %s %s", m.Request().Method, m.Request().URLStruct.String())
			}
		}
	}
	if gock.HasUnmatchedRequest() {
		reqs := gock.GetUnmatchedRequests()
		for _, req := range reqs {
			t.Logf("Request %s %s unmatched", req.Method, req.URL.String())
		}
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	t.Log(os.Environ())
	time.Sleep(30 * time.Second)
}
