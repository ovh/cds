package hooks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.GoRoutines = sdk.NewGoRoutines(context.Background())
	return s
}

func (s *Service) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(Configuration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid hooks service configuration"))
	}
	s.Router = &api.Router{
		Mux:    mux.NewRouter(),
		Config: sConfig.HTTP,
	}
	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
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

	s.ServiceName = s.Cfg.Name
	s.ServiceType = sdk.TypeHooks
	s.HTTPURL = s.Cfg.URL
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures

	if !sdk.IsURL(s.Cfg.URLPublic) {
		return fmt.Errorf("invalid hooks configuration, urlPublic configuration is mandatory")
	}

	if s.Cfg.RepositoryEventRetention == 0 {
		s.Cfg.RepositoryEventRetention = 30
	}
	if s.Cfg.OldRepositoryEventQueueLen == 0 {
		s.Cfg.OldRepositoryEventQueueLen = 200
	}
	if s.Cfg.OldRepositoryEventRetry == 0 {
		s.Cfg.OldRepositoryEventRetry = 1
	}
	if s.Cfg.OutgoingEventTTL == 0 {
		s.Cfg.OutgoingEventTTL = 7
	}
	if s.Cfg.AnalysisCheckMaxRetry == 0 {
		s.Cfg.AnalysisCheckMaxRetry = 30
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
	s.Cache, errCache = cache.New(s.Cfg.Cache.Redis, s.Cfg.Cache.TTL)
	if errCache != nil {
		return fmt.Errorf("cannot connect to redis instance : %v", errCache)
	}

	outgoingHookEventTTL := s.Cfg.OutgoingEventTTL * 3600 * 24

	//Init the DAO
	s.Dao = dao{store: s.Cache, outgoingHookEventTTL: outgoingHookEventTTL}

	// Get ui rul
	config, err := s.Client.ConfigUser()
	if err != nil {
		return err
	}
	s.UIURL = config.URLUI

	// Get current maintenance state
	var b bool
	if _, err := s.Dao.store.Get(MaintenanceHookKey, &b); err != nil {
		return fmt.Errorf("cannot get %s from redis: %v", MaintenanceHookKey, err)
	}
	s.Maintenance = b

	// Listen event on maintenance state
	go func() {
		if err := s.listenMaintenance(ctx); err != nil {
			log.Error(ctx, "error while initializing listen maintenance routine: %s", err)
		}
	}()

	if !s.Cfg.Disable {

		s.GoRoutines.RunWithRestart(ctx, "dequeueRepositoryEvent", func(ctx context.Context) {
			s.dequeueRepositoryEvent(ctx)
		})

		s.GoRoutines.RunWithRestart(ctx, "dequeueWorkflowRunOutgoingEvent", func(ctx context.Context) {
			s.dequeueWorkflowRunOutgoingEvent(ctx)
		})

		s.GoRoutines.RunWithRestart(ctx, "dequeueRepositoryEventCallback", func(ctx context.Context) {
			s.dequeueRepositoryEventCallback(ctx)
		})

		// Reenqueue old repository event
		if !s.Cfg.DisableRepositoryEventRetry {
			s.GoRoutines.RunWithRestart(ctx, "manageOldRepositoryEvent", func(ctx context.Context) {
				s.manageOldRepositoryEvent(ctx)
			})
			s.GoRoutines.RunWithRestart(ctx, "manageOldOUtgoingEvent", func(ctx context.Context) {
				s.manageOldWorkflowRunOutgoingEvent(ctx)
			})
		}

		// Delete old repository event
		s.GoRoutines.RunWithRestart(ctx, "cleanRepositoryEvent", func(ctx context.Context) {
			s.scheduleCleanOldRepositoryEvent(ctx)
		})

		//Start all the tasks
		go func() {
			if err := s.runTasks(ctx); err != nil {
				log.Error(ctx, "%v", err)
				cancel()
			}
		}()

		//Start the scheduler to execute all the tasks
		go func() {
			if err := s.runScheduler(ctx); err != nil {
				log.Error(ctx, "%v", err)
				cancel()
			}
		}()

		s.GoRoutines.RunWithRestart(ctx, "schedulerv2", func(ctx context.Context) {
			s.schedulerExecutionRoutine(ctx)
		})
	}

	if s.Cfg.WebhooksPublicKeySign != "" {
		webhookKey, err := jws.NewPublicKeyFromPEM([]byte(s.Cfg.WebhooksPublicKeySign))
		if err != nil {
			return sdk.WithStack(err)
		}
		s.WebHooksParsedPublicKey = webhookKey
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
			log.Info(ctx, "Hooks> Shutdown HTTP Server")
			server.Shutdown(ctx)
		}
	}()

	//Start the http server
	log.Info(ctx, "Hooks> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error(ctx, "Hooks> Cannot start cds-hooks: %s", err)
	}

	return ctx.Err()
}
