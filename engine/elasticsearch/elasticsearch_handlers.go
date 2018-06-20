package elasticsearch

import (
	"context"
	"net/http"

	"gopkg.in/olivere/elastic.v5"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
)

func (s *Service) getEventsHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		boolQuery := elastic.NewBoolQuery().Should(
			elastic.NewMatchQuery("ProjectKey", "*"),
		)
		result, errR := esClient.Search().Index(s.Cfg.ElasticSearch.Index).Type("sdk.EventRunWorkflow").Query(boolQuery).Sort("Timestamp", false).Do(context.Background())
		if errR != nil {
			return sdk.WrapError(errR, "getEventsHandler> Cannot get result")
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
