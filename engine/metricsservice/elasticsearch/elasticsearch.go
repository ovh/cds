package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"gopkg.in/olivere/elastic.v6"
)

type ES struct {
	client       *elastic.Client
	Endpoint     string
	EventIndex   string
	MetricsIndex string
}

const (
	indexNotFoundException = "index_not_found_exception"
	componentName          = "ElasticSearch"
)

func New(url, username, password string, sniff bool, eventIndex, metricsIndex string) (*ES, error) {
	c, e := elastic.NewClient(
		elastic.SetURL(url),
		elastic.SetBasicAuth(username, password),
		elastic.SetSniff(sniff),
	)
	if e != nil {
		return nil, sdk.WrapError(e, "unable to initialize ElasticSearch client")
	}
	v, e := c.ElasticsearchVersion(url)
	if e != nil {
		return nil, sdk.WrapError(e, "failed to connect to ElasticSearch instance: %s", url)
	}
	log.Debug("Connected to ElasticSearch running version %s", v)
	return &ES{
		client:       c,
		Endpoint:     url,
		EventIndex:   eventIndex,
		MetricsIndex: metricsIndex,
	}, nil
}

func (es *ES) loadMetric(ID string) (sdk.Metric, error) {
	var m sdk.Metric
	results, errR := es.client.Search().
		Index(es.MetricsIndex).
		Type(fmt.Sprintf("%T", sdk.Metric{})).
		Query(elastic.NewBoolQuery().Must(elastic.NewQueryStringQuery(fmt.Sprintf("_id:%s", ID)))).
		Sort("_score", false).
		Sort("run", false).
		Size(10).
		Do(context.Background())
	if errR != nil {
		if strings.Contains(errR.Error(), indexNotFoundException) {
			log.Warning("elasticsearch> loadMetric> %v", errR.Error())
			return m, nil
		}
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

func (es *ES) mergeMetric(newMetric *sdk.Metric, oldMetricValue map[string]float64) {
	for k, v := range oldMetricValue {
		if _, has := newMetric.Value[k]; has {
			newMetric.Value[k] = newMetric.Value[k] + v
		} else {
			newMetric.Value[k] = v
		}
	}
}
