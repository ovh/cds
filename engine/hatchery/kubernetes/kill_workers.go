package kubernetes

import (
	"context"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

func (h *HatcheryKubernetes) killAwolWorkers(ctx context.Context) error {
	pods, err := h.k8sClient.CoreV1().Pods(h.Config.Namespace).List(metav1.ListOptions{LabelSelector: LABEL_WORKER})
	if err != nil {
		return err
	}
	var globalErr error
	for _, pod := range pods.Items {

		labels := pod.GetLabels()
		toDelete := false
		for _, container := range pod.Status.ContainerStatuses {

			if (container.State.Terminated != nil && (container.State.Terminated.Reason == "Completed" || container.State.Terminated.Reason == "Error")) ||
				(container.State.Waiting != nil && container.State.Waiting.Reason == "ErrImagePull") {
				toDelete = true
			}
		}

		// If no job identifiers, no services on pod
		jobIdentifiers := h.getJobIdentiers(labels)
		if jobIdentifiers != nil {
			// Browse container to send end log for each service
			servicesLogs := make([]sdk.ServiceLog, 0)
			for _, container := range pod.Spec.Containers {
				subsStr := containerServiceNameRegexp.FindAllStringSubmatch(container.Name, -1)
				if len(subsStr) < 1 {
					continue
				}
				if len(subsStr[0]) < 3 {
					log.Error(ctx, "getServiceLogs> cannot find service id in the container name (%s) : %v", container.Name, subsStr)
					continue
				}
				reqServiceID, _ := strconv.ParseInt(subsStr[0][1], 10, 64)
				finalLog := sdk.ServiceLog{
					WorkflowNodeJobRunID:   jobIdentifiers.JobID,
					ServiceRequirementID:   reqServiceID,
					ServiceRequirementName: subsStr[0][2],
					Val:                    "End of Job",
					WorkerName:             pod.ObjectMeta.Name,
					JobName:                labels[hatchery.LabelServiceJobName],
					NodeRunName:            labels[hatchery.LabelServiceNodeRunName],
					WorkflowName:           labels[hatchery.LabelServiceWorkflowName],
					ProjectKey:             labels[hatchery.LabelServiceProjectKey],
					RunID:                  jobIdentifiers.RunID,
					WorkflowID:             jobIdentifiers.WorkflowID,
					WorkflowNodeRunID:      jobIdentifiers.NodeRunID,
				}
				servicesLogs = append(servicesLogs, finalLog)
			}
			if len(servicesLogs) > 0 {
				h.Common.SendServiceLog(ctx, servicesLogs, sdk.StatusSuccess)
			}
		}

		if toDelete {
			// If its a worker "register", check registration before deleting it
			if strings.HasPrefix(pod.Name, "register-") {
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
						log.Error(ctx, "killAndRemove> error on call client.WorkerModelSpawnError on worker model %s for register: %s", modelPath, err)
					}
				}

			}
			if err := h.k8sClient.CoreV1().Pods(pod.Namespace).Delete(pod.Name, nil); err != nil {
				globalErr = err
				log.Error(ctx, "hatchery:kubernetes> killAwolWorkers> Cannot delete pod %s (%s)", pod.Name, err)
			}
		}
	}
	return globalErr
}
