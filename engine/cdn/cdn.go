package cdn

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/lru"
	"github.com/ovh/cds/engine/cdn/storage"
	_ "github.com/ovh/cds/engine/cdn/storage/local"
	_ "github.com/ovh/cds/engine/cdn/storage/nfs"
	_ "github.com/ovh/cds/engine/cdn/storage/redis"
	_ "github.com/ovh/cds/engine/cdn/storage/s3"
	_ "github.com/ovh/cds/engine/cdn/storage/swift"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

const (
	defaultLruSize            = 128 * 1024 * 1024 // 128Mb
	defaultStepMaxSize        = 15 * 1024 * 1024  // 15Mb
	defaultStepLinesRateLimit = 1800
	defaultGlobalTCPRateLimit = 2 * 1024 * 1024 // 2Mb
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
		return cfg, sdk.WithStack(fmt.Errorf("invalid CDN service configuration"))
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

// ApplyConfiguration apply an object of type CDN.Configuration after checking it
func (s *Service) ApplyConfiguration(config interface{}) error {
	if err := s.CheckConfiguration(config); err != nil {
		return err
	}
	var ok bool
	s.Cfg, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("invalid configuration")
	}
	s.ServiceName = s.Cfg.Name
	s.ServiceType = sdk.TypeCDN
	s.HTTPURL = s.Cfg.URL
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures

	if s.Cfg.Cache.LruSize == 0 {
		s.Cfg.Cache.LruSize = defaultLruSize
	}
	if s.Cfg.Log.StepMaxSize == 0 {
		s.Cfg.Log.StepMaxSize = defaultStepMaxSize
	}
	if s.Cfg.Log.StepLinesRateLimit == 0 {
		s.Cfg.Log.StepLinesRateLimit = defaultStepLinesRateLimit
	}
	if s.Cfg.TCP.GlobalTCPRateLimit == 0 {
		s.Cfg.TCP.GlobalTCPRateLimit = defaultGlobalTCPRateLimit
	}
	if s.Cfg.Metrics.Frequency <= 0 {
		s.Cfg.Metrics.Frequency = 30
	}

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (s *Service) CheckConfiguration(config interface{}) error {
	sConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("invalid configuration")
	}

	if sConfig.URL == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}
	if sConfig.Name == "" {
		return fmt.Errorf("please enter a name in your CDN configuration")
	}

	return nil
}

func (s *Service) Start(ctx context.Context) error {
	if err := s.Common.Start(ctx); err != nil {
		return err
	}

	var err error
	log.Info(ctx, "Initializing redis cache on %s...", s.Cfg.Cache.Redis.Host)
	s.Cache, err = cache.New(s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password, s.Cfg.Cache.TTL)
	if err != nil {
		return sdk.WrapError(err, "cannot connect to redis instance")
	}

	log.Info(ctx, "Initializing database connection...")
	// Intialize database
	s.DBConnectionFactory, err = database.Init(ctx, s.Cfg.Database)
	if err != nil {
		return sdk.WrapError(err, "cannot connect to database")
	}

	log.Info(ctx, "Setting up database keys...")
	s.Mapper = gorpmapper.New()
	encryptionKeyConfig := s.Cfg.Database.EncryptionKey.GetKeys(gorpmapper.KeyEcnryptionIdentifier)
	signatureKeyConfig := s.Cfg.Database.SignatureKey.GetKeys(gorpmapper.KeySignIdentifier)
	if err := s.Mapper.ConfigureKeys(&signatureKeyConfig, &encryptionKeyConfig); err != nil {
		return sdk.WrapError(err, "cannot setup database keys")
	}

	// Init dao packages
	item.InitDBMapping(s.Mapper)
	storage.InitDBMapping(s.Mapper)

	log.Info(ctx, "Initializing lru connection...")
	s.LogCache, err = lru.NewRedisLRU(s.mustDBWithCtx(ctx), s.Cfg.Cache.LruSize, s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password)
	if err != nil {
		return sdk.WrapError(err, "cannot connect to redis instance for lru")
	}

	// Init storage units
	s.Units, err = storage.Init(ctx, s.Mapper, s.Cache, s.mustDBWithCtx(ctx), s.GoRoutines, s.Cfg.Units)
	if err != nil {
		log.Error(ctx, "unable to init storage unit: %v", err)
		return err
	}

	s.Units.Start(ctx, s.GoRoutines)

	s.GoRoutines.Run(ctx, "service.cdn-gc-items", func(ctx context.Context) {
		s.itemsGC(ctx)
	})
	s.GoRoutines.Run(ctx, "service.cdn-purge-items", func(ctx context.Context) {
		s.itemPurge(ctx)
	})

	s.GoRoutines.Run(ctx, "service.log-cache-eviction", func(ctx context.Context) {
		s.LogCache.Evict(ctx)
	})

	return nil
}

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	if err := s.initMetrics(ctx); err != nil {
		return err
	}

	if err := s.runTCPLogServer(ctx); err != nil {
		return err
	}

	log.Info(ctx, "Initializing HTTP router")
	s.initRouter(ctx)
	if err := s.initWebsocket(); err != nil {
		return err
	}
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.Cfg.HTTP.Addr, s.Cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		MaxHeaderBytes: 1 << 20,
	}

	// Gracefully shutdown the http server
	s.GoRoutines.Exec(ctx, "service.httpserver-shutdown", func(ctx context.Context) {
		<-ctx.Done()
		log.Info(ctx, "CDN> Shutdown HTTP Server")
		_ = server.Shutdown(ctx)
	})

	// Start the http server
	log.Info(ctx, "CDN> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error(ctx, "CDN> Cannot start cds-cdn: %v", err)
	}

	return ctx.Err()
}

func (s *Service) mustDB() *gorp.DbMap {
	db := s.DBConnectionFactory.GetDBMap(s.Mapper)()
	if db == nil {
		panic(fmt.Errorf("database unavailable"))
	}
	return db
}

func (s *Service) mustDBWithCtx(ctx context.Context) *gorp.DbMap {
	db := s.DBConnectionFactory.GetDBMap(s.Mapper)()
	db = db.WithContext(ctx).(*gorp.DbMap)
	if db == nil {
		panic(fmt.Errorf("database unavailable"))
	}
	return db
}
