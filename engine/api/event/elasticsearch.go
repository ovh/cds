package event

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PushInElasticSearch pushes event to an elasticsearch
func PushInElasticSearch(c context.Context, db gorp.SqlExecutor, store cache.Store) {
	querier := services.Querier(db, store)

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

			esServices, errS := querier.FindByType(services.TypeElasticsearch)
			if errS != nil {
				log.Warning("PushInElasticSearch> Unable to get elasticsearch service")
				continue
			}

			if len(esServices) == 0 {
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
