package migrateservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk/cdsclient"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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

// Configuration is the exposed type for database API configuration
type Configuration struct {
	Name      string `toml:"name" comment:"Name of this CDS Database Migrate service\n Enter a name to enable this service" json:"name"`
	URL       string `default:"http://localhost:8087" json:"url"`
	Directory string `toml:"directory" comment:"SQL Migration files directory" default:"sql" json:"directory"`
	HTTP      struct {
		Addr     string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port     int    `toml:"port" default:"8087" json:"port"`
		Insecure bool   `toml:"insecure" default:"false" commented:"true" comment:"sslInsecureSkipVerify, set to true if you use a self-signed SSL on CDS API" json:"insecure"`
	} `toml:"http" comment:"######################\n CDS DB Migrate HTTP Configuration \n######################" json:"http"`
	API service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	DB  database.DBConfiguration        `toml:"db" comment:"################################\n Postgresql Database settings \n###############################" json:"db"`
}

// New instanciates a new API object
func New() service.Service {
	return &dbmigservice{}
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
	log.Debug("%+v", s.cfg)
	s.Name = s.cfg.Name
	s.HTTPURL = s.cfg.URL

	s.Type = services.TypeDBMigrate
	s.ServiceName = "cds-migrate"
	s.MaxHeartbeatFailures = s.cfg.API.MaxHeartbeatFailures
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return nil
}

func (s *dbmigservice) BeforeStart() error {
	log.Info("DBMigrate> Starting Database migration...")
	errDo := s.doMigrate()
	if errDo != nil {
		log.Error("DBMigrate> Migration failed %v", errDo)
		s.currentStatus.err = errDo
	}

	log.Info("DBMigrate> Retrieving Database migration status...")
	status, errGet := s.getMigrate()
	if errGet != nil {
		log.Error("DBMigrate> Migration status unavailable %v", errGet)
	}
	if errDo == nil && errGet != nil {
		s.currentStatus.err = errGet
	}
	s.currentStatus.migrations = status

	// From now the database access won't be used. Erase the configuration...
	// This limits the attack surface
	s.cfg.DB = database.DBConfiguration{}

	return nil
}

func (s *dbmigservice) Serve(c context.Context) error {
	log.Info("DBMigrate> Starting service %s %s...", s.cfg.Name, sdk.VERSION)
	s.StartupTime = time.Now()

	//Init the http server
	s.initRouter(c)
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.cfg.HTTP.Addr, s.cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		//Start the http server
		log.Info("DBMigrate> Starting HTTP Server on port %d", s.cfg.HTTP.Port)
		if err := server.ListenAndServe(); err != nil {
			log.Error("DBMigrate> Listen and serve failed: %s", err)
		}
	}()

	//Gracefully shutdown the http server
	<-c.Done()
	log.Info("DBMigrate> Shutdown HTTP Server")
	if err := server.Shutdown(c); err != nil {
		return fmt.Errorf("unable to shutdown server: %v", err)
	}

	return c.Err()
}

func (s *dbmigservice) Status() sdk.MonitoringStatus {
	response := s.CommonMonitoring()
	if s.currentStatus.err != nil {
		response.Lines = append(response.Lines,
			sdk.MonitoringStatusLine{
				Component: "SQL",
				Value:     s.currentStatus.err.Error(),
				Status:    sdk.MonitoringStatusAlert,
			},
		)
		return response
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

	response.Lines = append(response.Lines,
		sdk.MonitoringStatusLine{
			Component: "SQL",
			Value:     fmt.Sprintf("%d/%d", theNumberOfSuccessfulMigations, len(s.currentStatus.migrations)),
			Status:    status,
		},
	)

	return response
}

func (s *dbmigservice) initRouter(ctx context.Context) {
	log.Debug("DBMigrate> Router initialized")
	r := s.Router
	r.SetHeaderFunc = api.DefaultHeaders
	r.Middlewares = append(r.Middlewares, service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey))

	r.Handle("/mon/version", nil, r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, api.Auth(false)))
	r.Handle("/", nil, r.GET(s.getMigrationHandler, api.Auth(false)))
}
