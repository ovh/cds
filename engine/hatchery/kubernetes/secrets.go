package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/rockbears/log"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/slug"
)

// Delete worker model registry and worker config secrets that are not used by any pods.
// This is used as a "gc", in the nominal case, the deletion of secrets is done when removing workers with killAwolWorkers
func (h *HatcheryKubernetes) deleteSecrets(ctx context.Context) error {
	pods, err := h.kubeClient.PodList(ctx, h.Config.Namespace, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s", LABEL_HATCHERY_NAME, h.Config.Name, LABEL_WORKER_NAME),
	})
	if err != nil {
		return sdk.WrapError(err, "cannot get pods with secret")
	}

	secrets, err := h.kubeClient.SecretList(ctx, h.Config.Namespace, metav1.ListOptions{LabelSelector: LABEL_HATCHERY_NAME})
	if err != nil {
		return sdk.WrapError(err, "cannot get secrets")
	}

	for _, secret := range secrets.Items {
		// created last 10min, too young to delete it
		if time.Since(secret.GetCreationTimestamp().Time) < 10*time.Minute {
			continue
		}
		log.Info(ctx, "deleting the secret %v with creationTimestamp %v", secret.Name, secret.GetCreationTimestamp().Time)
		secretLabels := secret.GetLabels()
		if secretLabels == nil {
			continue
		}
		var found bool
		for _, pod := range pods.Items {
			podLabels := pod.GetLabels()
			if podLabels == nil {
				continue
			}
			if w, ok := secretLabels[LABEL_WORKER_NAME]; ok && podLabels[LABEL_WORKER_NAME] == w {
				found = true
				break
			}
		}
		if !found {
			log.Debug(ctx, "delete secret %q", secret.Name)
			if err := h.kubeClient.SecretDelete(ctx, h.Config.Namespace, secret.Name, metav1.DeleteOptions{}); err != nil {
				log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "cannot delete secret %s", secret.Name))
			}
		}
	}

	return nil
}

// deleteSecretByName deletes a secret. If the secret does not exit, return nil
func (h *HatcheryKubernetes) deleteSecretByWorkerName(ctx context.Context, workerName string) error {
	secrets, err := h.kubeClient.SecretList(ctx, h.Config.Namespace, metav1.ListOptions{LabelSelector: LABEL_HATCHERY_NAME})
	if err != nil {
		return sdk.WrapError(err, "cannot get secrets")
	}

	for _, secret := range secrets.Items {
		secretLabels := secret.GetLabels()
		if secretLabels == nil {
			continue
		}
		if wname, ok := secretLabels[LABEL_WORKER_NAME]; ok && workerName == wname {
			if err := h.kubeClient.SecretDelete(ctx, h.Config.Namespace, secret.Name, metav1.DeleteOptions{}); err != nil {
				return sdk.WrapError(err, "cannot delete secret %s from worker %s", secret.Name, secretLabels[LABEL_WORKER_NAME])
			}
			// no break, we can have two secrets used by a worker (registry and config secrets)
		}
	}

	return nil
}

func (h *HatcheryKubernetes) createRegistrySecret(ctx context.Context, model sdk.WorkerStarterWorkerModel, workerName string) (string, error) {
	secretName := slug.Convert("cds-worker-registry-" + workerName)
	registry := "https://index.docker.io/v1/"
	if model.ModelV1 != nil && model.ModelV1.ModelDocker.Registry != "" {
		registry = model.ModelV1.ModelDocker.Registry
	} else {
		ref, err := reference.ParseNormalizedNamed(model.GetDockerImage())
		if err != nil {
			return "", sdk.WithStack(err)
		}
		domain := reference.Domain(ref)
		registry = domain
	}
	dockerCfg := fmt.Sprintf(`{"auths":{"%s":{"username":"%s","password":"%s"}}}`, registry, model.GetDockerUsername(), model.GetDockerPassword())

	if _, err := h.kubeClient.SecretCreate(ctx, h.Config.Namespace, &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: h.Config.Namespace,
			Labels: map[string]string{
				LABEL_HATCHERY_NAME: h.Configuration().Name,
				LABEL_WORKER_NAME:   workerName,
			},
		},
		Type: apiv1.SecretTypeDockerConfigJson,
		StringData: map[string]string{
			apiv1.DockerConfigJsonKey: dockerCfg,
		},
	}, metav1.CreateOptions{}); err != nil {
		return "", sdk.WrapError(err, "cannot create secret %s", secretName)
	}

	return secretName, nil
}

func (h *HatcheryKubernetes) createConfigSecret(ctx context.Context, workerConfig workerruntime.WorkerConfig) (string, error) {
	secretName := slug.Convert("cds-worker-config-" + workerConfig.Name)

	if _, err := h.kubeClient.SecretCreate(ctx, h.Config.Namespace, &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: h.Config.Namespace,
			Labels: map[string]string{
				LABEL_HATCHERY_NAME: h.Configuration().Name,
				LABEL_WORKER_NAME:   workerConfig.Name,
			},
		},
		Type: apiv1.SecretTypeOpaque,
		StringData: map[string]string{
			"CDS_CONFIG": workerConfig.EncodeBase64(),
		},
	}, metav1.CreateOptions{}); err != nil {
		return "", sdk.WrapError(err, "cannot create secret %s", secretName)
	}

	return secretName, nil
}
