package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gopkg.in/olivere/elastic.v5"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) getEventsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if s.Cfg.ElasticSearch.IndexEvents == "" {
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

		result, errR := esClient.Search().Index(s.Cfg.ElasticSearch.IndexEvents).Type("sdk.EventRunWorkflow").Query(boolQuery).Sort("timestamp", false).From(filters.CurrentItem).Size(15).Do(context.Background())
		if errR != nil {
			if strings.Contains(errR.Error(), "index_not_found_exception") {
				return service.WriteJSON(w, nil, http.StatusOK)
			}
			return sdk.WrapError(errR, "Cannot get result on index: %s", s.Cfg.ElasticSearch.IndexEvents)
		}
		return service.WriteJSON(w, result.Hits.Hits, http.StatusOK)
	}
}

func (s *Service) postEventHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if s.Cfg.ElasticSearch.IndexEvents == "" {
			return sdk.WrapError(sdk.ErrNotFound, "No events index found")
		}

		var e sdk.Event
		if err := service.UnmarshalBody(r, &e); err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}

		_, errI := esClient.Index().Index(s.Cfg.ElasticSearch.IndexEvents).Type(e.EventType).BodyJson(e).Do(context.Background())
		if errI != nil {
			return sdk.WrapError(errI, "Unable to insert event")
		}
		return nil
	}
}

func (s *Service) getMetricsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if s.Cfg.ElasticSearch.IndexMetrics == "" {
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

		results, errR := esClient.Search().
			Index(s.Cfg.ElasticSearch.IndexMetrics).
			Type(fmt.Sprintf("%T", sdk.Metric{})).
			Query(elastic.NewBoolQuery().Must(elastic.NewQueryStringQuery(stringQuery))).
			Sort("run", false).
			Size(10).
			Do(context.Background())
		if errR != nil {
			return sdk.WrapError(errR, "Unable to get result")
		}

		return service.WriteJSON(w, results.Hits.Hits, http.StatusOK)
	}
}

func (s *Service) postMetricsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if s.Cfg.ElasticSearch.IndexMetrics == "" {
			return sdk.WrapError(sdk.ErrNotFound, "postMetricsHandler> No metrics index found")
		}

		var metric sdk.Metric
		if err := service.UnmarshalBody(r, &metric); err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}

		id := fmt.Sprintf("%s-%d-%d-%d-%s", metric.ProjectKey, metric.WorkflowID, metric.ApplicationID, metric.Num, metric.Key)

		// Get metrics if already exists
		existingMetric, err := s.loadMetric(id)
		if err != nil {
			return sdk.WrapError(err, "unable to load metric")
		}
		if existingMetric.Value != nil {
			s.mergeMetric(&metric, existingMetric.Value)
		}

		_, errI := esClient.Index().Index(s.Cfg.ElasticSearch.IndexMetrics).Id(id).Type(fmt.Sprintf("%T", sdk.Metric{})).Timestamp(strconv.Itoa(int(metric.Date.Unix()))).BodyJson(metric).Do(context.Background())
		if errI != nil {
			return sdk.WrapError(errI, "Unable to insert event")
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

func (s *Service) loadMetric(ID string) (sdk.Metric, error) {
	var m sdk.Metric
	results, errR := esClient.Search().
		Index(s.Cfg.ElasticSearch.IndexMetrics).
		Type(fmt.Sprintf("%T", sdk.Metric{})).
		Query(elastic.NewBoolQuery().Must(elastic.NewQueryStringQuery(fmt.Sprintf("_id:%s", ID)))).
		Sort("_score", false).
		Sort("run", false).
		Size(10).
		Do(context.Background())
	if errR != nil {
		return m, sdk.WrapError(errR, "unable to get result")
	}

	if len(results.Hits.Hits) == 0 {
		return m, nil
	}

	if err := json.Unmarshal(*results.Hits.Hits[0].Source, &m); err != nil {
		return m, err
	}
	return m, nil
}

func (s *Service) mergeMetric(newMetric *sdk.Metric, oldMetricValue map[string]float64) {
	for k, v := range oldMetricValue {
		if _, has := newMetric.Value[k]; has {
			newMetric.Value[k] = newMetric.Value[k] + v
		} else {
			newMetric.Value[k] = v
		}
	}
}
