package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func NewHatcheryKubernetesTest(t *testing.T) *HatcheryKubernetes {
	h := new(HatcheryKubernetes)
	h.Client = cdsclient.New(cdsclient.Config{Host: "http://lolcat.api", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(h.Client.(cdsclient.Raw).HTTPClient())

	clientSet, errCl := kubernetes.NewForConfig(&rest.Config{Host: "http://lolcat.kube"})
	require.NoError(t, errCl)

	h.kubeClient = &kubernetesClient{clientSet}
	gock.InterceptClient(clientSet.CoreV1().RESTClient().(*rest.RESTClient).Client)

	h.Config.Name = "my-hatchery"
	h.Config.Namespace = "cds-workers"
	h.ServiceInstance = &sdk.Service{
		CanonicalService: sdk.CanonicalService{
			ID:   1,
			Name: "my-hatchery",
		},
	}
	return h
}
