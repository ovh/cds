package cdn

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/lru"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/cds"
	_ "github.com/ovh/cds/engine/cdn/storage/local"
	_ "github.com/ovh/cds/engine/cdn/storage/redis"
	_ "github.com/ovh/cds/engine/cdn/storage/swift"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// Default size for LRU cache is 128Mo
const defaultLruSize = 128 * 1024 * 1024

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
		return cfg, sdk.WithStack(fmt.Errorf("invalid CDN service configuration"))
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

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	var err error

	if s.Cfg.EnableLogProcessing {
		log.Info(ctx, "Initializing database connection...")
		// Intialize database
		s.DBConnectionFactory, err = database.Init(
			ctx,
			s.Cfg.Database.User,
			s.Cfg.Database.Role,
			s.Cfg.Database.Password,
			s.Cfg.Database.Name,
			s.Cfg.Database.Schema,
			s.Cfg.Database.Host,
			s.Cfg.Database.Port,
			s.Cfg.Database.SSLMode,
			s.Cfg.Database.ConnectTimeout,
			s.Cfg.Database.Timeout,
			s.Cfg.Database.MaxConn)
		if err != nil {
			return fmt.Errorf("cannot connect to database: %v", err)
		}

		log.Info(ctx, "Setting up database keys...")
		s.Mapper = gorpmapper.New()
		encryptionKeyConfig := s.Cfg.Database.EncryptionKey.GetKeys(gorpmapper.KeyEcnryptionIdentifier)
		signatureKeyConfig := s.Cfg.Database.SignatureKey.GetKeys(gorpmapper.KeySignIdentifier)
		if err := s.Mapper.ConfigureKeys(&signatureKeyConfig, &encryptionKeyConfig); err != nil {
			return fmt.Errorf("cannot setup database keys: %v", err)
		}

		// Init dao packages
		item.InitDBMapping(s.Mapper)
		storage.InitDBMapping(s.Mapper)

		// Init storage units
		s.Units, err = storage.Init(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.GoRoutines, s.Cfg.Units, s.Cfg.Log)
		if err != nil {
			return err
		}

		s.Units.Start(ctx, s.GoRoutines)

		s.GoRoutines.Run(ctx, "service.cdn-gc-items", func(ctx context.Context) {
			s.itemsGC(ctx)
		})
		s.GoRoutines.Run(ctx, "service.cdn-purge-items", func(ctx context.Context) {
			s.itemPurge(ctx)
		})

		// Start CDS Backend migration
		for _, st := range s.Units.Storages {
			cdsStorage, ok := st.(*cds.CDS)
			if !ok {
				continue
			}
			s.GoRoutines.Exec(ctx, "cdn-cds-backend-migration", func(ctx context.Context) {
				if err := s.SyncLogs(ctx, cdsStorage); err != nil {
					log.Error(ctx, "unable to sync logs: %v", err)
				}
			})
			break
		}

		log.Info(ctx, "Initializing log cache on %s", s.Cfg.Cache.Redis.Host)
		lruSize := s.Cfg.Cache.LruSize
		if lruSize == 0 {
			lruSize = defaultLruSize
		}
		s.LogCache, err = lru.NewRedisLRU(s.mustDBWithCtx(ctx), lruSize, s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password)
		if err != nil {
			return sdk.WrapError(err, "cannot connect to redis instance for lru")
		}
		s.GoRoutines.Run(ctx, "service.log-cache-eviction", func(ctx context.Context) {
			s.LogCache.Evict(ctx)
		})
	}

	log.Info(ctx, "Initializing redis cache on %s...", s.Cfg.Cache.Redis.Host)
	s.Cache, err = cache.New(s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password, s.Cfg.Cache.TTL)
	if err != nil {
		return fmt.Errorf("cannot connect to redis instance : %v", err)
	}

	if err := s.initMetrics(ctx); err != nil {
		return err
	}

	s.runTCPLogServer(ctx)

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
		log.Fatalf("CDN> Cannot start cds-cdn: %v", err)
	}
	return ctx.Err()
}

func (s *Service) mustDBWithCtx(ctx context.Context) *gorp.DbMap {
	db := s.DBConnectionFactory.GetDBMap(s.Mapper)()
	db = db.WithContext(ctx).(*gorp.DbMap)
	if db == nil {
		panic(fmt.Errorf("database unavailable"))
	}
	return db
}
