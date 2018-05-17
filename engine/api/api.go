package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/queue"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/warning"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Configuration is the configuraton structure for CDS API
type Configuration struct {
	Name string `toml:"name" default:"cdsinstance" comment:"Name of this CDS Instance"`
	URL  struct {
		API string `toml:"api" default:"http://localhost:8081"`
		UI  string `toml:"ui" default:"http://localhost:2015"`
	} `toml:"url" comment:"#####################\n CDS URLs Settings \n####################"`
	HTTP struct {
		Addr       string `toml:"addr" default:"" commented:"true" comment:"Listen HTTP address without port, example: 127.0.0.1"`
		Port       int    `toml:"port" default:"8081"`
		SessionTTL int    `toml:"sessionTTL" default:"60"`
	} `toml:"http"`
	GRPC struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen GRPC address without port, example: 127.0.0.1"`
		Port int    `toml:"port" default:"8082"`
	} `toml:"grpc"`
	Secrets struct {
		Key string `toml:"key"`
	} `toml:"secrets"`
	Database struct {
		User           string `toml:"user" default:"cds"`
		Password       string `toml:"password" default:"cds"`
		Name           string `toml:"name" default:"cds"`
		Host           string `toml:"host" default:"localhost"`
		Port           int    `toml:"port" default:"5432"`
		SSLMode        string `toml:"sslmode" default:"disable" comment:"DB SSL Mode: require (default), verify-full, or disable"`
		MaxConn        int    `toml:"maxconn" default:"20" comment:"DB Max connection"`
		ConnectTimeout int    `toml:"connectTimeout" default:"10" comment:"Maximum wait for connection, in seconds"`
		Timeout        int    `toml:"timeout" default:"3000" comment:"Statement timeout value in milliseconds"`
	} `toml:"database" comment:"################################\n Postgresql Database settings \n###############################"`
	Cache struct {
		TTL   int `toml:"ttl" default:"60"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax! <clustername>@sentinel1:26379,sentinel2:26379,sentinel3:26379"`
			Password string `toml:"password"`
		} `toml:"redis" comment:"Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup"`
	} `toml:"cache" comment:"######################\n CDS Cache Settings \n#####################\n"`
	Directories struct {
		Download string `toml:"download" default:"/tmp/cds/download"`
		Keys     string `toml:"keys" default:"/tmp/cds/keys"`
	} `toml:"directories"`
	Auth struct {
		DefaultGroup     string `toml:"defaultGroup" default:"" comment:"The default group is the group in which every new user will be granted at signup"`
		SharedInfraToken string `toml:"sharedInfraToken" default:"" comment:"Token for shared.infra group. This value will be used when shared.infra will be created\nat first CDS launch. This token can be used by CDS CLI, Hatchery, etc...\nThis is mandatory."`
		LDAP             struct {
			Enable   bool   `toml:"enable" default:"false"`
			Host     string `toml:"host"`
			Port     int    `toml:"port" default:"636"`
			SSL      bool   `toml:"ssl" default:"true"`
			Base     string `toml:"base" default:"dc=myorganization,dc=com"`
			DN       string `toml:"dn" default:"uid=%s,ou=people,dc=myorganization,dc=com"`
			Fullname string `toml:"fullname" default:"{{.givenName}} {{.sn}}"`
			BindDN   string `toml:"bindDN" default:"" comment:"Define it if ldapsearch need to be authenticated"`
			BindPwd  string `toml:"bindPwd" default:"" comment:"Define it if ldapsearch need to be authenticated"`
		} `toml:"ldap"`
		Local struct {
			SignupAllowedDomains string `toml:"signupAllowedDomains" default:"" comment:"Allow signup from selected domains only - comma separated. Example: your-domain.com,another-domain.com" commented:"true"`
		} `toml:"local"`
	} `toml:"auth" comment:"##############################\n CDS Authentication Settings#\n#############################"`
	SMTP struct {
		Disable  bool   `toml:"disable" default:"true"`
		Host     string `toml:"host"`
		Port     string `toml:"port"`
		TLS      bool   `toml:"tls"`
		User     string `toml:"user"`
		Password string `toml:"password"`
		From     string `toml:"from" default:"no-reply@cds.local"`
	} `toml:"smtp" comment:"#####################\n# CDS SMTP Settings \n####################"`
	Artifact struct {
		Mode  string `toml:"mode" default:"local" comment:"swift or local"`
		Local struct {
			BaseDirectory string `toml:"baseDirectory" default:"/tmp/cds/artifacts"`
		} `toml:"local"`
		Openstack struct {
			URL             string `toml:"url" comment:"Authentication Endpoint, generally value of $OS_AUTH_URL"`
			Username        string `toml:"username" comment:"Openstack Username, generally value of $OS_USERNAME"`
			Password        string `toml:"password" comment:"Openstack Password, generally value of $OS_PASSWORD"`
			Tenant          string `toml:"tenant" comment:"Openstack Tenant, generally value of $OS_TENANT_NAME, v2 auth only"`
			Domain          string `toml:"domain" comment:"Openstack Domain, generally value of $OS_DOMAIN_NAME, v3 auth only"`
			Region          string `toml:"region" comment:"Region, generally value of $OS_REGION_NAME"`
			ContainerPrefix string `toml:"containerPrefix" comment:"Use if your want to prefix containers for CDS Artifacts"`
			DisableTempURL  bool   `toml:"disableTempURL" default:"false" commented:"true" comment:"True if you want to disable Temporary URL in file upload"`
		} `toml:"openstack"`
	} `toml:"artifact" comment:"Either filesystem local storage or Openstack Swift Storage are supported"`
	Events struct {
		Kafka struct {
			Enabled  bool   `toml:"enabled"`
			Broker   string `toml:"broker"`
			Topic    string `toml:"topic"`
			User     string `toml:"user"`
			Password string `toml:"password"`
		} `toml:"kafka"`
	} `toml:"events" comment:"#######################\n CDS Events Settings \n######################"`
	Features struct {
		Izanami struct {
			ApiURL       string `toml:"apiurl"`
			ClientID     string `toml:"clientid"`
			ClientSecret string `toml:"clientsecret"`
			Token        string `toml:"token" comment:"Token shared between Izanami and CDS to be able to send webhooks from izanami"`
		} `toml:"izanami" comment:"Feature flipping provider: https://maif.github.io/izanami"`
	} `toml:"features" comment:"###########################\n CDS Features flipping Settings \n##########################"`
	Schedulers struct {
		Disabled bool `toml:"disabled" default:"false" commented:"true" comment:"This is mainly for dev purpose, you should not have to change it"`
	} `toml:"schedulers" comment:"###########################\n CDS Schedulers Settings \n##########################"`
	Vault struct {
		ConfigurationKey string `toml:"configurationKey"`
	} `toml:"vault"`
	Providers []ProviderConfiguration `toml:"providers" comment:"###########################\n CDS Providers Settings \n##########################"`
}

