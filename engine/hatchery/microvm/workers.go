package microvm

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/log"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/ovh/cds/sdk"
)

func (h *HatcheryMicroVM) labelSelector() string {
	return fmt.Sprintf("%s=%s,%s", LABEL_HATCHERY_NAME, h.Config.Name, LABEL_WORKER_NAME)
}

func (h *HatcheryMicroVM) listMicroVMs(ctx context.Context) ([]unstructured.Unstructured, error) {
	list, err := h.dynClient.Resource(microVMResource).Namespace(h.Config.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: h.labelSelector(),
	})
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list MicroVMs on namespace %s", h.Config.Namespace)
	}
	return list.Items, nil
}

func phase(mvm *unstructured.Unstructured) string {
	p, _, _ := unstructured.NestedString(mvm.Object, "status", "phase")
	return p
}

func isEnded(mvm *unstructured.Unstructured) bool {
	p := phase(mvm)
	return p == phaseSucceeded || p == phaseFailed
}

// isQuotaPending is true while the namespace quota refuses the MicroVM: it
// has not started yet, it is not a failure.
func isQuotaPending(mvm *unstructured.Unstructured) bool {
	if phase(mvm) != phasePending {
		return false
	}
	conditions, _, _ := unstructured.NestedSlice(mvm.Object, "status", "conditions")
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if cond["type"] == conditionQuotaExceeded && cond["status"] == string(metav1.ConditionTrue) {
			return true
		}
	}
	return false
}

// startTime is the boot time reported by the platform, zero before the boot
func startTime(mvm *unstructured.Unstructured) time.Time {
	s, _, _ := unstructured.NestedString(mvm.Object, "status", "startTime")
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// WorkersStarted returns the MicroVMs that are not ended, including the ones
// waiting for the quota, so that they count in the hatchery capacity.
func (h *HatcheryMicroVM) WorkersStarted(ctx context.Context) ([]string, error) {
	mvms, err := h.listMicroVMs(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(mvms))
	for i := range mvms {
		if isEnded(&mvms[i]) {
			continue
		}
		names = append(names, mvms[i].GetName())
	}
	return names, nil
}

// awolDecision tells whether a MicroVM must be deleted, and why
func (h *HatcheryMicroVM) awolDecision(mvm *unstructured.Unstructured, registered bool, now time.Time) (bool, string) {
	if isEnded(mvm) {
		reason, _, _ := unstructured.NestedString(mvm.Object, "status", "reason")
		message, _, _ := unstructured.NestedString(mvm.Object, "status", "message")
		return true, fmt.Sprintf("MicroVM ended: phase %s, reason %q, message %q", phase(mvm), reason, message)
	}
	if registered {
		return false, ""
	}
	created := mvm.GetCreationTimestamp().Time
	if isQuotaPending(mvm) {
		timeout := time.Duration(h.Config.QuotaPendingTimeout) * time.Second
		if now.Sub(created) > timeout {
			return true, fmt.Sprintf("refused by the namespace quota for more than %v", timeout)
		}
		return false, ""
	}
	since := created
	if t := startTime(mvm); !t.IsZero() {
		since = t
	}
	timeout := time.Duration(h.Config.RegistrationTimeout) * time.Second
	if now.Sub(since) > timeout {
		return true, fmt.Sprintf("no registered worker %v after %v", timeout, since)
	}
	return false, ""
}

func (h *HatcheryMicroVM) killAwolWorkers(ctx context.Context) error {
	mvms, err := h.listMicroVMs(ctx)
	if err != nil {
		return err
	}

	ctxList, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	workers, err := h.WorkerList(ctxList)
	if err != nil {
		return err
	}
	registered := make(map[string]struct{}, len(workers))
	for _, w := range workers {
		registered[w.GetName()] = struct{}{}
	}

	var globalErr error
	now := time.Now()
	for i := range mvms {
		mvm := &mvms[i]
		_, isRegistered := registered[mvm.GetLabels()[LABEL_WORKER_NAME]]
		toDelete, reason := h.awolDecision(mvm, isRegistered, now)
		if !toDelete {
			if isQuotaPending(mvm) {
				log.Info(ctx, "MicroVM %s is waiting for the namespace quota since %v", mvm.GetName(), mvm.GetCreationTimestamp().Time)
			}
			continue
		}
		if phase(mvm) == phaseSucceeded {
			log.Info(ctx, "deleting MicroVM %s: %s", mvm.GetName(), reason)
		} else {
			log.Warn(ctx, "deleting MicroVM %s: %s", mvm.GetName(), reason)
		}
		err := h.dynClient.Resource(microVMResource).Namespace(h.Config.Namespace).Delete(ctx, mvm.GetName(), metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			globalErr = sdk.WrapError(err, "cannot delete MicroVM %s", mvm.GetName())
			log.Error(ctx, "hatchery:microvm> killAwolWorkers> %v", globalErr)
		}
	}
	return globalErr
}

// deleteOrphanSecrets deletes the worker secrets without a MicroVM. The owner
// reference does it in the nominal case; this covers a failed spawn.
func (h *HatcheryMicroVM) deleteOrphanSecrets(ctx context.Context) error {
	mvms, err := h.listMicroVMs(ctx)
	if err != nil {
		return err
	}
	workers := make(map[string]struct{}, len(mvms))
	for i := range mvms {
		workers[mvms[i].GetLabels()[LABEL_WORKER_NAME]] = struct{}{}
	}

	secrets, err := h.kubeClient.CoreV1().Secrets(h.Config.Namespace).List(ctx, metav1.ListOptions{LabelSelector: h.labelSelector()})
	if err != nil {
		return sdk.WrapError(err, "cannot list secrets")
	}
	for _, secret := range secrets.Items {
		// A secret is created just before its MicroVM
		if time.Since(secret.CreationTimestamp.Time) < 10*time.Minute {
			continue
		}
		if _, has := workers[secret.Labels[LABEL_WORKER_NAME]]; has {
			continue
		}
		log.Info(ctx, "deleting orphan secret %s", secret.Name)
		if err := h.kubeClient.CoreV1().Secrets(h.Config.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
			log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "cannot delete secret %s", secret.Name))
		}
	}
	return nil
}
