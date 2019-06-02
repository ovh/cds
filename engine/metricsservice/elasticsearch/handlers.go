package elasticsearch

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"gopkg.in/olivere/elastic.v6"
)

func (es *ES) GetEvents(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if es.EventIndex == "" {
		return sdk.WrapError(sdk.ErrNotFound, "No events index found")
	}

	var filters sdk.EventFilter
	if err := service.UnmarshalBody(r, &filters); err != nil {
		return sdk.WrapError(err, "Unable to read body")
	}

	boolQuery := elastic.NewBoolQuery()
	for _, p := range filters.Filter.Projects {
		for _, w := range p.WorkflowNames {
			boolQuery.Should(elastic.NewQueryStringQuery(fmt.Sprintf("project_key:%s AND workflow_name:%s", p.Key, w)))
		}

	}
	result, errR := es.client.Search().Index(es.EventIndex).Type("sdk.EventRunWorkflow").Query(boolQuery).Sort("timestamp", false).From(filters.CurrentItem).Size(15).Do(context.Background())
	if errR != nil {
		if strings.Contains(errR.Error(), indexNotFoundException) {
			log.Warning("elasticsearch> getEventsHandler> %v", errR.Error())
			return service.WriteJSON(w, nil, http.StatusOK)
		}
		esReq := fmt.Sprintf(`client.Search().Index(%+v).Type("sdk.EventRunWorkflow").Query(%+v).Sort("timestamp", false).From(%+v).Size(15)`, es.EventIndex, boolQuery, filters.CurrentItem)
		return sdk.WrapError(errR, "Cannot get result on index: %s : query -> %s", es.EventIndex, esReq)
	}
	return service.WriteJSON(w, result.Hits.Hits, http.StatusOK)
}

func (es *ES) PostEvents(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if es.EventIndex == "" {
		return sdk.WrapError(sdk.ErrNotFound, "No events index found")
	}

	var e sdk.Event
	if err := service.UnmarshalBody(r, &e); err != nil {
		return sdk.WrapError(err, "Unable to read body")
	}

	_, errI := es.client.Index().Index(es.EventIndex).Type(e.EventType).BodyJson(e).Do(context.Background())
	if errI != nil {
		return sdk.WrapError(errI, "Unable to insert event")
	}
	return nil
}

func (es *ES) GetMetrics(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if es.MetricsIndex == "" {
		return sdk.WrapError(sdk.ErrNotFound, "getMetricsHandler> No metrics index found")
	}

	var request sdk.MetricRequest
	if err := service.UnmarshalBody(r, &request); err != nil {
		return sdk.WrapError(err, "Unable to read request")
	}

	stringQuery := fmt.Sprintf("key:%s AND project_key:%s", request.Key, request.ProjectKey)
	if request.ApplicationID != 0 {
		stringQuery = fmt.Sprintf("%s AND application_id:%d", stringQuery, request.ApplicationID)
	}
	if request.WorkflowID != 0 {
		stringQuery = fmt.Sprintf("%s AND workflow_id:%d", stringQuery, request.WorkflowID)
	}

	results, errR := es.client.Search().
		Index(es.MetricsIndex).
		Type(fmt.Sprintf("%T", sdk.Metric{})).
		Query(elastic.NewBoolQuery().Must(elastic.NewQueryStringQuery(stringQuery))).
		Sort("run", false).
		Size(10).
		Do(context.Background())
	if errR != nil {
		if strings.Contains(errR.Error(), indexNotFoundException) {
			log.Warning("elasticsearch> getMetricsHandler> %v", errR.Error())
			return service.WriteJSON(w, nil, http.StatusOK)
		}
		return sdk.WrapError(errR, "Unable to get result")
	}

	return service.WriteJSON(w, results.Hits.Hits, http.StatusOK)
}

func (es *ES) PostMetrics(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if es.MetricsIndex == "" {
		return sdk.WrapError(sdk.ErrNotFound, "postMetricsHandler> No metrics index found")
	}

	var metric sdk.Metric
	if err := service.UnmarshalBody(r, &metric); err != nil {
		return sdk.WrapError(err, "Unable to read body")
	}

	id := fmt.Sprintf("%s-%d-%d-%d-%s", metric.ProjectKey, metric.WorkflowID, metric.ApplicationID, metric.Num, metric.Key)

	// Get metrics if already exists
	existingMetric, err := es.loadMetric(id)
	if err != nil {
		return sdk.WrapError(err, "unable to load metric")
	}
	if existingMetric.Value != nil {
		es.mergeMetric(&metric, existingMetric.Value)
	}

	_, errI := es.client.Index().Index(es.MetricsIndex).Id(id).Type(fmt.Sprintf("%T", sdk.Metric{})).BodyJson(metric).Do(context.Background())
	if errI != nil {
		return sdk.WrapError(errI, "Unable to insert event")
	}
	return nil
}
