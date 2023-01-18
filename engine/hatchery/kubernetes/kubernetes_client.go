package kubernetes

import (
	"context"
	"os"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	logNS  = log.Field("k8s_ns")
	logPod = log.Field("k8s_pod")
)

func init() {
	log.RegisterField(logNS, logPod)
}

func initKubeClient(config HatcheryConfiguration) (KubernetesClient, error) {
	k8sTimeout := time.Second * 10

	if config.KubernetesConfigFile != "" {
		cfg, err := clientcmd.BuildConfigFromFlags(config.KubernetesMasterURL, config.KubernetesConfigFile)
		if err != nil {
			return nil, sdk.WrapError(err, "Cannot build config from flags")
		}
		cfg.Timeout = k8sTimeout

		clientSet, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, sdk.WrapError(err, "Cannot create client with newForConfig")
		}
		return &kubernetesClient{clientSet}, nil
	}

	if config.KubernetesMasterURL != "" {
		configK8s, err := clientcmd.BuildConfigFromKubeconfigGetter(config.KubernetesMasterURL, getStartingConfig(config))
		if err != nil {
			return nil, sdk.WrapError(err, "Cannot build config from config getter")
		}
		configK8s.Timeout = k8sTimeout

		if config.KubernetesCertAuthData != "" {
			configK8s.TLSClientConfig = rest.TLSClientConfig{
				CAData:   []byte(config.KubernetesCertAuthData),
				CertData: []byte(config.KubernetesClientCertData),
				KeyData:  []byte(config.KubernetesClientKeyData),
			}
		}

		// creates the clientset
		clientSet, err := kubernetes.NewForConfig(configK8s)
		if err != nil {
			return nil, sdk.WrapError(err, "Cannot create new config")
		}

		return &kubernetesClient{clientSet}, nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to configure k8s InClusterConfig")
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to configure k8s client with InClusterConfig")
	}

	return &kubernetesClient{clientSet}, nil
}

// getStartingConfig implements ConfigAccess
func getStartingConfig(config HatcheryConfiguration) func() (*clientcmdapi.Config, error) {
	return func() (*clientcmdapi.Config, error) {
		defaultClientConfigRules := clientcmd.NewDefaultClientConfigLoadingRules()
		overrideCfg := clientcmd.ConfigOverrides{
			AuthInfo: clientcmdapi.AuthInfo{
				Username: config.KubernetesUsername,
				Password: config.KubernetesPassword,
				Token:    config.KubernetesToken,
			},
		}

		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(defaultClientConfigRules, &overrideCfg)
		rawConfig, err := clientConfig.RawConfig()
		if os.IsNotExist(err) {
			return clientcmdapi.NewConfig(), nil
		}
		if err != nil {
			return nil, err
		}

		return &rawConfig, nil
	}
}

type KubernetesClient interface {
	PodCreate(ctx context.Context, ns string, spec *corev1.Pod, options metav1.CreateOptions) (*corev1.Pod, error)
	PodDelete(ctx context.Context, ns string, name string, options metav1.DeleteOptions) error
	PodGetRawLogs(ctx context.Context, ns string, name string, options *corev1.PodLogOptions) ([]byte, error)
	PodList(ctx context.Context, ns string, options metav1.ListOptions) (*corev1.PodList, error)
	SecretCreate(ctx context.Context, ns string, spec *corev1.Secret, options metav1.CreateOptions) (*corev1.Secret, error)
	SecretDelete(ctx context.Context, ns string, name string, options metav1.DeleteOptions) error
	SecretGet(ctx context.Context, ns string, name string, options metav1.GetOptions) (*corev1.Secret, error)
	SecretList(ctx context.Context, ns string, options metav1.ListOptions) (*corev1.SecretList, error)
}

type kubernetesClient struct {
	client *kubernetes.Clientset
}

var (
	_ KubernetesClient = new(kubernetesClient)
)

func (k *kubernetesClient) PodCreate(ctx context.Context, ns string, spec *corev1.Pod, options metav1.CreateOptions) (*corev1.Pod, error) {
	ctx, end := telemetry.Span(ctx, "kubernetesClient.PodCreate")
	defer end()
	ctx = context.WithValue(ctx, logNS, ns)
	ctx = context.WithValue(ctx, logPod, spec.Name)
	log.Info(ctx, "creating pod %s", spec.Name)
	pod, err := k.client.CoreV1().Pods(ns).Create(ctx, spec, options)
	return pod, sdk.WrapError(err, "unable to create pod %s", spec.Name)
}

func (k *kubernetesClient) PodDelete(ctx context.Context, ns string, name string, options metav1.DeleteOptions) error {
	ctx = context.WithValue(ctx, logNS, ns)
	ctx = context.WithValue(ctx, logPod, name)
	log.Info(ctx, "deleting pod %s", name)
	err := k.client.CoreV1().Pods(ns).Delete(ctx, name, options)
	return sdk.WrapError(err, "unable to delete pod %s", name)
}

func (k *kubernetesClient) PodList(ctx context.Context, ns string, opts metav1.ListOptions) (*corev1.PodList, error) {
	ctx = context.WithValue(ctx, logNS, ns)
	log.Info(ctx, "listing pod in namespace %s", ns)
	pods, err := k.client.CoreV1().Pods(ns).List(ctx, opts)
	return pods, sdk.WrapError(err, "unable to list pods in namespace %s", ns)
}

func (k *kubernetesClient) SecretCreate(ctx context.Context, ns string, spec *corev1.Secret, options metav1.CreateOptions) (*corev1.Secret, error) {
	secret, err := k.client.CoreV1().Secrets(ns).Create(ctx, spec, options)
	return secret, sdk.WrapError(err, "unable to create secret %s", spec.Name)
}

func (k *kubernetesClient) SecretDelete(ctx context.Context, ns string, name string, options metav1.DeleteOptions) error {
	err := k.client.CoreV1().Secrets(ns).Delete(ctx, name, options)
	return sdk.WrapError(err, "unable to delete secret %s", name)
}

func (k *kubernetesClient) SecretGet(ctx context.Context, ns string, name string, options metav1.GetOptions) (*corev1.Secret, error) {
	secret, err := k.client.CoreV1().Secrets(ns).Get(ctx, name, options)
	return secret, sdk.WrapError(err, "unable to get secret %s", name)
}

func (k *kubernetesClient) SecretList(ctx context.Context, ns string, options metav1.ListOptions) (*corev1.SecretList, error) {
	secrets, err := k.client.CoreV1().Secrets(ns).List(ctx, options)
	return secrets, sdk.WrapError(err, "unable to list secrets in namespace %s", ns)
}

func (k *kubernetesClient) PodGetRawLogs(ctx context.Context, ns string, name string, options *corev1.PodLogOptions) ([]byte, error) {
	ctx = context.WithValue(ctx, logNS, ns)
	ctx = context.WithValue(ctx, logPod, name)
	log.Debug(ctx, "get logs for pod %s", name)
	logs, err := k.client.CoreV1().Pods(ns).GetLogs(name, options).DoRaw(ctx)
	return logs, sdk.WrapError(err, "unable to get pod %s raw logs", name)
}
