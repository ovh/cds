package event_v2

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/cache"
)

const (
	eventQueue      = "events_v2"
	EventUIWS       = "event:ui"
	EventHatcheryWS = "event:run:job"
)

// Enqueue event into cache
func publish(ctx context.Context, store cache.Store, event sdk.EventV2) {
	if err := store.Enqueue(eventQueue, event); err != nil {
		log.Error(ctx, "EventV2.publish: %s", err)
		return
	}
}

// Dequeue runs in a goroutine and dequeue event from cache
func Dequeue(ctx context.Context, db *gorp.DbMap, store cache.Store, goroutines *sdk.GoRoutines) {
	for {
		if err := ctx.Err(); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "EventV2.DequeueEvent> Exiting: %v", err)
			return
		}

		e := sdk.EventV2{}
		if err := store.DequeueWithContext(ctx, eventQueue, 50*time.Millisecond, &e); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "EventV2.DequeueEvent> store.DequeueWithContext err: %v", err)
			continue
		}

		wg := sync.WaitGroup{}

		// Push to elasticsearch
		wg.Add(1)
		goroutines.Exec(ctx, "event.pushToElasticSearch", func(ctx context.Context) {
			defer wg.Done()
			if err := pushToElasticSearch(ctx, db, e); err != nil {
				log.Error(ctx, "EventV2.pushToElasticSearch: %v", err)
			}
		})

		// Create audit
		wg.Add(1)
		goroutines.Exec(ctx, "event.audit", func(ctx context.Context) {
			defer wg.Done()
			// TODO Audit
		})

		// Push to websockets channels
		wg.Add(1)
		goroutines.Exec(ctx, "event.websockets", func(ctx context.Context) {
			defer wg.Done()
			pushToWebsockets(ctx, store, e)
		})

		// Project notifications
		wg.Add(1)
		goroutines.Exec(ctx, "event.notifications", func(ctx context.Context) {
			defer wg.Done()
			if err := pushNotifications(ctx, db, e); err != nil {
				log.Error(ctx, "EventV2.pushNotifications: %v", err)
			}
		})

		wg.Wait()
	}
}

func pushToWebsockets(ctx context.Context, store cache.Store, e sdk.EventV2) {
	msg, err := json.Marshal(e)
	if err != nil {
		log.Error(ctx, "EventV2.pushToWebsockets: unable to marshal event: %v", err)
		return
	}
	if err := store.Publish(ctx, EventUIWS, string(msg)); err != nil {
		log.Error(ctx, "EventV2.pushToWebsockets: ui: %v", err)
	}

	if e.Type == sdk.EventRunJobEnqueued {
		if err := store.Publish(ctx, EventHatcheryWS, string(msg)); err != nil {
			log.Error(ctx, "EventV2.pushToWebsockets: hatchery: %v", err)
		}
	}
}

func pushNotifications(ctx context.Context, db *gorp.DbMap, e sdk.EventV2) error {
	if e.ProjectKey != "" {
		proj, err := project.Load(ctx, db, e.ProjectKey)
		if err != nil {
			return sdk.WrapError(err, "unable to load project %s", e.ProjectKey)
		}
		log.Debug(ctx, "Sending notification on project %s", proj.Key)
		// TODO send notification

	}
	return nil
}

func pushToElasticSearch(ctx context.Context, db *gorp.DbMap, e sdk.EventV2) error {
	esServices, err := services.LoadAllByType(ctx, db, sdk.TypeElasticsearch)
	if err != nil {
		return sdk.WrapError(err, "unable to load elasticsearch service")
	}

	if len(esServices) == 0 {
		return nil
	}

	e.Payload = nil
	log.Info(ctx, "sending event %q to %s services", e.Type, sdk.TypeElasticsearch)
	_, code, err := services.NewClient(db, esServices).DoJSONRequest(context.Background(), "POST", "/v2/events", e, nil)
	if code >= 400 || err != nil {
		return sdk.WrapError(err, "unable to send event %s to elasticsearch [%d]: %v", e.Type, code, err)
	}
	return nil
}
