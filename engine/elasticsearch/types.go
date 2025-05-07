package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/defensestation/osquery"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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
	SearchDoc(ctx context.Context, index string, query *osquery.SearchRequest, sorts []string, from, size int) (*opensearchapi.SearchResp, error)
	Ping(ctx context.Context) error
	IndexDocWithoutType(ctx context.Context, index, id string, body interface{}) (*opensearchapi.DocumentCreateResp, error)
}

type esClient struct {
	client *opensearchapi.Client
}

func (c *esClient) IndexDocWithoutType(ctx context.Context, index, id string, body interface{}) (*opensearchapi.DocumentCreateResp, error) {
	btes, err := json.Marshal(body)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to prepare index body")
	}
	if id == "" {
		id = sdk.UUID()
	}

	r := opensearchapi.DocumentCreateReq{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(btes),
	}

	var path strings.Builder
	// https://opensearch.org/docs/latest/api-reference/document-apis/index-document/#path-and-http-methods
	path.Grow(10 + len(r.Index) + len(r.DocumentID))
	path.WriteString("/")
	path.WriteString(r.Index)
	path.WriteString("/_doc/")
	path.WriteString(r.DocumentID)
	br, err := opensearch.BuildRequest(
		"PUT",
		path.String(),
		r.Body,
		map[string]string{},
		r.Header,
	)
	if err != nil {
		return nil, sdk.WrapError(err, "error on BuildRequest")
	}

	documentResp := opensearchapi.DocumentCreateResp{}

	resp, err := c.Do(ctx, br, &documentResp)
	if err != nil {
		return nil, err
	}
	if resp != nil && resp.StatusCode > 400 {
		return nil, sdk.WrapError(err, "error on create or update document")
	}
	return &documentResp, nil
}

func (c *esClient) Do(ctx context.Context, httpReq *http.Request, dataPointer interface{}) (*opensearch.Response, error) {
	httpReq = httpReq.WithContext(ctx)
	resp, err := c.client.Client.Perform(httpReq)
	if err != nil {
		return nil, err
	}

	response := &opensearch.Response{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
		Header:     resp.Header,
	}

	if dataPointer != nil && resp.Body != nil && !response.IsError() {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return response, sdk.WrapError(err, "error on read body")
		}

		response.Body = io.NopCloser(bytes.NewReader(data))

		if err := json.Unmarshal(data, dataPointer); err != nil {
			return response, sdk.WrapError(err, "error on json unmarshal data: %v", data)
		}
	}

	return response, nil
}

func (c *esClient) SearchDoc(ctx context.Context, index string, query *osquery.SearchRequest, sorts []string, from, size int) (*opensearchapi.SearchResp, error) {
	var body bytes.Buffer
	_ = json.NewEncoder(&body).Encode(query.Map())

	params := opensearchapi.SearchParams{
		Sort: sorts,
		Size: &size,
	}

	if from > -1 {
		params.From = &from
	}

	return c.client.Search(ctx, &opensearchapi.SearchReq{
		Indices: []string{index},
		Body:    &body,
		Params:  params,
	})
}

func (c *esClient) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &opensearchapi.PingReq{Params: opensearchapi.PingParams{}})
	return err
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
