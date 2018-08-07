package hooks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
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

// ApplyConfiguration apply an object of type hooks.Configuration after checking it
func (s *Service) ApplyConfiguration(config interface{}) error {
	if err := s.CheckConfiguration(config); err != nil {
		return err
	}
	var ok bool
	s.Cfg, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	s.Client = cdsclient.NewService(s.Cfg.API.HTTP.URL, 60*time.Second)
	s.API = s.Cfg.API.HTTP.URL
	s.Name = s.Cfg.Name
	s.HTTPURL = s.Cfg.URL
	s.Token = s.Cfg.API.Token
	s.Type = services.TypeHooks
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures
	s.ServiceName = "cds-hooks"

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (s *Service) CheckConfiguration(config interface{}) error {
	sConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	if sConfig.URL == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}
	if sConfig.Name == "" {
		return fmt.Errorf("please enter a name in your hooks configuration")
	}

	return nil
}

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	//Init the cache
	var errCache error
	s.Cache, errCache = cache.New(s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password, s.Cfg.Cache.TTL)
	if errCache != nil {
		return fmt.Errorf("Cannot connect to redis instance : %v", errCache)
	}

	//Init the DAO
	s.Dao = dao{s.Cache}

	if !s.Cfg.Disable {
		//Start all the tasks
		go func() {
			if err := s.runTasks(ctx); err != nil {
				log.Error("%v", err)
				cancel()
			}
		}()

		//Start the scheduler to execute all the tasks
		go func() {
			if err := s.runScheduler(ctx); err != nil {
				log.Error("%v", err)
				cancel()
			}
		}()
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
			log.Info("Hooks> Shutdown HTTP Server")
			server.Shutdown(ctx)
		}
	}()

	//Start the http server
	log.Info("Hooks> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Hooks> Cannot start cds-hooks: %s", err)
	}

	return ctx.Err()
}
