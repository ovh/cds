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
	"github.com/ovh/cds/engine/api/grpc"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/queue"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/stats"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Configuration is the configuraton structure for CDS API
type Configuration struct {
	URL struct {
		API string `default:"http://localhost:8081"`
		UI  string `default:"http://localhost:4200"`
	} `comment:"#####################\n# CDS URLs Settings #\n#####################"`
	HTTP struct {
		Port       int `default:"8081"`
		SessionTTL int `default:"60"`
	}
	GRPC struct {
		Port int `default:"8082"`
	}
	Secrets struct {
		Key string
	}
	Database struct {
		User     string `default:"cds"`
		Password string `default:"cds"`
		Name     string `default:"cds"`
		Host     string `default:"localhost"`
		Port     int    `default:"5432"`
		SSLMode  string `default:"disable"`
		MaxConn  int    `default:"20"`
		Timeout  int    `default:"3000"`
		Secret   string
	} `comment:"################################\n# Postgresql Database settings #\n################################"`
	Cache struct {
		Mode  string `default:"local" comment:"Cache Mode: redis or local"`
		TTL   int    `default:"60"`
		Redis struct {
			Host     string `default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379"`
			Password string
		} `comment:"Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup"`
	} `comment:"######################\n# CDS Cache Settings #\n######################\nIf your CDS is made of a unique instance, a local cache if enough, but rememeber that all cached data will be lost on startup."`
	Directories struct {
		Download string `default:"/tmp/cds/download"`
		Keys     string `default:"/tmp/cds/keys"`
	}
	Auth struct {
		DefaultGroup     string `default:"" comment:"The default group is the group in which every new user will be granted at signup"`
		SharedInfraToken string `default:"" comment:"Token for shared.infra group. This value will be used when shared.infra will be created\nat first CDS launch. This token can be used by CDS CLI, Hatchery, etc...\nThis is mandatory."`
		LDAP             struct {
			Enable   bool `default:"false"`
			Host     string
			Port     int    `default:"636"`
			SSL      bool   `default:"true"`
			Base     string `default:"dc=myorganization,dc=com"`
			DN       string `default:"uid=%s,ou=people,dc=myorganization,dc=com"`
			Fullname string `default:"{{.givenName}} {{.sn}}"`
		}
	} `comment:"##############################\n# CDS Authentication Settings#\n##############################"`
	SMTP struct {
		Disable  bool `default:"true"`
		Host     string
		Port     string
		TLS      bool
		User     string
		Password string
		From     string `default:"no-reply@cds.local"`
	} `comment:"#####################n# CDS SMTP Settings #\n#####################"`
	Artifact struct {
		Mode  string `default:"local" comment:"swift or local"`
		Local struct {
			BaseDirectory string `default:"/tmp/cds/artifacts"`
		}
		Openstack struct {
			URL             string `comment:"Authentication Endpoint, generally value of $OS_AUTH_URL"`
			Username        string `comment:"Openstack Username, generally value of $OS_USERNAME"`
			Password        string `comment:"Openstack Password, generally value of $OS_PASSWORD"`
			Tenant          string `comment:"Openstack Tenant, generally value of $OS_TENANT_NAME"`
			Region          string `comment:"Region, generally value of $OS_REGION_NAME"`
			ContainerPrefix string `comment:"Use if your want to prefix containers for CDS Artifacts"`
		}
	} `comment:"Either filesystem local storage or Openstack Swift Storage are supported"`
	Events struct {
		Kafka struct {
			Enabled  bool
			Broker   string
			Topic    string
			User     string
			Password string
		}
	} `comment:"#######################\n# CDS Events Settings #\n#######################"`
	Schedulers struct {
		Disabled bool `default:"false" commented:"true" comment:"This is mainly for dev purpose, you should not have to change it"`
	} `comment:"###########################\n# CDS Schedulers Settings #\n###########################"`
	VCS struct {
		Polling struct {
			Disabled bool `default:"false" commented:"true" comment:"This is mainly for dev purpose, you should not have to change it"`
		}
		Github struct {
			Secret           string
			DisableStatus    bool `default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on Github API"`
			DisableStatusURL bool `default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on Github API"`
		}
		Bitbucket struct {
			DisableStatus bool `default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on Bitbucket API"`
			ConsumerKey   string
			PrivateKey    string
		}
	} `comment:"####################\n# CDS VCS Settings #\n####################"`
	Vault struct {
		ConfigurationKey string
	}
}

// DefaultValues is the struc for API Default configuration default values
type DefaultValues struct {
	ServerSecretsKey     string
	AuthSharedInfraToken string
	// For LDAP Client
	LDAPBase  string
	GivenName string
	SN        string
}

// New instanciates a new API object
func New() *API {
	return &API{}
}

