package repositoriesmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// ReceiveEvents has to be launched as a goroutine.
func ReceiveEvents(ctx context.Context, DBFunc func() *gorp.DbMap, store cache.Store, cdsUIURL string) {
	for {
		if err := ctx.Err(); err != nil {
			log.Error(ctx, "repositoriesmanager.ReceiveEvents> exiting: %v", err)
			return
		}

		e := sdk.Event{}
		if err := store.DequeueWithContext(ctx, "events_repositoriesmanager", 250*time.Millisecond, &e); err != nil {
			log.Error(ctx, "repositoriesmanager.ReceiveEvents > store.DequeueWithContext err: %v", err)
			continue
		}

		db := DBFunc()
		if db != nil {
			if err := processEvent(ctx, db, e, store, cdsUIURL); err != nil {
				log.Error(ctx, "ReceiveEvents> err while processing error: %v", err)
				if err2 := RetryEvent(&e, err, store); err2 != nil {
					log.Error(ctx, "ReceiveEvents> err while processing error on retry: %v", err2)
				}
			}
			continue
		}
		if err := RetryEvent(&e, nil, store); err != nil {
			log.Error(ctx, "ReceiveEvents> err while retry event: %v", err)
		}
	}
}

// RetryEvent retries the events
func RetryEvent(e *sdk.Event, err error, store cache.Store) error {
	e.Attempts++
	if e.Attempts > 2 {
		return sdk.WrapError(err, "ReceiveEvents> Aborting event processing")
	}
	return store.Enqueue("events_repositoriesmanager", e)
}

func processEvent(ctx context.Context, db *gorp.DbMap, event sdk.Event, store cache.Store, cdsUIURL string) error {
	if event.EventType != fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}) {
		return nil
	}

	var eventWNR sdk.EventRunWorkflowNode
	if err := sdk.JSONUnmarshal(event.Payload, &eventWNR); err != nil {
		return sdk.WrapError(err, "cannot read payload")
	}
	if eventWNR.RepositoryManagerName == "" {
		return nil
	}

	c, err := AuthorizedClient(ctx, db, store, event.ProjectKey, eventWNR.RepositoryManagerName)
	if err != nil {
		return sdk.WrapError(err, "AuthorizedClient (%s, %s)", event.ProjectKey, eventWNR.RepositoryManagerName)
	}

	buildStatus := sdk.VCSBuildStatus{
		Description:        eventWNR.NodeName + ":" + eventWNR.Status,
		URLCDS:             fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", cdsUIURL, event.ProjectKey, event.WorkflowName, eventWNR.Number),
		Context:            fmt.Sprintf("%s-%s-%s", event.ProjectKey, event.WorkflowName, eventWNR.NodeName),
		Status:             eventWNR.Status,
		RepositoryFullname: eventWNR.RepositoryFullName,
		GitHash:            eventWNR.Hash,
		GerritChange:       eventWNR.GerritChange,
	}

	if err := c.SetStatus(ctx, buildStatus); err != nil {
		if err := RetryEvent(&event, err, store); err != nil {
			log.Error(ctx, "repositoriesmanager>processEvent> err while retry event: %v", err)
		}
		return sdk.WrapError(err, "event.EventType: %s", event.EventType)
	}

	return nil
}
