package kubernetes

import (
	"context"
	"fmt"

	"github.com/rockbears/log"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
)

func (h *HatcheryKubernetes) deleteSecrets(ctx context.Context) error {
	pods, err := h.kubeClient.PodList(ctx, h.Config.Namespace, metav1.ListOptions{LabelSelector: LABEL_SECRET})
	if err != nil {
		return sdk.WrapError(err, "cannot get pods with secret")
	}

	secrets, err := h.kubeClient.SecretList(ctx, h.Config.Namespace, metav1.ListOptions{LabelSelector: LABEL_SECRET})
	if err != nil {
		return sdk.WrapError(err, "cannot get secrets")
	}

	for _, secret := range secrets.Items {
		found := false
		for _, pod := range pods.Items {
			labels := pod.GetLabels()
			if labels != nil && labels[LABEL_SECRET] == secret.Name {
				found = true
				break
			}
		}
		if !found {
			if err := h.kubeClient.SecretDelete(ctx, h.Config.Namespace, secret.Name, metav1.DeleteOptions{}); err != nil {
				log.Error(ctx, "deleteSecrets> Cannot delete secret %s : %v", secret.Name, err)
			}
		}
	}

	return nil
}

func (h *HatcheryKubernetes) createSecret(ctx context.Context, secretName string, model sdk.Model) error {
	if _, err := h.kubeClient.SecretGet(ctx, h.Config.Namespace, secretName, metav1.GetOptions{}); err != nil {
		registry := "https://index.docker.io/v1/"
		if model.ModelDocker.Registry != "" {
			registry = model.ModelDocker.Registry
		}
		dockerCfg := fmt.Sprintf(`{"auths":{"%s":{"username":"%s","password":"%s"}}}`, registry, model.ModelDocker.Username, model.ModelDocker.Password)
		wmSecret := apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: h.Config.Namespace,
				Labels: map[string]string{
					LABEL_SECRET: model.Name,
				},
			},
			Type: apiv1.SecretTypeDockerConfigJson,
			StringData: map[string]string{
				apiv1.DockerConfigJsonKey: dockerCfg,
			},
		}
		if _, err := h.kubeClient.SecretCreate(ctx, h.Config.Namespace, &wmSecret, metav1.CreateOptions{}); err != nil {
			return sdk.WrapError(err, "Cannot create secret %s", secretName)
		}
	}

	return nil
}
