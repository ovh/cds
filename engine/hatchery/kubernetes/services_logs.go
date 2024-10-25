package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rockbears/log"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
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

	pods, err := h.kubeClient.PodList(ctx, h.Config.Namespace, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s", LABEL_HATCHERY_NAME, h.Config.Name, hatchery.LabelServiceJobID),
	})
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

		labels[hatchery.LabelServiceJobName] = annotations[hatchery.LabelServiceJobName]

		workerName := pod.ObjectMeta.Name

		var sinceSeconds int64 = 10
		for _, container := range pod.Spec.Containers {
			_, has := labels[hatchery.LabelServiceID]
			serviceVersion, hasv2 := labels[hatchery.LabelServiceVersion]
			if !has && !hasv2 {
				continue // not a service
			}

			// check only for worker model v1
			if !hasv2 && serviceVersion != hatchery.ValueLabelServiceVersion2 {
				workerName := labels[hatchery.LabelServiceWorker]
				// Check if there is a known worker in CDS api results for given worker name
				// If not we skip sending logs as the worker is not ready.
				// This will avoid problems validating log signature by the CDN service.
				if _, ok := apiWorkerNames[workerName]; !ok {
					continue
				}
			}

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

			labels[hatchery.LabelServiceID] = subsStr[0][1]
			labels[hatchery.LabelServiceReqName] = subsStr[0][2]
			labels[hatchery.LabelServiceWorker] = workerName

			// If no job identifier, no service on the pod
			jobIdentifiers := hatchery.GetServiceIdentifiersFromLabels(labels)
			if jobIdentifiers == nil {
				continue
			}

			commonMessage := hatchery.PrepareCommonLogMessage(h.ServiceName(), h.Service().ID, *jobIdentifiers, labels)

			logsSplitted := strings.Split(string(logs), "\n")
			for i := range logsSplitted {
				if i == len(logsSplitted)-1 && logsSplitted[i] == "" {
					break
				}
				msg := commonMessage
				msg.Signature.Timestamp = time.Now().UnixNano()
				msg.Value = sdk.RemoveNotPrintableChar(logsSplitted[i])
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
