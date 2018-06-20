package elasticsearch

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/olivere/elastic.v5"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

var esClient *elastic.Client

// New returns a new service
func New() *Service {
	s := new(Service)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
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

	s.Client = cdsclient.NewService(s.Cfg.API.HTTP.URL, 60*time.Second)
	s.API = s.Cfg.API.HTTP.URL
	s.Name = s.Cfg.Name
	s.HTTPURL = s.Cfg.URL
	s.Token = s.Cfg.API.Token
	s.Type = services.TypeElasticsearch
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

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	// Init es client
	var errClient error
	esClient, errClient = s.initClient()
	if errClient != nil {
		return sdk.WrapError(errClient, "Unable to create elasticsearchclient")
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
			log.Info("ElasticSearch> Shutdown HTTP Server")
			_ = server.Shutdown(ctx)
		}
	}()

	//Start the http server
	log.Info("ElasticSearch> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error("ElasticSearch> Listen and serve failed: %s", err)
	}

	return ctx.Err()
}

func (s *Service) initClient() (*elastic.Client, error) {
	return elastic.NewClient(
		elastic.SetURL(s.Cfg.ElasticSearch.URL),
		elastic.SetBasicAuth(s.Cfg.ElasticSearch.Username, s.Cfg.ElasticSearch.Password),
		elastic.SetSniff(false),
	)
}
