package kubernetes

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
)

func TestHatcheryKubernetes_KillAwolWorkers(t *testing.T) {
	defer gock.Off()
	h := NewHatcheryKubernetesTest(t)

	podsList := v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "worker-1",
					Namespace: "cds-workers",
					Labels: map[string]string{
						LABEL_HATCHERY_NAME: "my-hatchery",
						LABEL_WORKER_NAME:   "worker-1",
					},
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "worker-2",
					Namespace: "cds-workers",
					Labels: map[string]string{
						LABEL_HATCHERY_NAME: "my-hatchery",
						LABEL_WORKER_NAME:   "worker-2",
					},
				},
				Spec: v1.PodSpec{},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "worker-3",
					Namespace: "cds-workers",
					Labels: map[string]string{
						LABEL_HATCHERY_NAME: "my-hatchery",
						LABEL_WORKER_NAME:   "worker-3",
					},
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "Error",
								},
							},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "worker-4",
					Namespace: "cds-workers",
					Labels: map[string]string{
						LABEL_HATCHERY_NAME: "my-hatchery",
						LABEL_WORKER_NAME:   "worker-4",
					},
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason: "ErrImagePull",
								},
							},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "worker-5",
					Namespace: "cds-workers",
					Labels: map[string]string{
						LABEL_HATCHERY_NAME: "my-hatchery",
						LABEL_WORKER_NAME:   "worker-5",
					},
				},
			},
		},
	}
	gock.New("http://lolcat.kube").Get("/api/v1/namespaces/cds-workers/pods").Reply(http.StatusOK).JSON(podsList)

	gock.New("http://lolcat.kube").Delete("/api/v1/namespaces/cds-workers/pods/worker-1").Reply(http.StatusOK).JSON(nil)
	gock.New("http://lolcat.kube").Delete("/api/v1/namespaces/cds-workers/pods/worker-2").Reply(http.StatusOK).JSON(nil)
	gock.New("http://lolcat.kube").Delete("/api/v1/namespaces/cds-workers/pods/worker-3").Reply(http.StatusOK).JSON(nil)
	gock.New("http://lolcat.kube").Delete("/api/v1/namespaces/cds-workers/pods/worker-4").Reply(http.StatusOK).JSON(nil)

	gock.New("http://lolcat.api").Get("/worker").Reply(http.StatusOK).JSON([]sdk.Worker{{
		Name: "worker-5",
	}})

	err := h.killAwolWorkers(context.TODO())
	require.NoError(t, err)
	require.True(t, gock.IsDone())
}
