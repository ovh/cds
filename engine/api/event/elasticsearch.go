package event

import (
	"context"
	"encoding/json"

	"github.com/go-gorp/gorp"
	"gopkg.in/olivere/elastic.v5"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PushInElasticSearch pushes event to an elasticsearch
func PushInElasticSearch(c context.Context, db gorp.SqlExecutor, store cache.Store) {
	eventChan := make(chan sdk.Event, 10)
	Subscribe(eventChan)

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("PushInElasticSearch> Exiting: %v", c.Err())
				return
			}
		case e := <-eventChan:
			esServices, errS := services.FindByType(db, services.TypeElasticsearch)
			if errS != nil {
				log.Error("PushInElasticSearch> Unable to get elasticsearch service: %v", errS)
				continue
			}

			if len(esServices) == 0 {
				continue
			}

			switch e.EventType {
			case "sdk.EventPipelineBuild", "sdk.EventJob", "sdk.EventEngine":
				continue
			}
			e.Payload = nil
			code, errD := services.DoJSONRequest(context.Background(), esServices, "POST", "/events", e, nil)
			if code >= 400 || errD != nil {
				log.Error("PushInElasticSearch> Unable to send event %s to elasticsearch [%d]: %v", e.EventType, code, errD)
				continue
			}
		}
	}
}

// GetEvents retrieves events from elasticsearch
func GetEvents(db gorp.SqlExecutor, store cache.Store, filters sdk.EventFilter) ([]json.RawMessage, error) {
	srvs, err := services.FindByType(db, services.TypeElasticsearch)
	if err != nil {
		return nil, sdk.WrapError(err, "GetEvent> Unable to get elasticsearch service")
	}

	var esEvents []elastic.SearchHit
	if _, err := services.DoJSONRequest(context.Background(), srvs, "GET", "/events", filters, &esEvents); err != nil {
		return nil, sdk.WrapError(err, "GetEvent> Unable to get events")
	}

	events := make([]json.RawMessage, 0, len(esEvents))
	for _, h := range esEvents {
		events = append(events, *h.Source)
	}
	return events, nil
}
