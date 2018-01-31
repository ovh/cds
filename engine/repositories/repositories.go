package repositories

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
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

	log.Info("Repositories> Starting service %s %s...", s.Cfg.Name, sdk.VERSION)

	//Instanciate a cds client
	s.cds = cdsclient.NewService(s.Cfg.API.HTTP.URL, 60*time.Second)

	//First register(heartbeat)
	if err := s.doHeartbeat(); err != nil {
		log.Error("Repositories> Unable to register: %v", err)
		return err
	}
	log.Info("Repositories> Service registered")

	//Start the heartbeat gorourine
	go func() {
		if err := s.heartbeat(ctx); err != nil {
			log.Error("%v", err)
			cancel()
		}
	}()

	//Init the cache
	var errCache error
	s.Cache, errCache = cache.New(s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password, s.Cfg.Cache.TTL)
	if errCache != nil {
		return errCache
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

	//Set the dao
	s.dao = dao{
		store: s.Cache,
	}

	go func() {
		if err := s.processor(ctx); err != nil {
			log.Info("Repositories> Shutdown processor")
		}
	}()

	//Gracefully shutdown the http server
	go func() {
		select {
		case <-ctx.Done():
			log.Info("Repositories> Shutdown HTTP Server")
			server.Shutdown(ctx)
		}
	}()

	//Start the http server
	log.Info("Repositories> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error("Repositories> Listen and serve failed: %s", err)
	}

	return ctx.Err()
}
