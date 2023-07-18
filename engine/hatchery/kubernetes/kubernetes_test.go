package kubernetes

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

func TestHatcheryKubernetes_WorkersStarted(t *testing.T) {
	defer gock.Off()
	h := NewHatcheryKubernetesTest(t)

	podsList := v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "w1",
					Labels: map[string]string{
						LABEL_HATCHERY_NAME: "my-hatchery",
					},
				},
				Spec: v1.PodSpec{},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "w2",
					Labels: map[string]string{
						LABEL_HATCHERY_NAME: "my-hatchery",
					},
				},
				Spec: v1.PodSpec{},
			},
		},
	}
	gock.New("http://lolcat.kube").Get("/api/v1/namespaces/cds-workers/pods").Reply(http.StatusOK).JSON(podsList)

	ws, err := h.WorkersStarted(context.TODO())
	require.NoError(t, err)
	require.Equal(t, 2, len(ws))
	require.Equal(t, "w1", ws[0])
	require.Equal(t, "w2", ws[1])
	require.True(t, gock.IsDone())
}

func TestHatcheryKubernetes_Status(t *testing.T) {
	defer gock.Off()
	defer gock.Observe(nil)
	h := NewHatcheryKubernetesTest(t)
	h.Config.HatcheryCommonConfiguration.Provision.InjectEnvVars = []string{"PROVISION_ENV=MYVALUE"}

	m := &sdk.Model{
		Name: "model1",
		Group: &sdk.Group{
			Name: "group",
		},
	}

	gock.New("http://lolcat.kube").Delete("/api/v1/namespaces/cds-workers/secrets/cds-worker-config-my-worker").Reply(http.StatusOK)
	gock.New("http://lolcat.kube").Post("/api/v1/namespaces/cds-workers/secrets").Reply(http.StatusOK).JSON(v1.Pod{})
	gock.New("http://lolcat.kube").Post("/api/v1/namespaces/cds-workers/pods").Reply(http.StatusOK).JSON(v1.Pod{})

	gock.Observe(func(request *http.Request, mock gock.Mock) {
		t.Logf("%s %s", request.URL, request.Method)
		bodyContent, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		t.Logf("%s", string(bodyContent))

		if request.Method == http.MethodPost && strings.HasPrefix(request.URL.String(), "http://lolcat.kube/api/v1/namespaces/cds-workers/secrets") {
			var secretRequest v1.Secret
			require.NoError(t, json.Unmarshal(bodyContent, &secretRequest))

			require.Equal(t, "Secret", secretRequest.Kind)
			require.Equal(t, "cds-worker-config-my-worker", secretRequest.Name)
			require.Equal(t, "my-hatchery", secretRequest.Labels[LABEL_HATCHERY_NAME])
			require.Equal(t, "my-worker", secretRequest.Labels[LABEL_WORKER_NAME])
		}

		if request.Method == http.MethodPost && strings.HasPrefix(request.URL.String(), "http://lolcat.kube/api/v1/namespaces/cds-workers/pods") {
			var podRequest v1.Pod
			require.NoError(t, json.Unmarshal(bodyContent, &podRequest))

			require.Equal(t, "Pod", podRequest.Kind)
			require.Equal(t, "cds-workers", podRequest.ObjectMeta.Namespace)
			require.Equal(t, "my-hatchery", podRequest.Labels[LABEL_HATCHERY_NAME])
			require.Equal(t, "666", podRequest.Labels[hatchery.LabelServiceJobID])
			require.Equal(t, "999", podRequest.Labels[hatchery.LabelServiceNodeRunID])
			require.Equal(t, "my-worker", podRequest.Labels[LABEL_WORKER_NAME])
			require.Equal(t, "group-model1", podRequest.Labels[LABEL_WORKER_MODEL_PATH])

			require.Equal(t, 2, len(podRequest.Spec.Containers))
			require.Equal(t, "my-worker", podRequest.Spec.Containers[0].Name)
			require.Equal(t, int64(4096000000), podRequest.Spec.Containers[0].Resources.Requests.Memory().Value())
			var foundEnv, foundSecret bool
			for _, env := range podRequest.Spec.Containers[0].Env {
				if env.Name == "PROVISION_ENV" && env.Value == "MYVALUE" {
					foundEnv = true
				}
				if env.Name == "CDS_CONFIG" && env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == "cds-worker-config-my-worker" {
					foundSecret = true
				}
			}
			require.True(t, foundEnv, "\"PROVISION_ENV\" not found in env variables")
			require.True(t, foundSecret, "\"CDS_CONFIG\" not found in env variables")
			require.Equal(t, "service-0-pg", podRequest.Spec.Containers[1].Name)
			require.Equal(t, 1, len(podRequest.Spec.Containers[1].Env))
			require.Equal(t, "PG_USERNAME", podRequest.Spec.Containers[1].Env[0].Name)
			require.Equal(t, "username", podRequest.Spec.Containers[1].Env[0].Value)
		}
	})

	err := h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{
		JobID:      "666",
		NodeRunID:  999,
		Model:      sdk.WorkerStarterWorkerModel{ModelV1: m},
		WorkerName: "my-worker",
		Requirements: []sdk.Requirement{
			{
				Name:  "mem",
				Type:  sdk.MemoryRequirement,
				Value: "4096",
			}, {
				Name:  "pg",
				Type:  sdk.ServiceRequirement,
				Value: "postgresql:5.6.7 PG_USERNAME=username",
			},
		},
	})
	require.NoError(t, err)
	require.True(t, gock.IsDone())
}
