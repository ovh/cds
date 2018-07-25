package kubernetes

import (
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

func (h *HatcheryKubernetes) killAwolWorkers() error {
	pods, err := h.k8sClient.CoreV1().Pods(h.Config.KubernetesNamespace).List(metav1.ListOptions{LabelSelector: LABEL_WORKER})
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
				var modelIDS string
				for _, e := range pod.Spec.Containers[0].Env {
					if e.Name == "CDS_MODEL" {
						modelIDS = e.Value
					}
				}
				modelID, err := strconv.ParseInt(modelIDS, 10, 64)
				if err != nil {
					log.Error("killAndRemove> unable to get model from registering container %s", pod.Name)
				} else {
					hatchery.CheckWorkerModelRegister(h, modelID)
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
