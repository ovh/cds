package kubernetes

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"strings"
	"testing"

	"github.com/ovh/cds/sdk/hatchery"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
)

var loggerCall = 0

func Test_serviceLogs(t *testing.T) {
	t.Cleanup(gock.Off)

	h := NewHatcheryKubernetesTest(t)
	reader := rand.Reader
	bitSize := 2048
	key, err := rsa.GenerateKey(reader, bitSize)
	require.NoError(t, err)
	h.Common.PrivateKey = key

	gock.New("http://lolcat.api").Get("/config/cdn").Reply(http.StatusOK).JSON(sdk.CDNConfig{TCPURL: "tcphost:8090"})
	require.NoError(t, h.RefreshServiceLogger(context.TODO()))

	podsList := v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "kyubi",
					Labels: map[string]string{
						hatchery.LabelServiceJobID:        "666",
						hatchery.LabelServiceNodeRunID:    "999",
						hatchery.LabelServiceWorkflowID:   "1",
						hatchery.LabelServiceWorkflowName: "MyWorkflow",
						hatchery.LabelServiceProjectKey:   "KEY",
						hatchery.LabelServiceRunID:        "1",
						hatchery.LabelServiceNodeRunName:  "Mypip",
						hatchery.LabelServiceJobName:      "MyJob",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "service-666-blabla",
						},
					},
				},
			},
		},
	}
	gock.New("http://lolcat.kube").Get("/api/v1/namespaces/hachibi/pods").Reply(http.StatusOK).JSON(podsList)

	gock.New("http://lolcat.kube").AddMatcher(func(r *http.Request, rr *gock.Request) (bool, error) {
		b, err := gock.MatchPath(r, rr)
		assert.NoError(t, err)
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.String(), "http://lolcat.kube/api/v1/namespaces/hachibi/pods/pod-name/log?container=service-666-blabla") {
			if b {
				return true, nil
			}
			return false, nil
		}
		return true, nil
	}).Reply(http.StatusOK).Body(strings.NewReader("Je suis le log"))

	h.ServiceLogger = GetMockLogger()

	loggerCall = 0
	assert.NoError(t, h.getServicesLogs(context.TODO()))

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