type ProviderConfiguration struct {
	Name  string `toml:"name"`
	Token string `toml:"token"`
}

// DefaultValues is the struc for API Default configuration default values
type DefaultValues struct {
	ServerSecretsKey     string
	AuthSharedInfraToken string
	// For LDAP Client
	LDAPBase  string
	GivenName string
	SN        string
	BindDN    string
	BindPwd   string
}

// New instanciates a new API object
func New() *API {
	return &API{}
}

// Service returns an instance of sdk.Service for the API
func (*API) Service() sdk.Service {
	return sdk.Service{
		LastHeartbeat: time.Time{},
		Type:          services.TypeAPI,
	}
}

// API is a struct containing the configuration, the router, the database connection factory and so on
type API struct {
	service.Common
	Router              *Router
	Config              Configuration
	DBConnectionFactory *database.DBConnectionFactory
	StartupTime         time.Time
	lastUpdateBroker    *lastUpdateBroker
	eventsBroker        *eventsBroker
	warnChan            chan sdk.Event
	Cache               cache.Store
}

// ApplyConfiguration apply an object of type api.Configuration after checking it
func (a *API) ApplyConfiguration(config interface{}) error {
	if err := a.CheckConfiguration(config); err != nil {
		return err
	}

	var ok bool
	a.Config, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	a.Type = services.TypeAPI

	return nil
}

// DirectoryExists checks if the directory exists
func DirectoryExists(path string) (bool, error) {
	s, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return s.IsDir(), err
}

