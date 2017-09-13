package hooks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
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

	switch sConfig.Cache.Mode {
	case "local", "redis":
	default:
		return fmt.Errorf("Invalid cache mode")
	}

	return nil
}

// Serve will start the http api server
func (s *Service) Serve(ctx context.Context) error {
	if s.Cfg.Name == "" {
		s.Cfg.Name = hatchery.GenerateName("hooks", "")
	}

	log.Info("Hooks> Starting service %s...", s.Cfg.Name)

	//Instanciate a cds client
	s.cds = cdsclient.NewService(s.Cfg.API.HTTP.URL)

	//First register(heartbeat)
	if err := s.doHeartbeat(); err != nil {
		log.Error("Hooks> Unable to register: %v", err)
		return err
	}

	//Init the cache
	var errCache error
	s.Cache, errCache = cache.New(s.Cfg.Cache.Mode, s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password, s.Cfg.Cache.TTL)
	if errCache != nil {
		return errCache
	}

	//Start the heartbeat gorourine
	go func() {
		if err := s.heartbeat(ctx); err != nil {
			log.Error("%v", err)
		}
	}()

	//Start all the tasks
	go func() {
		if err := s.runTasks(ctx); err != nil {
			log.Error("%v", err)
		}
	}()

	s.initRouter(ctx)

	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", s.Cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		select {
		case <-ctx.Done():
			log.Info("Hooks> Shutdown HTTP Server")
			server.Shutdown(ctx)
		}
	}()

	log.Info("Hooks> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Hooks> Cannot start cds-hooks: %s", err)
	}
	return ctx.Err()
}

func (s *Service) initRouter(ctx context.Context) {
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.Handle("/webhook/{uuid}", r.POST(s.webhookHandler), r.GET(s.webhookHandler), r.DELETE(s.webhookHandler), r.PUT(s.webhookHandler))
}

func (s *Service) webhookHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return nil
	}
}
