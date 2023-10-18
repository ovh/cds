package kubernetes

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/hatchery"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (h *HatcheryKubernetes) killAwolWorkers(ctx context.Context) error {
	pods, err := h.kubeClient.PodList(ctx, h.Config.Namespace, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s", LABEL_HATCHERY_NAME, h.Config.Name, LABEL_WORKER_NAME),
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	workers, err := h.CDSClient().WorkerList(ctx)
	if err != nil {
		return err
	}

	var globalErr error
	for _, pod := range pods.Items {
		annotations := pod.GetAnnotations()
		labels := pod.GetLabels()
		if labels == nil {
			continue
		}

		var toDelete bool
		for _, container := range pod.Status.ContainerStatuses {
			terminated := (container.State.Terminated != nil && (container.State.Terminated.Reason == "Completed" || container.State.Terminated.Reason == "Error"))
			errImagePull := (container.State.Waiting != nil && container.State.Waiting.Reason == "ErrImagePull")
			if terminated || errImagePull {
				toDelete = true
				log.Debug(ctx, "pod %s/%s is terminated or in error", pod.Namespace, pod.Name)
				break
			}
		}

		if !toDelete {
			var found bool
			for _, w := range workers {
				if workerName, ok := labels[LABEL_WORKER_NAME]; ok && workerName == w.Name {
					found = true
					break
				}
			}
			if !found && time.Since(pod.CreationTimestamp.Time) > 3*time.Minute {
				toDelete = true
				log.Debug(ctx, "pod %s/%s didn't match a registered worker and was started since %v", pod.Namespace, pod.Name, pod.CreationTimestamp.Time)
			}
		}

		if toDelete {
			// If no job identifiers, no services on pod
			jobIdentifiers := getJobIdentiers(labels)
			if jobIdentifiers != nil {
				// Browse container to send end log for each service
				servicesLogs := make([]cdslog.Message, 0)
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
					finalLog := cdslog.Message{
						Level: logrus.InfoLevel,
						Value: "End of Job",
						Signature: cdn.Signature{
							Service: &cdn.SignatureService{
								HatcheryID:      h.Service().ID,
								HatcheryName:    h.ServiceName(),
								RequirementID:   reqServiceID,
								RequirementName: subsStr[0][2],
								WorkerName:      pod.ObjectMeta.Name,
							},
							ProjectKey:   labels[hatchery.LabelServiceProjectKey],
							WorkflowName: labels[hatchery.LabelServiceWorkflowName],
							WorkflowID:   jobIdentifiers.WorkflowID,
							RunID:        jobIdentifiers.RunID,
							NodeRunName:  labels[hatchery.LabelServiceNodeRunName],
							JobName:      annotations[hatchery.LabelServiceJobName],
							JobID:        jobIdentifiers.JobID,
							NodeRunID:    jobIdentifiers.NodeRunID,
							Timestamp:    time.Now().UnixNano(),
						},
					}
					servicesLogs = append(servicesLogs, finalLog)
				}
				if len(servicesLogs) > 0 {
					h.Common.SendServiceLog(ctx, servicesLogs, sdk.StatusTerminated)
				}
			}

			// If its a worker "register", check registration before deleting it
			if strings.HasPrefix(pod.Name, "register-") {
				var modelPath string
				for _, e := range pod.Spec.Containers[0].Env {
					if e.Name == "CDS_MODEL_PATH" {
						modelPath = e.Value
					}
				}

				if err := hatchery.CheckWorkerModelRegister(ctx, h, modelPath); err != nil {
					var spawnErr = sdk.SpawnErrorForm{
						Error: err.Error(),
					}
					tuple := strings.SplitN(modelPath, "/", 2)
					if err := h.CDSClient().WorkerModelSpawnError(tuple[0], tuple[1], spawnErr); err != nil {
						log.Error(ctx, "killAndRemove> error on call client.WorkerModelSpawnError on worker model %s for register: %s", modelPath, err)
					}
				}
			}
			if err := h.kubeClient.PodDelete(ctx, pod.Namespace, pod.Name, metav1.DeleteOptions{}); err != nil {
				globalErr = err
				log.Error(ctx, "hatchery:kubernetes> killAwolWorkers> Cannot delete pod %s (%s)", pod.Name, err)
			}

			if err := h.deleteSecretByWorkerName(ctx, labels[LABEL_WORKER_NAME]); err != nil {
				log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "cannot delete secret for worker %s", labels[LABEL_WORKER_NAME]))
			}

			log.Debug(ctx, "pod %s/%s killed", pod.Namespace, pod.Name)
		}
	}
	return globalErr
}
