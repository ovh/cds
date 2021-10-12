package kubernetes

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/hatchery"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (h *HatcheryKubernetes) getServicesLogs(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiWorkers, err := h.CDSClient().WorkerList(ctx)
	if err != nil {
		return sdk.WrapError(err, "cannot get worker list from CDS api")
	}
	apiWorkerNames := make(map[string]struct{}, len(apiWorkers))
	for i := range apiWorkers {
		apiWorkerNames[apiWorkers[i].Name] = struct{}{}
	}

	pods, err := h.kubeClient.PodList(ctx, h.Config.Namespace, metav1.ListOptions{LabelSelector: hatchery.LabelServiceJobID})
	if err != nil {
		return err
	}

	servicesLogs := make([]cdslog.Message, 0, len(pods.Items))
	for _, pod := range pods.Items {
		podName := pod.GetName()
		labels := pod.GetLabels()
		if labels == nil {
			log.Error(ctx, "getServicesLogs> labels is nil")
			continue
		}
		annotations := pod.GetAnnotations()
		if annotations == nil {
			log.Error(ctx, "annotations is nil")
		}

		// If no job identifier, no service on the pod
		jobIdentifiers := getJobIdentiers(labels)
		if jobIdentifiers == nil {
			continue
		}

		workerName := pod.ObjectMeta.Name
		// Check if there is a known worker in CDS api results for given worker name
		// If not we skip sending logs as the worker is not ready.
		// This will avoid problems validating log signature by the CDN service.
		if _, ok := apiWorkerNames[workerName]; !ok {
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
			logs, err := h.kubeClient.PodGetRawLogs(ctx, h.Config.Namespace, podName, &logsOpts)
			if err != nil {
				log.Error(ctx, "getServicesLogs> cannot get logs for container %s in pod %s, err : %v", container.Name, podName, err)
				continue
			}
			// No check on error thanks to the regexp
			reqServiceID, _ := strconv.ParseInt(subsStr[0][1], 10, 64)

			commonMessage := cdslog.Message{
				Level: logrus.InfoLevel,
				Signature: cdn.Signature{
					Service: &cdn.SignatureService{
						HatcheryID:      h.Service().ID,
						HatcheryName:    h.ServiceName(),
						RequirementID:   reqServiceID,
						RequirementName: subsStr[0][2],
						WorkerName:      workerName,
					},
					ProjectKey:   labels[hatchery.LabelServiceProjectKey],
					WorkflowName: labels[hatchery.LabelServiceWorkflowName],
					WorkflowID:   jobIdentifiers.WorkflowID,
					RunID:        jobIdentifiers.RunID,
					NodeRunName:  labels[hatchery.LabelServiceNodeRunName],
					JobName:      annotations[hatchery.LabelServiceJobName],
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

func getJobIdentiers(labels map[string]string) *hatchery.JobIdentifiers {
	serviceJobID, err := strconv.ParseInt(labels[hatchery.LabelServiceJobID], 10, 64)
	if err != nil {
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
