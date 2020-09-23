package repositories

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.GoRoutines = sdk.NewGoRoutines()
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

func (s *Service) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(Configuration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid repositories service configuration"))
	}

	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

// ApplyConfiguration apply an object of type repositories.Configuration after checking it
func (s *Service) ApplyConfiguration(config interface{}) error {
	if err := s.CheckConfiguration(config); err != nil {
		return err
	}
	var ok bool
	s.Cfg, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid Repositories configuration")
	}

	s.ServiceName = s.Cfg.Name
	s.ServiceType = sdk.TypeRepositories
	s.HTTPURL = s.Cfg.URL
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (s *Service) CheckConfiguration(config interface{}) error {
	sConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid Repositories configuration")
	}

	if sConfig.URL == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}
	if sConfig.Name == "" {
		return fmt.Errorf("please enter a name in your repositories configuration")
	}

	return nil
}

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	//Init the cache
	log.Info(ctx, "Initializing Redis connection (%s)...", s.Cfg.Cache.Redis.Host)
	var errCache error
	s.Cache, errCache = cache.New(s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password, s.Cfg.Cache.TTL)
	if errCache != nil {
		return fmt.Errorf("cannot connect to redis instance : %v", errCache)
	}

	var address = fmt.Sprintf("%s:%d", s.Cfg.HTTP.Addr, s.Cfg.HTTP.Port)
	log.Info(ctx, "Initializing HTTP router (%s)...", address)
	//Init the http server
	s.initRouter(ctx)
	server := &http.Server{
		Addr:           address,
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	//Set the dao
	s.dao = dao{
		store: s.Cache,
	}

	log.Info(ctx, "Initializing processor...")
	go func() {
		if err := s.processor(ctx); err != nil {
			log.Info(ctx, "Shutdown processor")
		}
	}()

	log.Info(ctx, "Initializing vacuumCleaner...")
	go func() {
		if err := s.vacuumCleaner(ctx); err != nil {
			log.Info(ctx, "Shutdown vacuumCleaner")
		}
	}()

	//Gracefully shutdown the http server
	go func() {
		<-ctx.Done()
		log.Info(ctx, "Shutdown HTTP Server")
		_ = server.Shutdown(ctx)
	}()

	//Start the http server
	log.Info(ctx, "Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error(ctx, "Listen and serve failed: %s", err)
	}

	return ctx.Err()
}
