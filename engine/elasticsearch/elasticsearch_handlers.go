package elasticsearch

import (
	"context"
	"fmt"
	"net/http"

	"gopkg.in/olivere/elastic.v5"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"strconv"
)

var indexEvent = ""
var indexMetric = ""

func (s *Service) getEventsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		var filters sdk.EventFilter
		if err := api.UnmarshalBody(r, &filters); err != nil {
			return sdk.WrapError(err, "getEventsHandler> Unable to read body")
		}

		boolQuery := elastic.NewBoolQuery()
		for _, p := range filters.Filter.Projects {
			for _, w := range p.WorkflowNames {
				boolQuery.Should(elastic.NewQueryStringQuery(fmt.Sprintf("project_key:%s AND workflow_name:%s", p.Key, w)))
			}

		}

		result, errR := esClient.Search().Index(indexEvent).Type("sdk.EventRunWorkflow").Query(boolQuery).Sort("timestamp", false).From(filters.CurrentItem).Size(15).Do(context.Background())
		if errR != nil {
			return sdk.WrapError(errR, "getEventsHandler> Cannot get result on index: %s", indexEvent)
		}
		return service.WriteJSON(w, result.Hits.Hits, http.StatusOK)
	}
}

func (s *Service) postEventHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var e sdk.Event
		if err := api.UnmarshalBody(r, &e); err != nil {
			return sdk.WrapError(err, "postEventHandler> Unable to read body")
		}

		_, errI := esClient.Index().Index(indexEvent).Type(e.EventType).BodyJson(e).Do(context.Background())
		if errI != nil {
			return sdk.WrapError(errI, "postEventHandler> Unable to insert event")
		}
		return nil
	}
}

func (s *Service) getMetricsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var request sdk.MetricRequest
		if err := api.UnmarshalBody(r, &request); err != nil {
			return sdk.WrapError(err, "getMetricsHandler> unable to read request")
		}

		log.Warning("Request: %+v", request)
		stringQuery := fmt.Sprintf("key:%s AND project_key:%s", request.Key, request.ProjectKey)
		if request.ApplicationID != 0 {
			stringQuery = fmt.Sprintf("%s AND application_id:%d", stringQuery, request.ApplicationID)
		}
		if request.WorkflowID != 0 {
			stringQuery = fmt.Sprintf("%s AND workflow_id:%s", stringQuery, request.WorkflowID)
		}

		log.Warning("ES Query: %s", stringQuery)
		log.Warning("Index: %s", indexMetric)
		results, errR := esClient.Search().
			Index(indexMetric).
			Type(fmt.Sprintf("%T", sdk.Metric{})).
			Query(elastic.NewBoolQuery().Must(elastic.NewQueryStringQuery(stringQuery))).
			Sort("timestamp", false).
			Size(10).
			Do(context.Background())
		if errR != nil {
			return sdk.WrapError(errR, "getMetricsHandler> Unable to get result")
		}
		return service.WriteJSON(w, results.Hits.Hits, http.StatusOK)
	}
}

func (s *Service) postMetricsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var metric sdk.Metric
		if err := api.UnmarshalBody(r, &metric); err != nil {
			return sdk.WrapError(err, "postEventHandler> Unable to read body")
		}
		_, errI := esClient.Index().Index(indexMetric).Type(fmt.Sprintf("%T", sdk.Metric{})).Timestamp(strconv.Itoa(int(metric.Date.Unix()))).BodyJson(metric).Do(context.Background())
		if errI != nil {
			return sdk.WrapError(errI, "postEventHandler> Unable to insert event")
		}
		return nil
	}
}

func (s *Service) getStatusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(), status)
	}
}
