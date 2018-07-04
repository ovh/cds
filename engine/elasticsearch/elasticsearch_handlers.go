package elasticsearch

import (
	"context"
	"fmt"
	"net/http"

	"gopkg.in/olivere/elastic.v5"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
)

func (s *Service) getEventsHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		var filters sdk.EventFilter
		if err := api.UnmarshalBody(r, &filters); err != nil {
			return sdk.WrapError(err, "getEventsHandler> Unable to read body")
		}

		boolQuery := elastic.NewBoolQuery()
		for _, p := range filters.Filter.Projects {
			if p.AllWorkflows {
				boolQuery.Should(elastic.NewMatchQuery("project_key", p.Key))
			} else {
				for _, w := range p.WorkflowNames {
					boolQuery.Should(elastic.NewQueryStringQuery(fmt.Sprintf("project_key:%s AND workflow_name:%s", p.Key, w)))
				}

			}
		}

		result, errR := esClient.Search().Index(s.Cfg.ElasticSearch.Index).Type("sdk.EventRunWorkflow").Query(boolQuery).Sort("timestamp", false).From(filters.CurrentItem).Size(15).Do(context.Background())
		if errR != nil {
			return sdk.WrapError(errR, "getEventsHandler> Cannot get result on index: %s", s.Cfg.ElasticSearch.Index)
		}
		return api.WriteJSON(w, result.Hits.Hits, http.StatusOK)
	}
}

func (s *Service) postEventHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var e sdk.Event
		if err := api.UnmarshalBody(r, &e); err != nil {
			return sdk.WrapError(err, "postEventHandler> Unable to read body")
		}

		_, errI := esClient.Index().Index(s.Cfg.ElasticSearch.Index).Type(e.EventType).BodyJson(e).Do(context.Background())
		if errI != nil {
			return sdk.WrapError(errI, "postEventHandler> Unable to insert event")
		}
		return nil
	}
}

func (s *Service) getStatusHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return api.WriteJSON(w, s.Status(), status)
	}
}
