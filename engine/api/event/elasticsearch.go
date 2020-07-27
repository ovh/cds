package event

import (
	"context"
	"encoding/json"

	"github.com/go-gorp/gorp"
	"gopkg.in/olivere/elastic.v6"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PushInElasticSearch pushes event to an elasticsearch
func PushInElasticSearch(ctx context.Context, db gorp.SqlExecutor, store cache.Store) {
	eventChan := make(chan sdk.Event, 10)
	Subscribe(eventChan)

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "PushInElasticSearch> Exiting: %v", ctx.Err())
				return
			}
		case e := <-eventChan:
			esServices, errS := services.LoadAllByType(ctx, db, sdk.TypeElasticsearch)
			if errS != nil {
				log.Error(ctx, "PushInElasticSearch> Unable to get elasticsearch service: %v", errS)
				continue
			}

			if len(esServices) == 0 {
				continue
			}

			switch e.EventType {
			case "sdk.EventEngine":
				continue
			}
			e.Payload = nil
			_, code, errD := services.NewClient(db, esServices).DoJSONRequest(context.Background(), "POST", "/events", e, nil)
			if code >= 400 || errD != nil {
				log.Error(ctx, "PushInElasticSearch> Unable to send event %s to elasticsearch [%d]: %v", e.EventType, code, errD)
				continue
			}
		}
	}
}

// GetEvents retrieves events from elasticsearch
func GetEvents(ctx context.Context, db gorp.SqlExecutor, store cache.Store, filters sdk.EventFilter) ([]json.RawMessage, error) {
	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeElasticsearch)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to get elasticsearch service")
	}

	var esEvents []elastic.SearchHit
	if _, _, err := services.NewClient(db, srvs).DoJSONRequest(context.Background(), "GET", "/events", filters, &esEvents); err != nil {
		return nil, sdk.WrapError(err, "Unable to get events")
	}

	events := make([]json.RawMessage, 0, len(esEvents))
	for _, h := range esEvents {
		events = append(events, *h.Source)
	}
	return events, nil
}