// CheckConfiguration checks the validity of the configuration object
func (a *API) CheckConfiguration(config interface{}) error {
	aConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid API configuration")
	}

	if aConfig.URL.API == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}

	if aConfig.Directories.Download == "" {
		return fmt.Errorf("Invalid download directory")
	}

	if ok, err := DirectoryExists(aConfig.Directories.Download); !ok {
		if err := os.MkdirAll(aConfig.Directories.Download, os.FileMode(0700)); err != nil {
			return fmt.Errorf("Unable to create directory %s: %v", aConfig.Directories.Download, err)
		}
		log.Info("Directory %s has been created", aConfig.Directories.Download)
	} else if err != nil {
		return fmt.Errorf("Invalid download directory: %v", err)
	}

	if aConfig.Directories.Keys == "" {
		return fmt.Errorf("Invalid keys directory")
	}

	if ok, err := DirectoryExists(aConfig.Directories.Keys); !ok {
		if err := os.MkdirAll(aConfig.Directories.Keys, os.FileMode(0700)); err != nil {
			return fmt.Errorf("Unable to create directory %s: %v", aConfig.Directories.Keys, err)
		}
		log.Info("Directory %s has been created", aConfig.Directories.Keys)
	} else if err != nil {
		return fmt.Errorf("Invalid keys directory: %v", err)
	}

	switch aConfig.Artifact.Mode {
	case "local", "openstack", "swift":
	default:
		return fmt.Errorf("Invalid artifact mode")
	}

	if aConfig.Artifact.Mode == "local" {
		if aConfig.Artifact.Local.BaseDirectory == "" {
			return fmt.Errorf("Invalid artifact local base directory")
		}
		if ok, err := DirectoryExists(aConfig.Artifact.Local.BaseDirectory); !ok {
			if err := os.MkdirAll(aConfig.Artifact.Local.BaseDirectory, os.FileMode(0700)); err != nil {
				return fmt.Errorf("Unable to create directory %s: %v", aConfig.Artifact.Local.BaseDirectory, err)
			}
			log.Info("Directory %s has been created", aConfig.Artifact.Local.BaseDirectory)
		} else if err != nil {
			return fmt.Errorf("Invalid artifact local base directory: %v", err)
		}
	}

	if len(aConfig.Secrets.Key) != 32 {
		return fmt.Errorf("Invalid secret key. It should be 32 bits (%d)", len(aConfig.Secrets.Key))
	}
	return nil
}

func getUserSession(c context.Context) string {
	i := c.Value(auth.ContextUserSession)
	if i == nil {
		return ""
	}
	u, ok := i.(string)
	if !ok {
		return ""
	}
	return u
}

func getUser(c context.Context) *sdk.User {
	i := c.Value(auth.ContextUser)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.User)
	if !ok {
		return nil
	}
	return u
}

func getProvider(c context.Context) *string {
	i := c.Value(auth.ContextProvider)
	if i == nil {
		return nil
	}
	u, ok := i.(string)
	if !ok {
		return nil
	}
	return &u
}

func getAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}

func isHatcheryOrWorker(r *http.Request) bool {
	switch getAgent(r) {
	case sdk.HatcheryAgent:
		return true
	case sdk.WorkerAgent:
		return true
	default:
		return false
	}
}

func getWorker(c context.Context) *sdk.Worker {
	i := c.Value(auth.ContextWorker)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.Worker)
	if !ok {
		return nil
	}
	return u
}

func getHatchery(c context.Context) *sdk.Hatchery {
	i := c.Value(auth.ContextHatchery)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.Hatchery)
	if !ok {
		return nil
	}
	return u
}

func getService(c context.Context) *sdk.Service {
	i := c.Value(auth.ContextService)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.Service)
	if !ok {
		return nil
	}
	return u
}

func (a *API) mustDB() *gorp.DbMap {
	db := a.DBConnectionFactory.GetDBMap()
	if db == nil {
		panic(fmt.Errorf("Database unavailable"))
	}
	return db
}

func (a *API) mustDBWithCtx(ctx context.Context) *gorp.DbMap {
	db := a.DBConnectionFactory.GetDBMap()
	db = db.WithContext(ctx).(*gorp.DbMap)
	if db == nil {
		panic(fmt.Errorf("Database unavailable"))
	}

	return db
}

