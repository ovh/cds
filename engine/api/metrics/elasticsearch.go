package metrics

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"gopkg.in/olivere/elastic.v6"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
)

var metricsChan chan sdk.Metric

// Init the metrics package which push to elasticSearch service
func Init(ctx context.Context, DBFunc func() *gorp.DbMap) {
	metricsChan = make(chan sdk.Metric, 50)

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "metrics.pushInElasticSearch> Exiting: %v", ctx.Err())
				return
			}
		case e := <-metricsChan:
			db := DBFunc()
			esServices, errS := services.LoadAllByType(ctx, db, sdk.TypeElasticsearch)
			if errS != nil {
				log.Error(ctx, "metrics.pushInElasticSearch> Unable to get elasticsearch service: %v", errS)
				continue
			}

			if len(esServices) == 0 {
				continue
			}

			_, code, errD := services.NewClient(DBFunc(), esServices).DoJSONRequest(context.Background(), "POST", "/metrics", e, nil)
			if code >= 400 || errD != nil {
				log.Error(ctx, "metrics.pushInElasticSearch> Unable to send metrics to elasticsearch [%d]: %v", code, errD)
				continue
			}
		}
	}
}

// GetMetrics retrieves metrics from elasticsearch
func GetMetrics(ctx context.Context, db gorp.SqlExecutor, key string, appID int64, metricName string) ([]json.RawMessage, error) {
	metricsRequest := sdk.MetricRequest{
		ProjectKey:    key,
		ApplicationID: appID,
		Key:           metricName,
	}

	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeElasticsearch)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to get elasticsearch service")
	}

	var esMetrics []elastic.SearchHit
	if _, _, err := services.NewClient(db, srvs).DoJSONRequest(context.Background(), "GET", "/metrics", metricsRequest, &esMetrics); err != nil {
		return nil, sdk.WrapError(err, "Unable to get metrics")
	}

	events := make([]json.RawMessage, len(esMetrics))
	for i := range esMetrics {
		events[len(esMetrics)-1-i] = *esMetrics[i].Source
	}
	return events, nil
}

// PushVulnerabilities Create metrics from vulnerabilities and send them
func PushVulnerabilities(projKey string, appID int64, workflowID int64, num int64, summary map[string]float64) {
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
func PushUnitTests(projKey string, appID int64, workflowID int64, num int64, tests sdk.TestsStats) {
	m := sdk.Metric{
		Date:          time.Now(),
		ProjectKey:    projKey,
		ApplicationID: appID,
		WorkflowID:    workflowID,
		Key:           sdk.MetricKeyUnitTest,
		Num:           num,
	}

	summary := make(map[string]float64, 3)
	summary["total"] = float64(tests.Total)
	summary["ko"] = float64(tests.TotalKO)
	summary["ok"] = float64(tests.TotalOK)
	summary["skip"] = float64(tests.TotalSkipped)

	m.Value = summary

	metricsChan <- m
}
