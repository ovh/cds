package kubernetes

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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
						LABEL_HATCHERY_NAME: "kyubi",
					},
				},
				Spec: v1.PodSpec{},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wrong",
					Labels: map[string]string{
						LABEL_HATCHERY_NAME: "jubi",
					},
				},
				Spec: v1.PodSpec{},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "w2",
					Labels: map[string]string{
						LABEL_HATCHERY_NAME: "kyubi",
					},
				},
				Spec: v1.PodSpec{},
			},
		},
	}
	gock.New("http://lolcat.kube").Get("/api/v1/namespaces/hachibi/pods").Reply(http.StatusOK).JSON(podsList)

	ws := h.WorkersStarted(context.TODO())
	require.Equal(t, 2, len(ws))
	require.Equal(t, "w1", ws[0])
	require.Equal(t, "w2", ws[1])
	require.True(t, gock.IsDone())
}

func TestHatcheryKubernetes_Status(t *testing.T) {
	defer gock.Off()
	defer gock.Observe(nil)
	h := NewHatcheryKubernetesTest(t)

	m := &sdk.Model{
		Name: "model1",
		Group: &sdk.Group{
			Name: "group",
		},
	}

	podResponse := v1.Pod{}
	gock.New("http://lolcat.kube").Post("/api/v1/namespaces/hachibi/pods").Reply(http.StatusOK).JSON(podResponse)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		if request.Body == nil {
			return
		}
		bodyContent, err := ioutil.ReadAll(request.Body)
		assert.NoError(t, err)
		var podRequest v1.Pod
		require.NoError(t, json.Unmarshal(bodyContent, &podRequest))

		t.Logf("%s", string(bodyContent))
		require.Equal(t, "hachibi", podRequest.ObjectMeta.Namespace)
		require.Equal(t, "kyubi", podRequest.Labels["CDS_HATCHERY_NAME"])
		require.Equal(t, "666", podRequest.Labels[hatchery.LabelServiceJobID])
		require.Equal(t, "999", podRequest.Labels[hatchery.LabelServiceNodeRunID])
		require.Equal(t, "execution", podRequest.Labels["CDS_WORKER"])
		require.Equal(t, "model1", podRequest.Labels["CDS_WORKER_MODEL"])

		require.Equal(t, 2, len(podRequest.Spec.Containers))
		require.Equal(t, "k8s-toto", podRequest.Spec.Containers[0].Name)
		require.Equal(t, int64(4096), podRequest.Spec.Containers[0].Resources.Requests.Memory().Value())
		require.Equal(t, "service-0-pg", podRequest.Spec.Containers[1].Name)
		require.Equal(t, 1, len(podRequest.Spec.Containers[1].Env))
		require.Equal(t, "PG_USERNAME", podRequest.Spec.Containers[1].Env[0].Name)
		require.Equal(t, "toto", podRequest.Spec.Containers[1].Env[0].Value)
	}
	gock.Observe(checkRequest)

	err := h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{
		JobID:      666,
		NodeRunID:  999,
		Model:      m,
		WorkerName: "k8s-toto",
		Requirements: []sdk.Requirement{
			{
				Name:  "mem",
				Type:  sdk.MemoryRequirement,
				Value: "4096",
			}, {
				Name:  "pg",
				Type:  sdk.ServiceRequirement,
				Value: "postgresql:5.6.7 PG_USERNAME=toto",
			},
		},
	})
	require.NoError(t, err)
	require.True(t, gock.IsDone())
}
