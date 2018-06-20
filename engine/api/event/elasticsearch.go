package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PushInElasticSearch pushes event to an elasticsearch
func PushInElasticSearch(c context.Context, db gorp.SqlExecutor, store cache.Store) {
	querier := services.Querier(db, store)
	pubSub := store.Subscribe("events_pubsub")
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("PushInElasticSearch> Exiting: %v", c.Err())
				return
			}
		case <-tick.C:
			msg, err := store.GetMessageFromSubscription(c, pubSub)
			if err != nil {
				log.Warning("PushInElasticSearch> Cannot get message %s: %s", msg, err)
				continue
			}

			esServices, errS := querier.FindByType(services.TypeElasticsearch)
			if errS != nil {
				log.Warning("PushInElasticSearch> Unable to get elasticsearch service")
				continue
			}

			if len(esServices) == 0 {
				continue
			}

			var e sdk.Event
			if err := json.Unmarshal([]byte(msg), &e); err != nil {
				log.Warning("PushInElasticSearch> Cannot unmarshal event %s: %s", msg, err)
				continue
			}

			switch e.EventType {
			case "sdk.EventPipelineBuild", "sdk.EventJob":
				continue
			}
			e.Payload = nil
			code, errD := services.DoJSONRequest(esServices, "POST", "/events", e, nil)
			if code >= 400 || errD != nil {
				log.Warning("PushInElasticSearch> Unable to send event to elasticsearch [%d]: %v", code, errD)
				continue
			}
		}
	}
}
