package kubernetes

import (
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"testing"
)

func TestHatcheryKubernetes_KillAwolWorkers(t *testing.T) {
	defer gock.Off()
	h := NewHatcheryKubernetesTest(t)

	podsList := v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "w1",
					Namespace: "kyubi",
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
					Name:      "wrong",
					Namespace: "kyubi",
				},
				Spec: v1.PodSpec{},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "w2",
					Namespace: "kyubi",
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
					Name:      "w3",
					Namespace: "kyubi",
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
		},
	}
	gock.New("http://lolcat.kube").Get("/api/v1/namespaces/hachibi/pods").Reply(http.StatusOK).JSON(podsList)

	gock.New("http://lolcat.kube").Delete("/api/v1/namespaces/kyubi/pods/w1").Reply(http.StatusOK).JSON(nil)
	gock.New("http://lolcat.kube").Delete("/api/v1/namespaces/kyubi/pods/w2").Reply(http.StatusOK).JSON(nil)
	gock.New("http://lolcat.kube").Delete("/api/v1/namespaces/kyubi/pods/w3").Reply(http.StatusOK).JSON(nil)

	err := h.killAwolWorkers()
	require.NoError(t, err)
	require.True(t, gock.IsDone())
}
