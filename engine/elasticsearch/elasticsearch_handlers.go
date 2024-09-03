package elasticsearch

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/defensestation/osquery"
	"github.com/rockbears/log"

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

		var conditions []osquery.Mappable
		conditions = append(conditions, osquery.Term("type_event", "sdk.EventRunWorkflow"))

		for _, p := range filters.Filter.Projects {
			for _, w := range p.WorkflowNames {
				conditions = append(conditions,
					osquery.Term("project_key", p.Key),
					osquery.Term("workflow_name", w),
				)
			}
		}

		query := osquery.Query(osquery.Bool().Must(conditions...))
		result, err := s.esClient.SearchDoc(ctx,
			s.Cfg.ElasticSearch.IndexEvents,
			query,
			[]string{"timestamp:desc"},
			filters.CurrentItem, 15)
		if err != nil {
			if strings.Contains(err.Error(), indexNotFoundException) {
				log.Warn(ctx, "elasticsearch> getEventsHandler> %v", err.Error())
				return service.WriteJSON(w, nil, http.StatusOK)
			}
			return sdk.WrapError(err, "cannot get result on index: %s", s.Cfg.ElasticSearch.IndexEvents)
		}
		return service.WriteJSON(w, result.Hits.Hits, http.StatusOK)
	}
}

func (s *Service) postEventV2Handler() service.Handler {
	return func(ctx context.Context, _ http.ResponseWriter, r *http.Request) error {
		if s.Cfg.ElasticSearch.IndexEventsV2 == "" {
			return sdk.WrapError(sdk.ErrNotFound, "No events v2 index found")
		}

		var e sdk.FullEventV2
		if err := service.UnmarshalBody(r, &e); err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}

		if _, err := s.esClient.IndexDocWithoutType(ctx, s.Cfg.ElasticSearch.IndexEventsV2, "", e); err != nil {
			return sdk.WrapError(err, "Unable to insert event v2")
		}
		return nil
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

		if _, err := s.esClient.IndexDocWithoutType(ctx, s.Cfg.ElasticSearch.IndexEvents, "", e); err != nil {
			return sdk.WrapError(err, "Unable to insert event")
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

		var conditions []osquery.Mappable
		conditions = append(conditions, osquery.Term("key", request.Key))
		conditions = append(conditions, osquery.Term("project_key", request.ProjectKey))
		if request.ApplicationID != 0 {
			conditions = append(conditions, osquery.Term("application_id", strconv.FormatInt(request.ApplicationID, 10)))
		}
		if request.WorkflowID != 0 {
			conditions = append(conditions, osquery.Term("workflow_id", strconv.FormatInt(request.WorkflowID, 10)))
		}

		query := osquery.Query(osquery.Bool().Must(conditions...))
		results, err := s.esClient.SearchDoc(ctx,
			s.Cfg.ElasticSearch.IndexMetrics,
			query,
			[]string{"run:desc"},
			-1, 10)
		if err != nil {
			if strings.Contains(err.Error(), indexNotFoundException) {
				log.Warn(ctx, "elasticsearch> getMetricsHandler> %v", err.Error())
				return service.WriteJSON(w, nil, http.StatusOK)
			}
			return sdk.WrapError(err, "Unable to get result")
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
		existingMetric, err := s.loadMetric(ctx, id)
		if err != nil {
			return sdk.WrapError(err, "unable to load metric")
		}
		if existingMetric.Value != nil {
			s.mergeMetric(&metric, existingMetric.Value)
		}

		if _, err := s.esClient.IndexDocWithoutType(ctx, s.Cfg.ElasticSearch.IndexMetrics, id, metric); err != nil {
			return sdk.WrapError(err, "Unable to insert event")
		}
		return nil
	}
}

func (s *Service) getStatusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(ctx), status)
	}
}

func (s *Service) loadMetric(ctx context.Context, ID string) (sdk.Metric, error) {
	var m sdk.Metric

	query := osquery.Query(osquery.IDs(ID))

	results, err := s.esClient.SearchDoc(ctx, s.Cfg.ElasticSearch.IndexMetrics,
		query,
		[]string{"_score:desc", "run:desc"},
		-1, 10)
	if err != nil {
		log.Warn(ctx, "elasticsearch> loadMetric> %v", err.Error())
		if strings.Contains(err.Error(), indexNotFoundException) {
			return m, nil
		}
		return m, sdk.WrapError(err, "unable to get result")
	}

	if len(results.Hits.Hits) == 0 {
		return m, nil
	}

	log.Info(ctx, "loadMetric : %v", string(results.Hits.Hits[0].Source))

	if err := sdk.JSONUnmarshal(results.Hits.Hits[0].Source, &m); err != nil {
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
