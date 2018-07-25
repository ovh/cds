package kubernetes

import (
	"fmt"
	"strconv"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (h *HatcheryKubernetes) getServicesLogs() error {
	pods, err := h.k8sClient.CoreV1().Pods(h.Config.KubernetesNamespace).List(metav1.ListOptions{LabelSelector: LABEL_SERVICE_JOB_ID})
	if err != nil {
		return err
	}

	servicesLogs := make([]sdk.ServiceLog, 0, len(pods.Items))
	for _, pod := range pods.Items {
		podName := pod.GetName()
		labels := pod.GetLabels()
		if labels == nil {
			log.Error("getServicesLogs> labels is nil")
			continue
		}

		serviceJobID, errPj := strconv.ParseInt(labels[LABEL_SERVICE_JOB_ID], 10, 64)
		if errPj != nil {
			log.Error("getServicesLogs> cannot parse service job id for pod service %s, err : %v", podName, errPj)
			continue
		}

		var sinceSeconds int64 = 10
		for _, container := range pod.Spec.Containers {
			subsStr := containerServiceNameRegexp.FindAllStringSubmatch(container.Name, -1)
			if len(subsStr) < 1 {
				continue
			}
			if len(subsStr[0]) < 3 {
				log.Error("getServiceLogs> cannot find service id in the container name (%s) : %v", container.Name, subsStr)
				continue
			}
			logsOpts := apiv1.PodLogOptions{SinceSeconds: &sinceSeconds, Container: container.Name, Timestamps: true}
			logs, errLogs := h.k8sClient.CoreV1().Pods(h.Config.KubernetesNamespace).GetLogs(podName, &logsOpts).DoRaw()
			if errLogs != nil {
				log.Error("getServicesLogs> cannot get logs for container %s in pod %s, err : %v", container.Name, podName, errLogs)
				continue
			}

			// No check on error thanks to the regexp
			reqServiceID, _ := strconv.ParseInt(subsStr[0][1], 10, 64)

			servicesLogs = append(servicesLogs, sdk.ServiceLog{
				WorkflowNodeJobRunID:   serviceJobID,
				ServiceRequirementID:   reqServiceID,
				ServiceRequirementName: subsStr[0][2],
				Val: string(logs),
			})
		}
	}

	if len(servicesLogs) > 0 {
		// Do call api
		if err := h.Client.QueueServiceLogs(servicesLogs); err != nil {
			return fmt.Errorf("Hatchery> Swarm> Cannot send service logs : %v", err)
		}
	}

	return nil
}
