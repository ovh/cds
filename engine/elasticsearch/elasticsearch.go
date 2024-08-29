package elasticsearch

import (
	"context"
	"fmt"
	"net/http"
	"time"

	elastic "github.com/elastic/go-elasticsearch/v8"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.GoRoutines = sdk.NewGoRoutines(context.Background())
	return s
}

// ApplyConfiguration apply an object of type elasticsearch.Configuration after checking it
func (s *Service) ApplyConfiguration(config interface{}) error {
	if err := s.CheckConfiguration(config); err != nil {
		return err
	}
	var ok bool
	s.Cfg, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("ApplyConfiguration> Invalid Elasticsearch configuration")
	}
	s.Router = &api.Router{
		Mux:    mux.NewRouter(),
		Config: s.Cfg.HTTP,
	}
	s.HTTPURL = s.Cfg.URL
	s.ServiceName = s.Cfg.Name
	s.ServiceType = sdk.TypeElasticsearch
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (s *Service) CheckConfiguration(config interface{}) error {
	sConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("CheckConfiguration> Invalid Elasticsearch configuration")
	}

	if sConfig.URL == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}
	if sConfig.Name == "" {
		return fmt.Errorf("please enter a name in your Elasticsearch configuration")
	}

	return nil
}

func (s *Service) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(Configuration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid Elasticsearch configuration"))
	}

	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	// Init es client
	var err error
	s.esClient, err = s.initClient()
	if err != nil {
		return sdk.WrapError(err, "Unable to create elasticsearch client")
	}

	//Init the http server
	s.initRouter(ctx)
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.Cfg.HTTP.Addr, s.Cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	//Gracefully shutdown the http server
	go func() {
		select {
		case <-ctx.Done():
			log.Info(ctx, "ElasticSearch> Shutdown HTTP Server")
			_ = server.Shutdown(ctx)
		}
	}()

	if s.Cfg.EventBus.JobSummaryKafka.BrokerAddresses != "" {
		if err := s.consumeKafka(ctx, s.Cfg.EventBus.JobSummaryKafka); err != nil {
			log.Error(ctx, "unable to start kafka consumer: %v", err)
		}
	}

	//Start the http server
	log.Info(ctx, "ElasticSearch> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error(ctx, "ElasticSearch> Listen and serve failed: %v", err)
	}

	return ctx.Err()
}

func (s *Service) initClient() (ESClient, error) {
	x, err := elastic.NewTypedClient(elastic.Config{
		Addresses: []string{s.Cfg.ElasticSearch.URL},
		Username:  s.Cfg.ElasticSearch.Username,
		Password:  s.Cfg.ElasticSearch.Password,
	})
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	var c = esClient{
		client: x,
	}
	return &c, nil
}
