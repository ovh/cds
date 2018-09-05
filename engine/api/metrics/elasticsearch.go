package metrics

import (
	"context"
	"encoding/json"

	"github.com/go-gorp/gorp"
	"gopkg.in/olivere/elastic.v5"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var metricsChan chan sdk.Metric

func Init(c context.Context, DBFunc func() *gorp.DbMap) {
	metricsChan = make(chan sdk.Metric, 50)
	sdk.GoRoutine("metrics.PushInElasticSearch", func() { pushInElasticSearch(c, DBFunc) })
}

func pushInElasticSearch(c context.Context, DBFunc func() *gorp.DbMap) {
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("metrics.pushInElasticSearch> Exiting: %v", c.Err())
				return
			}
		case e := <-metricsChan:
			db := DBFunc()
			esServices, errS := services.FindByType(db, services.TypeElasticsearch)
			if errS != nil {
				log.Error("metrics.pushInElasticSearch> Unable to get elasticsearch service: %v", errS)
				continue
			}

			if len(esServices) == 0 {
				continue
			}

			code, errD := services.DoJSONRequest(context.Background(), esServices, "POST", "/metrics", e, nil)
			if code >= 400 || errD != nil {
				log.Error("metrics.pushInElasticSearch> Unable to send metrics to elasticsearch [%d]: %v", code, errD)
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

	var esMetrics []elastic.SearchHit
	if _, err := services.DoJSONRequest(context.Background(), srvs, "GET", "/metrics", req, &esMetrics); err != nil {
		return nil, sdk.WrapError(err, "GetMetrics> Unable to get metrics")
	}

	events := make([]json.RawMessage, 0, len(esMetrics))
	for _, h := range esMetrics {
		events = append(events, *h.Source)
	}
	return events, nil
}