// Serve will start the http api server
func (a *API) Serve(ctx context.Context) error {
	log.Info("Starting CDS API Server %s", sdk.VERSION)

	a.StartupTime = time.Now()

	//Initialize secret driver
	secret.Init(a.Config.Secrets.Key)

	//Initialize mail package
	log.Info("Initializing mail driver...")
	mail.Init(a.Config.SMTP.User,
		a.Config.SMTP.Password,
		a.Config.SMTP.From,
		a.Config.SMTP.Host,
		a.Config.SMTP.Port,
		a.Config.SMTP.TLS,
		a.Config.SMTP.Disable)

	// Initialize feature packages
	log.Info("Initializing feature flipping with izanami %s", a.Config.Features.Izanami.ApiURL)
	if a.Config.Features.Izanami.ApiURL != "" {
		if err := feature.Init(a.Config.Features.Izanami.ApiURL, a.Config.Features.Izanami.ClientID, a.Config.Features.Izanami.ClientSecret); err != nil {
			return fmt.Errorf("Feature flipping not enabled with izanami: %s", err)
		}
	}

	//Initialize artifacts storage
	log.Info("Initializing %s objectstore...", a.Config.Artifact.Mode)
	var objectstoreKind objectstore.Kind
	switch a.Config.Artifact.Mode {
	case "openstack":
		objectstoreKind = objectstore.Openstack
	case "swift":
		objectstoreKind = objectstore.Swift
	case "filesystem", "local":
		objectstoreKind = objectstore.Filesystem
	default:
		return fmt.Errorf("unsupported objecstore mode : %s", a.Config.Artifact.Mode)
	}

	cfg := objectstore.Config{
		Kind: objectstoreKind,
		Options: objectstore.ConfigOptions{
			Openstack: objectstore.ConfigOptionsOpenstack{
				Address:         a.Config.Artifact.Openstack.URL,
				Username:        a.Config.Artifact.Openstack.Username,
				Password:        a.Config.Artifact.Openstack.Password,
				Tenant:          a.Config.Artifact.Openstack.Tenant,
				Domain:          a.Config.Artifact.Openstack.Domain,
				Region:          a.Config.Artifact.Openstack.Region,
				ContainerPrefix: a.Config.Artifact.Openstack.ContainerPrefix,
				DisableTempURL:  a.Config.Artifact.Openstack.DisableTempURL,
			},
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: a.Config.Artifact.Local.BaseDirectory,
			},
		},
	}

	if err := objectstore.Initialize(ctx, cfg); err != nil {
		return fmt.Errorf("cannot initialize storage: %v", err)
	}

	log.Info("Initializing database connection...")
	//Intialize database
	var errDB error
	a.DBConnectionFactory, errDB = database.Init(
		a.Config.Database.User,
		a.Config.Database.Password,
		a.Config.Database.Name,
		a.Config.Database.Host,
		a.Config.Database.Port,
		a.Config.Database.SSLMode,
		a.Config.Database.ConnectTimeout,
		a.Config.Database.Timeout,
		a.Config.Database.MaxConn)
	if errDB != nil {
		return fmt.Errorf("cannot connect to database: %v", errDB)
	}

	log.Info("Bootstrapping database...")
	defaultValues := sdk.DefaultValues{
		DefaultGroupName: a.Config.Auth.DefaultGroup,
		SharedInfraToken: a.Config.Auth.SharedInfraToken,
	}
	if err := bootstrap.InitiliazeDB(defaultValues, a.DBConnectionFactory.GetDBMap); err != nil {
		return fmt.Errorf("cannot setup databases: %v", err)
	}

	if err := workflow.CreateBuiltinWorkflowHookModels(a.DBConnectionFactory.GetDBMap()); err != nil {
		return fmt.Errorf("cannot setup builtin workflow hook models: %v", err)
	}

	if err := platform.CreateBuiltinModels(a.DBConnectionFactory.GetDBMap()); err != nil {
		return fmt.Errorf("cannot setup platforms: %v", err)
	}

	log.Info("Initializing redis cache on %s...", a.Config.Cache.Redis.Host)
	//Init the cache
	var errCache error
	a.Cache, errCache = cache.New(
		a.Config.Cache.Redis.Host,
		a.Config.Cache.Redis.Password,
		a.Config.Cache.TTL)
	if errCache != nil {
		return fmt.Errorf("cannot connect to cache store: %v", errCache)
	}

	log.Info("Initializing HTTP router")
	a.Router = &Router{
		Mux:        mux.NewRouter(),
		Background: ctx,
	}
	a.InitRouter()

	//Init events package
	event.Cache = a.Cache

	//Initiliaze hook package
	hook.Init(a.Config.URL.API)

	//Intialize notification package
	notification.Init(a.Config.URL.UI)

	log.Info("Initializing Authentication driver...")
	// Initialize the auth driver
	var authMode string
	var authOptions interface{}
	switch a.Config.Auth.LDAP.Enable {
	case true:
		authMode = "ldap"
		authOptions = auth.LDAPConfig{
			Host:         a.Config.Auth.LDAP.Host,
			Port:         a.Config.Auth.LDAP.Port,
			Base:         a.Config.Auth.LDAP.Base,
			DN:           a.Config.Auth.LDAP.DN,
			SSL:          a.Config.Auth.LDAP.SSL,
			UserFullname: a.Config.Auth.LDAP.Fullname,
			BindDN:       a.Config.Auth.LDAP.BindDN,
			BindPwd:      a.Config.Auth.LDAP.BindPwd,
		}
	default:
		authMode = "local"
	}

	storeOptions := sessionstore.Options{
		TTL:   a.Config.HTTP.SessionTTL * 60, // Second to minutes
		Cache: a.Cache,
	}

	var errdriver error
	a.Router.AuthDriver, errdriver = auth.GetDriver(ctx, authMode, authOptions, storeOptions, a.DBConnectionFactory.GetDBMap)
	if errdriver != nil {
		return fmt.Errorf("error: %v", errdriver)
	}

	log.Info("Initializing event broker...")
	kafkaOptions := event.KafkaConfig{
		Enabled:         a.Config.Events.Kafka.Enabled,
		BrokerAddresses: a.Config.Events.Kafka.Broker,
		User:            a.Config.Events.Kafka.User,
		Password:        a.Config.Events.Kafka.Password,
		Topic:           a.Config.Events.Kafka.Topic,
	}
	if err := event.Initialize(kafkaOptions); err != nil {
		log.Error("error while initializing event system: %s", err)
	} else {
		go event.DequeueEvent(ctx)
	}

	if err := worker.Initialize(ctx, a.DBConnectionFactory.GetDBMap, a.Cache); err != nil {
		log.Error("error while initializing workers routine: %s", err)
	}

	log.Info("Initializing internal routines...")
	go queue.Pipelines(ctx, a.Cache, a.DBConnectionFactory.GetDBMap)
	go pipeline.AWOLPipelineKiller(ctx, a.DBConnectionFactory.GetDBMap, a.Cache)
	go hatchery.Heartbeat(ctx, a.DBConnectionFactory.GetDBMap)
	go auditCleanerRoutine(ctx, a.DBConnectionFactory.GetDBMap)
	go metrics.Initialize(ctx, a.DBConnectionFactory.GetDBMap, a.Config.Name)
	go repositoriesmanager.ReceiveEvents(ctx, a.DBConnectionFactory.GetDBMap, a.Cache)
	go action.RequirementsCacheLoader(ctx, 5*time.Second, a.DBConnectionFactory.GetDBMap, a.Cache)
	go hookRecoverer(ctx, a.DBConnectionFactory.GetDBMap, a.Cache)
	go services.KillDeadServices(ctx, services.NewRepository(a.mustDB, a.Cache))
	go poller.Initialize(ctx, a.Cache, 10, a.DBConnectionFactory.GetDBMap)
	go migrate.CleanOldWorkflow(ctx, a.Cache, a.DBConnectionFactory.GetDBMap, a.Config.URL.API)
	go migrate.KeyMigration(a.Cache, a.DBConnectionFactory.GetDBMap, &sdk.User{Admin: true})

	a.warnChan = make(chan sdk.Event)
	event.Subscribe(a.warnChan)
	go warning.Compute(ctx, a.Cache, a.DBConnectionFactory.GetDBMap, a.warnChan)
	go a.serviceAPIHeartbeat(ctx)

	//Temporary migration code
	if os.Getenv("CDS_MIGRATE_ENABLE") == "true" {
		go func() {
			if err := migrate.MigrateActionDEPRECATEDGitClone(a.mustDB, a.Cache); err != nil {
				log.Error("Bootstrap Error: %v", err)
			}
		}()
	}
	if !a.Config.Schedulers.Disabled {
		go scheduler.Initialize(ctx, a.Cache, 10, a.DBConnectionFactory.GetDBMap)
	} else {
		log.Warning("⚠ Cron Scheduler is disabled")
	}
	go workflow.Initialize(ctx, a.Cache, a.DBConnectionFactory.GetDBMap)

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", a.Config.HTTP.Addr, a.Config.HTTP.Port),
		Handler:        a.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		select {
		case <-ctx.Done():
			log.Warning("Cleanup SQL connections")
			s.Shutdown(ctx)
			a.DBConnectionFactory.Close()
			event.Publish(sdk.EventEngine{Message: "shutdown"}, nil)
			event.Close()
		}
	}()

	event.Publish(sdk.EventEngine{Message: fmt.Sprintf("started - listen on %d", a.Config.HTTP.Port)}, nil)

	go func() {
		//TLS is disabled for the moment. We need to serve TLS on HTTP too
		if err := grpcInit(a.DBConnectionFactory, a.Config.GRPC.Addr, a.Config.GRPC.Port, false, "", ""); err != nil {
			log.Error("Cannot start GRPC server: %v", err)
		}
	}()

	log.Info("Starting CDS API HTTP Server on %s:%d", a.Config.HTTP.Addr, a.Config.HTTP.Port)
	if err := s.ListenAndServe(); err != nil {
		return fmt.Errorf("Cannot start HTTP server: %v", err)
	}

	return nil
}
