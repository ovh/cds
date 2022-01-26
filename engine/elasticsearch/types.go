package elasticsearch

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/event"
)

const indexNotFoundException = "index_not_found_exception"

// Service is the repostories service
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
}

// Configuration is the vcs configuration structure
type Configuration struct {
	Name          string                          `toml:"name" comment:"Name of this CDS elasticsearch Service\n Enter a name to enable this service" json:"name"`
	HTTP          service.HTTPRouterConfiguration `toml:"http" comment:"######################\n CDS Elasticsearch HTTP Configuration \n######################" json:"http"`
	URL           string                          `default:"http://localhost:8088" json:"url"`
	ElasticSearch struct {
		URL             string `toml:"url" json:"url"`
		Username        string `toml:"username" json:"username"`
		Password        string `toml:"password" json:"-"`
		IndexEvents     string `toml:"indexEvents" commented:"true" comment:"index to store CDS events" json:"indexEvents"`
		IndexMetrics    string `toml:"indexMetrics" commented:"true" comment:"index to store CDS metrics" json:"indexMetrics"`
		IndexJobSummary string `toml:"indexJobSummary" commented:"true" comment:"index to store CDS jobs summaries" json:"indexJobSummary"`
	} `toml:"elasticsearch" comment:"######################\n CDS ElasticSearch Settings \nSupport for elasticsearch 5.6\n######################" json:"elasticsearch"`
	EventBus struct {
		JobSummaryKafka event.KafkaConsumerConfig `toml:"jobSummaryKafka" json:"jobSummaryKafka" commented:"true" mapstructure:"jobSummaryKafka"`
	} `toml:"events" json:"events" commented:"true" mapstructure:"events"`
	API service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS Indexes Settings \n######################" json:"api"`
}
