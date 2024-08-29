package elasticsearch

import (
	"context"

	elastic "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/index"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/searchtype"

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
	SearchDoc(ctx context.Context, index string, docType string, query *types.Query, sorts []types.SortCombinations, from, size int) (*search.Response, error)
	Ping(ctx context.Context) (bool, error)
	IndexDocWithoutType(ctx context.Context, index, id string, body interface{}) (*index.Response, error)
}

type esClient struct {
	client *elastic.TypedClient
}

func (c *esClient) IndexDocWithoutType(ctx context.Context, index, id string, body interface{}) (*index.Response, error) {
	if id == "" {
		return c.client.Index(index).Request(body).Do(ctx)
	}
	return c.client.Index(index).Id(id).Request(body).Do(ctx)
}

func (c *esClient) SearchDoc(ctx context.Context, index string, docType string, query *types.Query, sorts []types.SortCombinations, from, size int) (*search.Response, error) {
	if from > -1 {
		return c.client.Search().Index(index).SearchType(searchtype.SearchType{Name: docType}).Query(query).Sort(sorts...).From(from).Size(10).Do(ctx)
	}
	return c.client.Search().Index(index).SearchType(searchtype.SearchType{Name: docType}).Query(query).Sort(sorts...).Size(10).Do(ctx)
}

func (c *esClient) Ping(ctx context.Context) (bool, error) {
	return c.client.Core.Ping().IsSuccess(ctx)
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
