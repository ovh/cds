package kubernetes

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

func (h *HatcheryKubernetes) killAwolWorkers() error {
	pods, err := h.k8sClient.CoreV1().Pods(h.Config.Namespace).List(metav1.ListOptions{LabelSelector: LABEL_WORKER})
	if err != nil {
		return err
	}

	var globalErr error
	for _, pod := range pods.Items {
		toDelete := false
		for _, container := range pod.Status.ContainerStatuses {
			if (container.State.Terminated != nil && (container.State.Terminated.Reason == "Completed" || container.State.Terminated.Reason == "Error")) ||
				(container.State.Waiting != nil && container.State.Waiting.Reason == "ErrImagePull") {
				toDelete = true
			}
		}
		if toDelete {
			// If its a worker "register", check registration before deleting it
			if strings.Contains(pod.Name, "register-") {
				var modelPath string
				for _, e := range pod.Spec.Containers[0].Env {
					if e.Name == "CDS_MODEL_PATH" {
						modelPath = e.Value
					}
				}

				if err := hatchery.CheckWorkerModelRegister(h, modelPath); err != nil {
					var spawnErr = sdk.SpawnErrorForm{
						Error: err.Error(),
					}
					tuple := strings.SplitN(modelPath, "/", 2)
					if err := h.CDSClient().WorkerModelSpawnError(tuple[0], tuple[1], spawnErr); err != nil {
						log.Error("killAndRemove> error on call client.WorkerModelSpawnError on worker model %d for register: %s", modelPath, err)
					}
				}

			}
			if err := h.k8sClient.CoreV1().Pods(pod.Namespace).Delete(pod.Name, nil); err != nil {
				globalErr = err
				log.Error("hatchery:kubernetes> killAwolWorkers> Cannot delete pod %s (%s)", pod.Name, err)
			}
		}
	}
	return globalErr
}
