package event

import (
	"context"
	"encoding/json"

	"github.com/go-gorp/gorp"
	"gopkg.in/olivere/elastic.v6"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PushToMetrics pushes event to a metrics service
func PushToMetrics(c context.Context, db gorp.SqlExecutor, store cache.Store) {
	eventChan := make(chan sdk.Event, 10)
	Subscribe(eventChan)

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("PushToMetrics> Exiting: %v", c.Err())
				return
			}
		case e := <-eventChan:
			metricsSvc, errS := services.FindByType(db, services.TypeMetrics)
			if errS != nil {
				log.Error("PushToMetrics> Unable to get metrics service: %v", errS)
				continue
			}

			if len(metricsSvc) == 0 {
				continue
			}

			switch e.EventType {
			case "sdk.EventJob", "sdk.EventEngine":
				continue
			}
			e.Payload = nil
			code, errD := services.DoJSONRequest(context.Background(), metricsSvc, "POST", "/events", e, nil)
			if code >= 400 || errD != nil {
				log.Error("PushToMetrics> Unable to send event %s [%d]: %v", e.EventType, code, errD)
				continue
			}
		}
	}
}

// GetEvents retrieves events from metrics service
func GetEvents(db gorp.SqlExecutor, store cache.Store, filters sdk.EventFilter) ([]json.RawMessage, error) {
	srvs, err := services.FindByType(db, services.TypeMetrics)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to get metrics service")
	}

	var esEvents []elastic.SearchHit
	if _, err := services.DoJSONRequest(context.Background(), srvs, "GET", "/events", filters, &esEvents); err != nil {
		return nil, sdk.WrapError(err, "Unable to get events")
	}

	events := make([]json.RawMessage, 0, len(esEvents))
	for _, h := range esEvents {
		events = append(events, *h.Source)
	}
	return events, nil
}
