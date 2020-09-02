package repositoriesmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//ReceiveEvents has to be launched as a goroutine.
func ReceiveEvents(ctx context.Context, DBFunc func() *gorp.DbMap, store cache.Store) {
	for {
		e := sdk.Event{}
		if err := store.DequeueWithContext(ctx, "events_repositoriesmanager", 250*time.Millisecond, &e); err != nil {
			log.Error(ctx, "repositoriesmanager.ReceiveEvents > store.DequeueWithContext err: %v", err)
			continue
		}
		if err := ctx.Err(); err != nil {
			log.Error(ctx, "Exiting repositoriesmanager.ReceiveEvents: %v", err)
			return
		}

		db := DBFunc()
		if db != nil {
			tx, err := db.Begin()
			if err != nil {
				log.Error(ctx, "ReceiveEvents> err opening tx: %v", err)
			}
			if err := processEvent(ctx, tx, e, store); err != nil {
				log.Error(ctx, "ReceiveEvents> err while processing error: %v", err)
				if err2 := RetryEvent(&e, err, store); err2 != nil {
					log.Error(ctx, "ReceiveEvents> err while processing error on retry: %v", err2)
				}
			}
			if err := tx.Commit(); err != nil {
				tx.Rollback() // nolint
				log.Error(ctx, "ReceiveEvents> err commit tx: %v", err)
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

func processEvent(ctx context.Context, db gorpmapper.SqlExecutorWithTx, event sdk.Event, store cache.Store) error {
	if event.EventType != fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}) {
		return nil
	}

	var eventWNR sdk.EventRunWorkflowNode

	if err := json.Unmarshal(event.Payload, &eventWNR); err != nil {
		return sdk.WrapError(err, "cannot read payload")
	}
	if eventWNR.RepositoryManagerName == "" {
		return nil
	}
	vcsServer, err := LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, db, event.ProjectKey, eventWNR.RepositoryManagerName)
	if err != nil {
		return sdk.WrapError(err, "AuthorizedClient (%s, %s)", event.ProjectKey, eventWNR.RepositoryManagerName)
	}

	c, err := AuthorizedClient(ctx, db, store, event.ProjectKey, vcsServer)
	if err != nil {
		return sdk.WrapError(err, "AuthorizedClient (%s, %s)", event.ProjectKey, eventWNR.RepositoryManagerName)
	}

	if err := c.SetStatus(ctx, event); err != nil {
		if err := RetryEvent(&event, err, store); err != nil {
			log.Error(ctx, "repositoriesmanager>processEvent> err while retry event: %v", err)
		}
		return sdk.WrapError(err, "event.EventType: %s", event.EventType)
	}

	return nil
}
