package metricsservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk/cdsclient"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/metricsservice/elasticsearch"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	s.metricProviders = make(map[string]MetricProvider)
	s.StartupTime = time.Now()
	return s
}

func (s *Service) getProvider(name string) (MetricProvider, error) {
	log.Debug("Metrics> getProvider")
	providerCfg, has := s.Cfg.Providers[name]
	if !has {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	if providerCfg.ElasticSearch != nil {
		return elasticsearch.New(
			providerCfg.Endpoint,
			providerCfg.ElasticSearch.Username,
			providerCfg.ElasticSearch.Password,
			false,
			providerCfg.ElasticSearch.IndexEvents,
			providerCfg.ElasticSearch.IndexMetrics,
		)
	}
	return nil, nil
}

// ApplyConfiguration apply an object of type Metrics.Configuration after checking it
func (s *Service) ApplyConfiguration(config interface{}) error {
	if err := s.CheckConfiguration(config); err != nil {
		return err
	}
	var ok bool
	s.Cfg, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("invalid Metrics configuration")
	}

	s.Client = cdsclient.NewService(s.Cfg.API.HTTP.URL, 60*time.Second, s.Cfg.API.HTTP.Insecure)
	s.API = s.Cfg.API.HTTP.URL
	s.Name = s.Cfg.Name
	s.HTTPURL = s.Cfg.URL
	s.Token = s.Cfg.API.Token
	s.Type = services.TypeMetrics
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures
	s.ServiceName = "cds-metrics"

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (s *Service) CheckConfiguration(config interface{}) error {
	sConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("invalid Metrics configuration")
	}

	if sConfig.URL == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}
	if sConfig.Name == "" {
		return fmt.Errorf("please enter a name in your Metrics configuration")
	}

	return nil
}

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	log.Info("Metrics> Starting service %s %s...", s.Cfg.Name, sdk.VERSION)

	//init the metrics providers
	err := s.initMetrics(c)
	if err != nil {
		return sdk.WrapError(err, "failed to init metric providers")
	}

	//Init the http server
	s.initRouter(c)
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.Cfg.HTTP.Addr, s.Cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	//Start the http server
	log.Info("Metrics> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error("Metrics> Listen and serve failed: %s", err)
	}

	//Gracefully shutdown the http server
	go func() {
		select {
		case <-c.Done():
			log.Info("Metrics> Shutdown HTTP Server")
			server.Shutdown(c)
		}
	}()

	return c.Err()
}

func (s *Service) initMetrics(c context.Context) error {
	if len(s.Cfg.Providers) == 0 {
		return fmt.Errorf("no providers configured")
	}
	for name := range s.Cfg.Providers {
		log.Debug("Metrics> Setting up provider : %s", name)
		p, e := s.getProvider(name)
		if e != nil {
			return sdk.WrapError(e, "failed to add a metrics provider")
		}
		s.metricProviders[name] = p
	}
	return nil
}
