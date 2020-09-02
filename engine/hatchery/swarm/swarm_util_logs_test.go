package swarm

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

var loggerCall = 0

func Test_serviceLogs(t *testing.T) {
	defer gock.Off()
	h := InitTestHatcherySwarm(t)
	reader := rand.Reader
	bitSize := 2048
	key, err := rsa.GenerateKey(reader, bitSize)
	require.NoError(t, err)
	h.Common.PrivateKey = key

	gock.New("https://lolcat.api").Get("/config/cdn").Reply(http.StatusOK).JSON(sdk.CDNConfig{TCPURL: "tcphost:8090"})
	require.NoError(t, h.RefreshServiceLogger(context.TODO()))

	containers := []types.Container{
		{
			ID:    "swarmy-model1-w1",
			Names: []string{"swarmy-model1-w1"},
			Labels: map[string]string{
				"hatchery":    "swarmy",
				"worker_name": "swarmy-model1-w1",
			},
		},
		{
			ID:    "service-1",
			Names: []string{"swarmy-model1-w1"},
			Labels: map[string]string{
				"hatchery":                        "swarmy",
				"worker_name":                     "swarmy-model1-w1",
				hatchery.LabelServiceNodeRunID:    "999",
				hatchery.LabelServiceJobID:        "666",
				hatchery.LabelServiceID:           "1",
				hatchery.LabelServiceWorkflowID:   "1",
				hatchery.LabelServiceWorkflowName: "MyWorkflow",
				hatchery.LabelServiceProjectKey:   "KEY",
				hatchery.LabelServiceRunID:        "1",
				hatchery.LabelServiceNodeRunName:  "Mypip",
				hatchery.LabelServiceJobName:      "MyJob",
			},
		},
	}

	gock.New("https://lolcat.host").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)

	gock.New("https://lolcat.host").AddMatcher(func(r *http.Request, rr *gock.Request) (bool, error) {
		b, err := gock.MatchPath(r, rr)
		assert.NoError(t, err)
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.String(), "https://lolcat.host/v6.66/containers/service-1/logs") {
			if b {
				return true, nil
			}
			return false, nil
		}
		return true, nil
	}).Reply(http.StatusOK).Body(strings.NewReader("Je suis le log"))

	h.ServiceLogger = GetMockLogger()

	loggerCall = 0
	assert.NoError(t, h.getServicesLogs())

	for _, p := range gock.Pending() {
		t.Logf("%+v", p.Request().URLStruct.String())
	}
	require.True(t, gock.IsDone())
	require.Equal(t, 1, loggerCall)
}

func GetMockLogger() *logrus.Logger {
	log := logrus.New()
	log.AddHook(&HookMock{})
	return log
}

type HookMock struct{}

func (h *HookMock) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.InfoLevel,
	}
}
func (h *HookMock) Fire(e *logrus.Entry) error {
	loggerCall++
	return nil
}
