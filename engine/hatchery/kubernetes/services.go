package kubernetes

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

func (h *HatcheryKubernetes) getServicesLogs(ctx context.Context) error {
	pods, err := h.k8sClient.CoreV1().Pods(h.Config.Namespace).List(metav1.ListOptions{LabelSelector: hatchery.LabelServiceJobID})
	if err != nil {
		return err
	}

	servicesLogs := make([]log.Message, 0, len(pods.Items))
	for _, pod := range pods.Items {
		podName := pod.GetName()
		labels := pod.GetLabels()
		if labels == nil {
			log.Error(ctx, "getServicesLogs> labels is nil")
			continue
		}

		// If no job identifier, no service on the pod
		jobIdentifiers := h.getJobIdentiers(labels)
		if jobIdentifiers == nil {
			continue
		}

		var sinceSeconds int64 = 10
		for _, container := range pod.Spec.Containers {
			subsStr := containerServiceNameRegexp.FindAllStringSubmatch(container.Name, -1)
			if len(subsStr) < 1 {
				continue
			}
			if len(subsStr[0]) < 3 {
				log.Error(ctx, "getServiceLogs> cannot find service id in the container name (%s) : %v", container.Name, subsStr)
				continue
			}
			logsOpts := apiv1.PodLogOptions{SinceSeconds: &sinceSeconds, Container: container.Name, Timestamps: true}
			logs, errLogs := h.k8sClient.CoreV1().Pods(h.Config.Namespace).GetLogs(podName, &logsOpts).DoRaw()
			if errLogs != nil {
				log.Error(ctx, "getServicesLogs> cannot get logs for container %s in pod %s, err : %v", container.Name, podName, errLogs)
				continue
			}
			// No check on error thanks to the regexp
			reqServiceID, _ := strconv.ParseInt(subsStr[0][1], 10, 64)

			commonMessage := log.Message{
				Level: logrus.InfoLevel,
				Signature: log.Signature{
					Service: &log.SignatureService{
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
					JobName:      labels[hatchery.LabelServiceJobName],
					JobID:        jobIdentifiers.JobID,
					NodeRunID:    jobIdentifiers.NodeRunID,
				},
			}

			logsSplitted := strings.Split(string(logs), "\n")
			for i := range logsSplitted {
				if i == len(logsSplitted)-1 && logsSplitted[i] == "" {
					break
				}
				msg := commonMessage
				msg.Signature.Timestamp = time.Now().UnixNano()
				msg.Value = logsSplitted[i]
				servicesLogs = append(servicesLogs, msg)
			}
		}
	}

	if len(servicesLogs) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		h.Common.SendServiceLog(ctx, servicesLogs, sdk.StatusNotTerminated)
	}
	return nil
}

func (h *HatcheryKubernetes) getJobIdentiers(labels map[string]string) *hatchery.JobIdentifiers {
	serviceJobID, errPj := strconv.ParseInt(labels[hatchery.LabelServiceJobID], 10, 64)
	if errPj != nil {
		return nil
	}

	runID, err := strconv.ParseInt(labels[hatchery.LabelServiceRunID], 10, 64)
	if err != nil {
		return nil
	}

	workflowID, err := strconv.ParseInt(labels[hatchery.LabelServiceWorkflowID], 10, 64)
	if err != nil {
		return nil
	}

	nodeRunID, err := strconv.ParseInt(labels[hatchery.LabelServiceNodeRunID], 10, 64)
	if err != nil {
		return nil
	}
	return &hatchery.JobIdentifiers{
		WorkflowID: workflowID,
		RunID:      runID,
		NodeRunID:  nodeRunID,
		JobID:      serviceJobID,
	}
}
