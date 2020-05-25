package repositoriesmanager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//ReceiveEvents has to be launched as a goroutine.
func ReceiveEvents(ctx context.Context, DBFunc func() *gorp.DbMap, store cache.Store) {
	for {
		e := sdk.Event{}
		if err := store.DequeueWithContext(ctx, "events_repositoriesmanager", &e); err != nil {
			log.Error(ctx, "repositoriesmanager.ReceiveEvents > store.DequeueWithContext err: %v", err)
			continue
		}
		if err := ctx.Err(); err != nil {
			log.Error(ctx, "Exiting repositoriesmanager.ReceiveEvents: %v", err)
			return
		}

		db := DBFunc()
		if db != nil {
			if err := processEvent(ctx, db, e, store); err != nil {
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

//RetryEvent retries the events
func RetryEvent(e *sdk.Event, err error, store cache.Store) error {
	e.Attempts++
	if e.Attempts > 2 {
		return sdk.WrapError(err, "ReceiveEvents> Aborting event processing")
	}
	return store.Enqueue("events_repositoriesmanager", e)
}

func processEvent(ctx context.Context, db *gorp.DbMap, event sdk.Event, store cache.Store) error {
	var c sdk.VCSAuthorizedClientService
	var errC error

	if event.EventType != fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}) {
		return nil
	}

	var eventWNR sdk.EventRunWorkflowNode

	if err := json.Unmarshal(event.Payload, &eventWNR); err != nil {
		return fmt.Errorf("cannot read payload: %v", err)
	}
	if eventWNR.RepositoryManagerName == "" {
		return nil
	}
	vcsServer, err := LoadForProject(db, event.ProjectKey, eventWNR.RepositoryManagerName)
	if err != nil {
		return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", event.ProjectKey, eventWNR.RepositoryManagerName, err)
	}

	c, err = AuthorizedClient(ctx, db, store, event.ProjectKey, vcsServer)
	if err != nil {
		return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", event.ProjectKey, eventWNR.RepositoryManagerName, errC)
	}

	if err := c.SetStatus(ctx, event); err != nil {
		if err2 := RetryEvent(&event, err, store); err2 != nil {
			log.Error(ctx, "repositoriesmanager>processEvent> err while retry event: %v", err2)
		}
		return fmt.Errorf("repositoriesmanager>processEvent> SetStatus > event.EventType:%s err:%s", event.EventType, err)
	}

	return nil
}