// API is a struct containing the configuration, the router, the database connection factory and so on
type API struct {
	Router              *Router
	Config              Configuration
	DBConnectionFactory *database.DBConnectionFactory
	StartupTime         time.Time
	lastUpdateBroker    *lastUpdateBroker
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
		return fmt.Errorf("Invalid configuration")
	}

	if aConfig.URL.API == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}

	if aConfig.Directories.Download == "" {
		return fmt.Errorf("Invalid download directory")
	}

	if ok, err := DirectoryExists(aConfig.Directories.Download); !ok {
		if err := os.MkdirAll(aConfig.Directories.Download, os.FileMode(0600)); err != nil {
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
		if err := os.MkdirAll(aConfig.Directories.Keys, os.FileMode(0600)); err != nil {
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
			if err := os.MkdirAll(aConfig.Artifact.Local.BaseDirectory, os.FileMode(0600)); err != nil {
				return fmt.Errorf("Unable to create directory %s: %v", aConfig.Artifact.Local.BaseDirectory, err)
			}
			log.Info("Directory %s has been created", aConfig.Artifact.Local.BaseDirectory)
		} else if err != nil {
			return fmt.Errorf("Invalid artifact local base directory: %v", err)
		}
	}

	switch aConfig.Cache.Mode {
	case "local", "redis":
	default:
		return fmt.Errorf("Invalid cache mode")
	}

	if len(aConfig.Secrets.Key) != 32 {
		return fmt.Errorf("Invalid secret key. It should be 32 bits (%d)", len(aConfig.Secrets.Key))
	}

	return nil
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

func getAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
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

func (a *API) mustDB() *gorp.DbMap {
	db := a.DBConnectionFactory.GetDBMap()
	if db == nil {
		panic(fmt.Errorf("Database unavailable"))
	}
	return db
}

// Serve will start the http api server
func (a *API) Serve(ctx context.Context) error {
	log.Info("Starting CDS API Server...")

	a.StartupTime = time.Now()

	//Initialize secret driver
	secret.Init(a.Config.Secrets.Key)

	//Initialize mail package
	mail.Init(a.Config.SMTP.User,
		a.Config.SMTP.Password,
		a.Config.SMTP.From,
		a.Config.SMTP.Host,
		a.Config.SMTP.Port,
		a.Config.SMTP.TLS,
		a.Config.SMTP.Disable)

	//Initialize artifacts storage
	var objectstoreKind objectstore.Kind
	switch a.Config.Artifact.Mode {
	case "openstack", "swift":
		objectstoreKind = objectstore.Openstack
	case "filesystem", "local":
		objectstoreKind = objectstore.Filesystem
	default:
		log.Fatalf("Unsupported objecstore mode : %s", a.Config.Artifact.Mode)
	}

	cfg := objectstore.Config{
		Kind: objectstoreKind,
		Options: objectstore.ConfigOptions{
			Openstack: objectstore.ConfigOptionsOpenstack{
				Address:         a.Config.Artifact.Openstack.URL,
				Username:        a.Config.Artifact.Openstack.Username,
				Password:        a.Config.Artifact.Openstack.Password,
				Tenant:          a.Config.Artifact.Openstack.Tenant,
				Region:          a.Config.Artifact.Openstack.Region,
				ContainerPrefix: a.Config.Artifact.Openstack.ContainerPrefix,
			},
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: a.Config.Artifact.Local.BaseDirectory,
			},
		},
	}

	if err := objectstore.Initialize(ctx, cfg); err != nil {
		log.Fatalf("Cannot initialize storage: %s", err)
	}

	//Intialize database
	var errDB error
	a.DBConnectionFactory, errDB = database.Init(
		a.Config.Database.User,
		a.Config.Database.Password,
		a.Config.Database.Name,
		a.Config.Database.Host,
		a.Config.Database.Port,
		a.Config.Database.SSLMode,
		a.Config.Database.Timeout,
		a.Config.Database.MaxConn)
	if errDB != nil {
		log.Error("Cannot connect to database: %s", errDB)
		os.Exit(3)
	}

	defaultValues := sdk.DefaultValues{
		DefaultGroupName: a.Config.Auth.DefaultGroup,
		SharedInfraToken: a.Config.Auth.SharedInfraToken,
	}
	if err := bootstrap.InitiliazeDB(defaultValues, a.DBConnectionFactory.GetDBMap); err != nil {
		log.Error("Cannot setup databases: %s", err)
	}

	if err := workflow.CreateBuiltinWorkflowHookModels(a.DBConnectionFactory.GetDBMap()); err != nil {
		log.Error("Cannot setup builtin workflow hook models")
	}

	//Init the cache
	var errCache error
	a.Cache, errCache = cache.New(
		a.Config.Cache.Mode,
		a.Config.Cache.Redis.Host,
		a.Config.Cache.Redis.Password,
		a.Config.Cache.TTL)
	if errCache != nil {
		log.Error("Cannot connect to cache store: %s", errCache)
		os.Exit(3)
	}

	a.Router = &Router{
		Mux:        mux.NewRouter(),
		Background: ctx,
	}
	a.InitRouter()

	//Intialize repositories manager
	rmInitOpts := repositoriesmanager.InitializeOpts{
		KeysDirectory:          a.Config.Directories.Keys,
		UIBaseURL:              a.Config.URL.UI,
		APIBaseURL:             a.Config.URL.API,
		DisableGithubSetStatus: a.Config.VCS.Github.DisableStatus,
		DisableGithubStatusURL: a.Config.VCS.Github.DisableStatusURL,
		DisableStashSetStatus:  a.Config.VCS.Bitbucket.DisableStatus,
		GithubSecret:           a.Config.VCS.Github.Secret,
		StashPrivateKey:        a.Config.VCS.Bitbucket.PrivateKey,
		StashConsumerKey:       a.Config.VCS.Bitbucket.ConsumerKey,
	}
	if err := repositoriesmanager.Initialize(rmInitOpts, a.DBConnectionFactory.GetDBMap, a.Cache); err != nil {
		log.Warning("Error initializing repositories manager connections: %s", err)
	}

	//Init pipeline package
	pipeline.Store = a.Cache

	//Init events package
	event.Cache = a.Cache

	//Initiliaze hook package
	hook.Init(a.Config.URL.API)

	//Intialize notification package
	notification.Init(a.Config.URL.API, a.Config.URL.UI)

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
		}
	default:
		authMode = "local"
	}

	storeOptions := sessionstore.Options{
		Mode:          a.Config.Cache.Mode,
		TTL:           a.Config.Cache.TTL,
		RedisHost:     a.Config.Cache.Redis.Host,
		RedisPassword: a.Config.Cache.Redis.Password,
	}

	var errdriver error
	a.Router.AuthDriver, errdriver = auth.GetDriver(ctx, authMode, authOptions, storeOptions, a.DBConnectionFactory.GetDBMap)
	if errdriver != nil {
		log.Fatalf("Error: %v", errdriver)
	}

	kafkaOptions := event.KafkaConfig{
		Enabled:         a.Config.Events.Kafka.Enabled,
		BrokerAddresses: a.Config.Events.Kafka.Broker,
		User:            a.Config.Events.Kafka.User,
		Password:        a.Config.Events.Kafka.Password,
		Topic:           a.Config.Events.Kafka.Topic,
	}
	if err := event.Initialize(kafkaOptions); err != nil {
		log.Warning("⚠ Error while initializing event system: %s", err)
	} else {
		go event.DequeueEvent(ctx)
	}

	if err := worker.Initialize(ctx, a.DBConnectionFactory.GetDBMap, a.Cache); err != nil {
		log.Warning("⚠ Error while initializing workers routine: %s", err)
	}

	go queue.Pipelines(ctx, a.Cache, a.DBConnectionFactory.GetDBMap)
	go pipeline.AWOLPipelineKiller(ctx, a.DBConnectionFactory.GetDBMap)
	go hatchery.Heartbeat(ctx, a.DBConnectionFactory.GetDBMap)
	go auditCleanerRoutine(ctx, a.DBConnectionFactory.GetDBMap)
	go repositoriesmanager.ReceiveEvents(ctx, a.DBConnectionFactory.GetDBMap, a.Cache)
	go stats.StartRoutine(ctx, a.DBConnectionFactory.GetDBMap)
	go action.RequirementsCacheLoader(ctx, 5*time.Second, a.DBConnectionFactory.GetDBMap, a.Cache)
	go hookRecoverer(ctx, a.DBConnectionFactory.GetDBMap, a.Cache)

	go user.PersistentSessionTokenCleaner(ctx, a.DBConnectionFactory.GetDBMap)

	if !a.Config.VCS.Polling.Disabled {
		go poller.Initialize(ctx, a.Cache, 10, a.DBConnectionFactory.GetDBMap)
	} else {
		log.Warning("⚠ Repositories polling is disabled")
	}

	if !a.Config.Schedulers.Disabled {
		go scheduler.Initialize(ctx, a.Cache, 10, a.DBConnectionFactory.GetDBMap)
	} else {
		log.Warning("⚠ Cron Scheduler is disabled")
	}

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", a.Config.HTTP.Port),
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
			event.Publish(sdk.EventEngine{Message: "shutdown"})
			event.Close()
		}
	}()

	event.Publish(sdk.EventEngine{Message: fmt.Sprintf("started - listen on %d", a.Config.HTTP.Port)})

	go func() {
		//TLS is disabled for the moment. We need to serve TLS on HTTP too
		if err := grpc.Init(a.DBConnectionFactory, a.Config.GRPC.Port, false, "", ""); err != nil {
			log.Fatalf("Cannot start grpc cds-server: %s", err)
		}
	}()

	log.Info("Starting HTTP Server on port %d", a.Config.HTTP.Port)
	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Cannot start cds-server: %s", err)
	}

	return nil
}
