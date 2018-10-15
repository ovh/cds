package metrics

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"
	"gopkg.in/olivere/elastic.v5"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/venom"
)

var metricsChan chan sdk.Metric

// Init the metrics package which push to elasticSearch service
func Init(ctx context.Context, DBFunc func() *gorp.DbMap) {
	metricsChan = make(chan sdk.Metric, 50)
	sdk.GoRoutine(ctx, "metrics.PushInElasticSearch", func(c context.Context) { pushInElasticSearch(c, DBFunc) })
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
func GetMetrics(db gorp.SqlExecutor, key string, appID int64, metricName string) ([]json.RawMessage, error) {
	metricsRequest := sdk.MetricRequest{
		ProjectKey:    key,
		ApplicationID: appID,
		Key:           metricName,
	}

	srvs, err := services.FindByType(db, services.TypeElasticsearch)
	if err != nil {
		return nil, sdk.WrapError(err, "GetMetrics> Unable to get elasticsearch service")
	}

	var esMetrics []elastic.SearchHit
	if _, err := services.DoJSONRequest(context.Background(), srvs, "GET", "/metrics", metricsRequest, &esMetrics); err != nil {
		return nil, sdk.WrapError(err, "GetMetrics> Unable to get metrics")
	}

	events := make([]json.RawMessage, 0, len(esMetrics))
	for _, h := range esMetrics {
		events = append(events, *h.Source)
	}
	return events, nil
}

// PushVulnerabilities Create metrics from vulnerabilities and send them
func PushVulnerabilities(projKey string, appID int64, workflowID int64, num int64, summary map[string]int64) {
	m := sdk.Metric{
		Date:          time.Now(),
		ProjectKey:    projKey,
		WorkflowID:    workflowID,
		Num:           num,
		ApplicationID: appID,
		Key:           sdk.MetricKeyVulnerability,
		Value:         summary,
	}
	metricsChan <- m
}

// PushUnitTests Create metrics from unit tests and send them
func PushUnitTests(projKey string, appID int64, workflowID int64, num int64, tests venom.Tests) {
	m := sdk.Metric{
		Date:          time.Now(),
		ProjectKey:    projKey,
		ApplicationID: appID,
		WorkflowID:    workflowID,
		Key:           sdk.MetricKeyUnitTest,
		Num:           num,
	}

	summary := make(map[string]int, 3)
	summary["total"] = tests.Total
	summary["ko"] = tests.TotalKO
	summary["ok"] = tests.TotalOK
	summary["skip"] = tests.TotalSkipped

	m.Value = summary

	metricsChan <- m
}
