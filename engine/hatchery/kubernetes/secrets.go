package kubernetes

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (h *HatcheryKubernetes) deleteSecrets() error {
	pods, err := h.k8sClient.CoreV1().Pods(h.Config.Namespace).List(metav1.ListOptions{LabelSelector: LABEL_SECRET})
	if err != nil {
		return sdk.WrapError(err, "cannot get pods with secret")
	}
	secrets, errS := h.k8sClient.CoreV1().Secrets(h.Config.Namespace).List(metav1.ListOptions{LabelSelector: LABEL_SECRET})
	if errS != nil {
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
			if err := h.k8sClient.CoreV1().Secrets(h.Config.Namespace).Delete(secret.Name, nil); err != nil {
				log.Error("deleteSecrets> Cannot delete secret %s : %v", secret.Name, err)
			}
		}
	}

	return nil
}

func (h *HatcheryKubernetes) createSecret(secretName string, model sdk.Model) error {
	h.k8sClient.CoreV1().Secrets(h.Config.Namespace)
	_, err := h.k8sClient.CoreV1().Secrets(h.Config.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
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
		_, errCreate := h.k8sClient.CoreV1().Secrets(h.Config.Namespace).Create(&wmSecret)
		if errCreate != nil {
			return sdk.WrapError(errCreate, "Cannot create secret %s", secretName)
		}
	}

	return nil
}
