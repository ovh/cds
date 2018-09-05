package metrics

import (
	"context"

	"github.com/go-gorp/gorp"

	"encoding/json"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"gopkg.in/olivere/elastic.v5"
)

var metricsChan chan sdk.Metric

func Init(c context.Context, db gorp.SqlExecutor) {
	metricsChan = make(chan sdk.Metric, 50)
	sdk.GoRoutine("metrics.PushInElasticSearch", func() { pushInElasticSearch(c, db) })
}

func pushInElasticSearch(c context.Context, db gorp.SqlExecutor) {
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("metrics.send> Exiting: %v", c.Err())
				return
			}
		case e := <-metricsChan:
			esServices, errS := services.FindByType(db, services.TypeElasticsearch)
			if errS != nil {
				log.Error("metrics.send> Unable to get elasticsearch service: %v", errS)
				continue
			}

			if len(esServices) == 0 {
				continue
			}

			code, errD := services.DoJSONRequest(context.Background(), esServices, "POST", "/metrics", e, nil)
			if code >= 400 || errD != nil {
				log.Error("metrics.send> Unable to send metricsto elasticsearch [%d]: %v", code, errD)
				continue
			}
		}
	}
}

// GetMetrics retrieves metrics from elasticsearch
func GetMetrics(db gorp.SqlExecutor, req sdk.MetricRequest) ([]json.RawMessage, error) {
	srvs, err := services.FindByType(db, services.TypeElasticsearch)
	if err != nil {
		return nil, sdk.WrapError(err, "GetMetrics> Unable to get elasticsearch service")
	}

	var esEvents []elastic.SearchHit
	if _, err := services.DoJSONRequest(context.Background(), srvs, "GET", "/metrics", req, &esEvents); err != nil {
		return nil, sdk.WrapError(err, "GetMetrics> Unable to get metrics")
	}

	events := make([]json.RawMessage, 0, len(esEvents))
	for _, h := range esEvents {
		events = append(events, *h.Source)
	}
	return events, nil
}
