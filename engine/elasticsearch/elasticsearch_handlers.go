package elasticsearch

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
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

		boolQuery := types.Query{
			Bool: &types.BoolQuery{
				Must: []types.Query{
					types.Query{
						SimpleQueryString: &types.SimpleQueryStringQuery{
							Query: "type_event:sdk.EventRunWorkflow",
						},
					},
				},
			},
		}

		for _, p := range filters.Filter.Projects {
			for _, w := range p.WorkflowNames {
				boolQuery.Bool.Must = append(boolQuery.Bool.Must,
					types.Query{
						SimpleQueryString: &types.SimpleQueryStringQuery{
							Query: "project_key:" + p.Key,
						},
					},
					types.Query{
						SimpleQueryString: &types.SimpleQueryStringQuery{
							Query: "workflow_name:" + w,
						},
					})
			}
		}

		result, err := s.esClient.SearchDoc(ctx,
			s.Cfg.ElasticSearch.IndexEvents,
			fmt.Sprintf("%T", sdk.Event{}),
			&boolQuery,
			[]types.SortCombinations{"timestamp"}, // default is DESC: https://github.com/elastic/elasticsearch-specification/blob/07bf82537a186562d8699685e3704ea338b268ef/specification/_types/sort.ts#L93-L97
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

		query := types.Query{
			Bool: &types.BoolQuery{
				Must: []types.Query{
					types.Query{
						SimpleQueryString: &types.SimpleQueryStringQuery{
							Query: "key:" + request.Key,
						},
					},
					types.Query{
						SimpleQueryString: &types.SimpleQueryStringQuery{
							Query: "project_key:" + request.ProjectKey,
						},
					},
				},
			},
		}

		if request.ApplicationID != 0 {
			query.Bool.Must = append(query.Bool.Must, types.Query{
				SimpleQueryString: &types.SimpleQueryStringQuery{
					Query: "application_id:" + strconv.FormatInt(request.ApplicationID, 10),
				},
			})
		}

		if request.WorkflowID != 0 {
			query.Bool.Must = append(query.Bool.Must, types.Query{
				SimpleQueryString: &types.SimpleQueryStringQuery{
					Query: "workflow_id:" + strconv.FormatInt(request.WorkflowID, 10),
				},
			})
		}

		results, err := s.esClient.SearchDoc(ctx,
			s.Cfg.ElasticSearch.IndexMetrics,
			fmt.Sprintf("%T", sdk.Metric{}),
			&query,
			[]types.SortCombinations{"run"}, // default is DESC: https://github.com/elastic/elasticsearch-specification/blob/07bf82537a186562d8699685e3704ea338b268ef/specification/_types/sort.ts#L93-L97
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

	query := &types.Query{
		Ids: &types.IdsQuery{
			Values: []string{ID},
		},
	}

	results, err := s.esClient.SearchDoc(ctx, s.Cfg.ElasticSearch.IndexMetrics,
		fmt.Sprintf("%T", sdk.Metric{}),
		query,
		[]types.SortCombinations{"_score", "run"},
		-1, 10)
	if err != nil {
		if strings.Contains(err.Error(), indexNotFoundException) {
			log.Warn(ctx, "elasticsearch> loadMetric> %v", err.Error())
			return m, nil
		}
		return m, sdk.WrapError(err, "unable to get result")
	}

	if len(results.Hits.Hits) == 0 {
		return m, nil
	}

	if err := sdk.JSONUnmarshal(results.Hits.Hits[0].Source_, &m); err != nil {
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
