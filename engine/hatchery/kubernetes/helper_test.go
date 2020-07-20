package kubernetes

import (
	"github.com/ovh/cds/sdk"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/ovh/cds/sdk/cdsclient"
)

func NewHatcheryKubernetesTest(t *testing.T) *HatcheryKubernetes {
	h := new(HatcheryKubernetes)
	h.Client = cdsclient.New(cdsclient.Config{Host: "http://lolcat.api", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(h.Client.(cdsclient.Raw).HTTPClient())

	clientSet, errCl := kubernetes.NewForConfig(&rest.Config{Host: "http://lolcat.kube"})
	require.NoError(t, errCl)

	h.k8sClient = clientSet
	gock.InterceptClient(h.k8sClient.CoreV1().RESTClient().(*rest.RESTClient).Client)

	h.Config.Name = "kyubi"
	h.Config.Namespace = "hachibi"
	h.ServiceInstance = &sdk.Service{
		CanonicalService: sdk.CanonicalService{
			ID:   1,
			Name: "kyubi",
		},
	}
	return h
}
