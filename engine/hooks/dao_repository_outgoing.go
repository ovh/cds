package hooks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

/**
 * Receive outgoing event
 *    1. Save event: -> hooks:outgoing:repository:bitbucketserver-my_bibucket_server-my/repo
 *    2. Insert event_key in inprogress list: -> hooks:queue:repository:outgoing:inprogress
 *    3. Enqueue event_key for scheduling: ->  hooks:queue:repository:outgoing
 *
 */

func (d *dao) GetOutgoingMemberKey(projKey, vcsName, repoName, workflowName string) string {
	return fmt.Sprintf("%s-%s-%s-%s", projKey, vcsName, repoName, workflowName)
}

func (d *dao) SaveWorkflowRunOutgoingEvent(_ context.Context, e *sdk.HookWorkflowRunOutgoingEvent) error {
	e.LastUpdate = time.Now().UnixMilli()
	k := strings.ToLower(cache.Key(workflowRunOutgoingEventRootKey, d.GetOutgoingMemberKey(e.Event.WorkflowProject, e.Event.WorkflowVCSServer, e.Event.WorkflowRepository, e.Event.WorkflowName)))
	return d.store.SetAdd(k, e.UUID, e)
}

func (d *dao) EnqueueWorkflowRunOutgoingEvent(ctx context.Context, e *sdk.HookWorkflowRunOutgoingEvent) error {
	// Use to identify event in progress:
	k := strings.ToLower(cache.Key(workflowRunOutgoingEventRootKey, d.GetOutgoingMemberKey(e.Event.WorkflowProject, e.Event.WorkflowVCSServer, e.Event.WorkflowRepository, e.Event.WorkflowName), e.UUID))
	log.Debug(ctx, "enqueue outgoing event: %s", k)

	if err := d.store.SetRemove(workflowRunOutgoingEventInProgressKey, e.UUID, k); err != nil {
		return err
	}
	if err := d.store.SetAdd(workflowRunOutgoingEventInProgressKey, e.UUID, k); err != nil {
		return err
	}

	if err := d.store.Enqueue(workflowRunOutgoingEventQueue, k); err != nil {
		return err
	}

	d.enqueuedWorkflowRunOutgoingEventIncr()
	return nil
}

func (d *dao) WorkflowRunOutgoingEventQueueLen() (int, error) {
	return d.store.QueueLen(workflowRunOutgoingEventQueue)
}

func (d *dao) LockWorkflowRunOutgoingEvent(hookEventUUID string) (bool, error) {
	lockKey := d.getWorkflowRunOutgoingEventLockKey(hookEventUUID)
	return d.store.Lock(lockKey, 30*time.Second, 200, 60)
}

func (d *dao) UnlockWorkflowRunOutgoingEvent(hookEventUUID string) error {
	lockKey := d.getWorkflowRunOutgoingEventLockKey(hookEventUUID)
	return d.store.Unlock(lockKey)
}

func (d *dao) getWorkflowRunOutgoingEventLockKey(hookEventUUID string) string {
	return strings.ToLower(cache.Key(workflowRunOutgoingEventLockRootKey, hookEventUUID))
}

func (d *dao) RemoveWorkflowRunOutgoingEventFromInProgressList(ctx context.Context, e sdk.HookWorkflowRunOutgoingEvent) error {
	return d.store.SetRemove(workflowRunOutgoingEventInProgressKey, e.UUID, e)
}

func (d *dao) ListInProgressWorkflowRunOutgoingEvent(ctx context.Context) ([]string, error) {
	nbOutgoingEventInProgress, err := d.store.SetCard(workflowRunOutgoingEventInProgressKey)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to setCard %v", workflowRunOutgoingEventInProgressKey)
	}
	inProgressEvents := make([]*string, 0, nbOutgoingEventInProgress)
	for i := 0; i < nbOutgoingEventInProgress; i++ {
		content := ""
		inProgressEvents = append(inProgressEvents, &content)
	}
	if err := d.store.SetScan(ctx, workflowRunOutgoingEventInProgressKey, sdk.InterfaceSlice(inProgressEvents)...); err != nil {
		return nil, sdk.WrapError(err, "Unable to scan %s", workflowRunOutgoingEventInProgressKey)
	}

	eventKeys := make([]string, 0, len(inProgressEvents))
	for _, k := range inProgressEvents {
		eventKeys = append(eventKeys, *k)
	}

	return eventKeys, nil
}

func (d *dao) ListWorkflowRunOutgoingEvents(ctx context.Context, proj, vcsServer, repository, workflow string) ([]sdk.HookWorkflowRunOutgoingEvent, error) {
	k := strings.ToLower(cache.Key(workflowRunOutgoingEventRootKey, d.GetOutgoingMemberKey(proj, vcsServer, repository, workflow)))
	nbEvents, err := d.store.SetCard(k)
	if err != nil {
		return nil, err
	}
	events := make([]*sdk.HookWorkflowRunOutgoingEvent, nbEvents)
	for i := 0; i < nbEvents; i++ {
		events[i] = &sdk.HookWorkflowRunOutgoingEvent{}
	}
	if err := d.store.SetScan(ctx, k, sdk.InterfaceSlice(events)...); err != nil {
		return nil, err
	}
	finalEvents := make([]sdk.HookWorkflowRunOutgoingEvent, 0, len(events))
	for _, e := range events {
		finalEvents = append(finalEvents, *e)
	}
	return finalEvents, nil
}

func (d *dao) OutgoingEventCallbackBalance() (int64, int64) {
	return d.enqueuedWorkflowRunOutgoingEvents, d.dequeuedWorkflowRunOutgoingEvents
}
