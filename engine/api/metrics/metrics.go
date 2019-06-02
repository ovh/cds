package metrics

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/sguiheux/go-coverage"
	"gopkg.in/olivere/elastic.v6"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/venom"
)

var metricsChan chan sdk.Metric

// Init the metrics package which push to elasticSearch service
func Init(c context.Context, DBFunc func() *gorp.DbMap) {
	metricsChan = make(chan sdk.Metric, 50)

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("metrics.init> Exiting: %v", c.Err())
				return
			}
		case e := <-metricsChan:
			db := DBFunc()
			metSrvcs, errS := services.FindByType(db, services.TypeMetrics)
			if errS != nil {
				log.Error("metrics.init> Unable to get metrics service: %v", errS)
				continue
			}

			if len(metSrvcs) == 0 {
				continue
			}

			code, errD := services.DoJSONRequest(context.Background(), metSrvcs, "POST", "/metrics", e, nil)
			if code >= 400 || errD != nil {
				log.Error("metrics.init> Unable to send metrics [%d]: %v", code, errD)
				continue
			}
		}
	}
}

// GetMetrics retrieves metrics from metrics service
func GetMetrics(db gorp.SqlExecutor, key string, appID int64, metricName string) ([]json.RawMessage, error) {
	metricsRequest := sdk.MetricRequest{
		ProjectKey:    key,
		ApplicationID: appID,
		Key:           metricName,
	}

	srvs, err := services.FindByType(db, services.TypeMetrics)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to get metrics service")
	}

	var mets []elastic.SearchHit
	if _, err := services.DoJSONRequest(context.Background(), srvs, "GET", "/metrics", metricsRequest, &mets); err != nil {
		return nil, sdk.WrapError(err, "Unable to get metrics")
	}

	events := make([]json.RawMessage, len(mets))
	for i := range mets {
		events[len(mets)-1-i] = *mets[i].Source
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
func PushUnitTests(projKey string, appID int64, workflowID int64, num int64, tests venom.Tests) {
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

// PushCoverage Create metrics from coverage and send them
func PushCoverage(projKey string, appID int64, workflowID int64, num int64, cover coverage.Report) {
	m := sdk.Metric{
		Date:          time.Now(),
		ProjectKey:    projKey,
		ApplicationID: appID,
		WorkflowID:    workflowID,
		Key:           sdk.MetricKeyCoverage,
		Num:           num,
	}

	summary := make(map[string]float64, 3)
	summary["covered_lines"] = float64(cover.CoveredLines)
	summary["total_lines"] = float64(cover.TotalLines)
	summary["percent"] = (summary["covered_lines"] / summary["total_lines"]) * 100

	m.Value = summary

	metricsChan <- m
}
