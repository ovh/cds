package elasticsearch

import (
	"context"

	"github.com/olivere/elastic/v7"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/event"
)

const indexNotFoundException = "index_not_found_exception"

// Service is the elasticsearch service
type Service struct {
	service.Common
	Cfg      Configuration
	Router   *api.Router
	esClient ESClient
}

type ESClient interface {
	IndexDoc(ctx context.Context, index, docType, id string, body interface{}) (*elastic.IndexResponse, error)
	SearchDoc(ctx context.Context, indices []string, docType string, query elastic.Query, sorts []elastic.Sorter, from, size int) (*elastic.SearchResult, error)
	Ping(ctx context.Context, url string) (*elastic.PingResult, int, error)
	IndexDocWithoutType(ctx context.Context, index, id string, body interface{}) (*elastic.IndexResponse, error)
}

type esClient struct {
	client *elastic.Client
}

func (c *esClient) IndexDoc(ctx context.Context, index, docType, id string, body interface{}) (*elastic.IndexResponse, error) {
	if id == "" {
		c.client.Index().Index(index).Type(docType).BodyJson(body).Do(ctx)
	}
	return c.client.Index().Index(index).Type(docType).Id(id).BodyJson(body).Do(ctx)
}

func (c *esClient) IndexDocWithoutType(ctx context.Context, index, id string, body interface{}) (*elastic.IndexResponse, error) {
	if id == "" {
		c.client.Index().Index(index).BodyJson(body).Do(ctx)
	}
	return c.client.Index().Index(index).Id(id).BodyJson(body).Do(ctx)
}

func (c *esClient) SearchDoc(ctx context.Context, indices []string, docType string, query elastic.Query, sorts []elastic.Sorter, from, size int) (*elastic.SearchResult, error) {
	if from > -1 {
		return c.client.Search().Index(indices...).Type(docType).Query(query).SortBy(sorts...).From(from).Size(10).Do(ctx)
	}
	return c.client.Search().Index(indices...).Type(docType).Query(query).SortBy(sorts...).Size(10).Do(ctx)
}

func (c *esClient) Ping(ctx context.Context, url string) (*elastic.PingResult, int, error) {
	return c.client.Ping(url).Do(ctx)
}

var _ ESClient = new(esClient)

// Configuration is the vcs configuration structure
type Configuration struct {
	Name          string                          `toml:"name" comment:"Name of this CDS elasticsearch Service\n Enter a name to enable this service" json:"name"`
	HTTP          service.HTTPRouterConfiguration `toml:"http" comment:"######################\n CDS Elasticsearch HTTP Configuration \n######################" json:"http"`
	URL           string                          `default:"http://localhost:8088" json:"url"`
	ElasticSearch struct {
		URL             string `toml:"url" json:"url"`
		Username        string `toml:"username" json:"username"`
		Password        string `toml:"password" json:"-"`
		IndexEventsV2   string `toml:"indexEventsV2" commented:"true" comment:"index to store CDS events v2" json:"indexEventsV2"`
		IndexEvents     string `toml:"indexEvents" commented:"true" comment:"index to store CDS events" json:"indexEvents"`
		IndexMetrics    string `toml:"indexMetrics" commented:"true" comment:"index to store CDS metrics" json:"indexMetrics"`
		IndexJobSummary string `toml:"indexJobSummary" commented:"true" comment:"index to store CDS jobs summaries" json:"indexJobSummary"`
	} `toml:"elasticsearch" comment:"######################\n CDS ElasticSearch Settings \nSupport for elasticsearch 5.6\n######################" json:"elasticsearch"`
	EventBus struct {
		JobSummaryKafka event.KafkaConsumerConfig `toml:"jobSummaryKafka" json:"jobSummaryKafka" commented:"true" mapstructure:"jobSummaryKafka"`
	} `toml:"events" json:"events" commented:"true" mapstructure:"events"`
	API service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS Indexes Settings \n######################" json:"api"`
}
