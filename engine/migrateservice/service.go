package migrateservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

type dbmigservice struct {
	service.Common
	cfg           Configuration
	Router        *api.Router
	currentStatus struct {
		err        error
		migrations []sdk.DatabaseMigrationStatus
	}
}

var _ service.BeforeStart = new(dbmigservice)

// Configuration is the exposed type for database API configuration
type Configuration struct {
	Name      string `toml:"name" comment:"Name of this CDS Database Migrate service\n Enter a name to enable this service" json:"name"`
	URL       string `default:"http://localhost:8087" json:"url"`
	Directory string `toml:"directory" comment:"SQL Migration files directory" default:"sql" json:"directory"`
	HTTP      struct {
		Addr     string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port     int    `toml:"port" default:"8087" json:"port"`
		Insecure bool   `toml:"insecure" default:"false" commented:"true" comment:"sslInsecureSkipVerify, set to true if you use a self-signed SSL on CDS API" json:"insecure"`
	} `toml:"http" comment:"#####################################\n CDS DB Migrate HTTP configuration \n####################################" json:"http"`
	API        service.APIServiceConfiguration `toml:"api" comment:"####################\n CDS API Settings \n###################" json:"api"`
	ServiceAPI struct {
		Enable bool                     `toml:"enable" default:"true" comment:"set to false to disable migration for API database" json:"enable"`
		DB     database.DBConfiguration `toml:"db" comment:"################################\n Postgresql Database settings \n###############################" json:"db"`
	} `toml:"serviceAPI" comment:"################################################\n CDS DB Migrate configuration for API service \n######################111######################" json:"service_api"`
	ServiceCDN struct {
		Enable bool                     `toml:"enable" default:"true" comment:"set to false to disable migration for CDN database" json:"enable"`
		DB     database.DBConfiguration `toml:"db" comment:"################################\n Postgresql Database settings \n###############################" json:"db"`
	} `toml:"serviceCDN" comment:"################################################\n CDS DB Migrate configuration for CDN service \n###############################################" json:"service_cdn"`
}

// New instanciates a new API object
func New() service.Service {
	s := &dbmigservice{}
	s.GoRoutines = sdk.NewGoRoutines()
	return s
}

func (s *dbmigservice) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(Configuration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid migrate service configuration"))
	}

	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

func (s *dbmigservice) CheckConfiguration(cfg interface{}) error {
	_, ok := cfg.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}
	return nil
}

func (s *dbmigservice) ApplyConfiguration(cfg interface{}) error {
	if err := s.CheckConfiguration(cfg); err != nil {
		return err
	}

	dbCfg, _ := cfg.(Configuration)

	s.cfg = dbCfg
	s.ServiceName = s.cfg.Name
	s.ServiceType = sdk.TypeDBMigrate
	s.HTTPURL = s.cfg.URL

	s.MaxHeartbeatFailures = s.cfg.API.MaxHeartbeatFailures
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return nil
}

func (s *dbmigservice) BeforeStart(ctx context.Context) error {
	status, err := doMigrateAll(ctx, s.cfg)
	if err != nil {
		log.Error(ctx, "DBMigrate> Migration failed %v", err)
		s.currentStatus.err = err
	}
	s.currentStatus.migrations = status

	// From now the database access won't be used. Erase the configuration...
	// This limits the attack surface
	s.cfg.ServiceAPI.DB = database.DBConfiguration{}
	s.cfg.ServiceCDN.DB = database.DBConfiguration{}
	return nil
}

func (s *dbmigservice) Serve(ctx context.Context) error {
	log.Info(ctx, "DBMigrate> Starting service %s %s...", s.cfg.Name, sdk.VERSION)
	s.StartupTime = time.Now()

	//Init the http server
	s.initRouter(ctx)
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.cfg.HTTP.Addr, s.cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		//Start the http server
		log.Info(ctx, "DBMigrate> Starting HTTP Server on port %d", s.cfg.HTTP.Port)
		if err := server.ListenAndServe(); err != nil {
			log.Error(ctx, "DBMigrate> Listen and serve failed: %s", err)
		}
	}()

	//Gracefully shutdown the http server
	<-ctx.Done()
	log.Info(ctx, "DBMigrate> Shutdown HTTP Server")
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("unable to shutdown server: %v", err)
	}

	return ctx.Err()
}

func (s *dbmigservice) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := s.NewMonitoringStatus()
	if s.currentStatus.err != nil {
		m.AddLine(
			sdk.MonitoringStatusLine{
				Component: "SQL",
				Value:     s.currentStatus.err.Error(),
				Status:    sdk.MonitoringStatusAlert,
			},
		)
		return m
	}

	var theNumberOfSuccessfulMigations int
	for _, m := range s.currentStatus.migrations {
		if m.Migrated {
			theNumberOfSuccessfulMigations++
		}
	}

	var status = sdk.MonitoringStatusWarn
	if theNumberOfSuccessfulMigations == len(s.currentStatus.migrations) {
		status = sdk.MonitoringStatusOK
	}

	m.AddLine(
		sdk.MonitoringStatusLine{
			Component: "SQL",
			Value:     fmt.Sprintf("%d/%d", theNumberOfSuccessfulMigations, len(s.currentStatus.migrations)),
			Status:    status,
		},
	)

	return m
}

func (s *dbmigservice) initRouter(ctx context.Context) {
	log.Debug("DBMigrate> Router initialized")
	r := s.Router
	r.SetHeaderFunc = service.DefaultHeaders
	r.DefaultAuthMiddleware = service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey)

	r.Handle("/mon/version", nil, r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/", nil, r.GET(s.getMigrationHandler, service.OverrideAuth(service.NoAuthMiddleware)))
}
