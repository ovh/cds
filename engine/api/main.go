package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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

var startupTime time.Time
var baseURL string

type Configuration struct {
	URL struct {
		API string
		UI  string
	}
	HTTP struct {
		Port       string
		SessionTTL string
	}
	GRPC struct {
		Port string
	}
	Secrets struct {
		Key string
	}
	Log struct {
		Level string
	}
	Database struct {
		User     string
		Password string
		Name     string
		Host     string
		Port     string
		SSLMode  string
		MaxConn  int
		Timeout  int
		Secret   string
	}
	Cache struct {
		Mode  string
		TTL   string
		Redis struct {
			Host     string
			Password string
		}
	}
	Directories struct {
		Download string
		Keys     string
	}
	Auth struct {
		DefaultGroup     string
		SharedInfraToken string
		LDAP             struct {
			Enable   bool
			Host     string
			Port     int
			SSL      bool
			Base     string
			DN       string
			Fullname string
		}
	}
	SMTP struct {
		Disable  bool
		Host     string
		Port     string
		TLS      bool
		User     string
		Password string
		From     string
	}
	Artifact struct {
		Mode  string
		Local struct {
			BaseDirectory string
		}
		Openstack struct {
			URL             string
			Username        string
			Password        string
			Tenant          string
			Region          string
			ContainerPrefix string
		}
	}
	Events struct {
		Kafka struct {
			Enabled  bool
			Broker   string
			Topic    string
			User     string
			Password string
		}
	}
	Schedulers struct {
		Disabled bool
	}
	VCS struct {
		Polling struct {
			Disabled bool
		}
		Github struct {
			Secret           string
			DisableStatus    bool
			DisableStatusURL bool
		}
		Bitbucket struct {
			DisableStatus bool
			ConsumerKey   string
			PrivateKey    string
		}
	}
	Vault struct {
		ConfigurationKey string
	}
}

type API struct {
	Config Configuration
}

func (api *API) Init(config interface{}) error {
	var ok bool
	api.Config, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	//Check the first config key
	if api.Config.URL.API == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}

	return nil
}

func (api *API) Serve(ctx context.Context) error {

	log.Initialize(&log.Conf{Level: api.Config.Log.Level})
	log.Info("Starting CDS API Server...")

	startupTime = time.Now()

	go func() {
		select {
		case <-ctx.Done():
			log.Warning("Cleanup SQL connections")
			database.Close()
			event.Publish(sdk.EventEngine{Message: "shutdown"})
			event.Close()
		}
	}()

	//Initialize secret driver
	secret.Init(api.Config.Secrets.Key)

	//Initialize mail package
	mail.Init(api.Config.SMTP.User,
		api.Config.SMTP.Password,
		api.Config.SMTP.From,
		api.Config.SMTP.Host,
		api.Config.SMTP.Port,
		api.Config.SMTP.TLS,
		api.Config.SMTP.Disable)

	//Initialize artifacts storage
	var objectstoreKind objectstore.Kind
	switch api.Config.Artifact.Mode {
	case "openstack", "swift":
		objectstoreKind = objectstore.Openstack
	case "filesystem", "local":
		objectstoreKind = objectstore.Filesystem
	default:
		log.Fatalf("Unsupported objecstore mode : %s", api.Config.Artifact.Mode)
	}

	cfg := objectstore.Config{
		Kind: objectstoreKind,
		Options: objectstore.ConfigOptions{
			Openstack: objectstore.ConfigOptionsOpenstack{
				Address:         api.Config.Artifact.Openstack.URL,
				Username:        api.Config.Artifact.Openstack.Username,
				Password:        api.Config.Artifact.Openstack.Password,
				Tenant:          api.Config.Artifact.Openstack.Tenant,
				Region:          api.Config.Artifact.Openstack.Region,
				ContainerPrefix: api.Config.Artifact.Openstack.ContainerPrefix,
			},
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: api.Config.Artifact.Local.BaseDirectory,
			},
		},
	}

	if err := objectstore.Initialize(ctx, cfg); err != nil {
		log.Fatalf("Cannot initialize storage: %s", err)
	}

	//Intialize database
	if _, err := database.Init(
		api.Config.Database.User,
		api.Config.Database.Password,
		api.Config.Database.Name,
		api.Config.Database.Host,
		api.Config.Database.Port,
		api.Config.Database.SSLMode,
		api.Config.Database.Timeout,
		api.Config.Database.MaxConn,
	); err != nil {
		log.Error("Cannot connect to database: %s", err)
		os.Exit(3)
	}

	defaultValues := bootstrap.DefaultValues{
		DefaultGroupName: api.Config.Auth.DefaultGroup,
		SharedInfraToken: api.Config.Auth.SharedInfraToken,
	}
	if err := bootstrap.InitiliazeDB(defaultValues, database.GetDBMap); err != nil {
		log.Error("Cannot setup databases: %s", err)
	}

	cache.Initialize(viper.GetString(viperCacheMode), viper.GetString(viperCacheRedisHost), viper.GetString(viperCacheRedisPassword), viper.GetInt(viperCacheTTL))
	InitLastUpdateBroker(ctx, database.GetDBMap)

	router = &Router{
		mux: mux.NewRouter(),
	}
	router.init()
	router.url = viper.GetString(viperURLAPI)

	baseURL = viper.GetString(viperURLUI)

	//Intialize repositories manager
	rmInitOpts := repositoriesmanager.InitializeOpts{
		KeysDirectory:          viper.GetString(viperKeysDirectory),
		UIBaseURL:              baseURL,
		APIBaseURL:             viper.GetString(viperURLAPI),
		DisableGithubSetStatus: viper.GetBool(viperVCSRepoGithubStatusDisabled),
		DisableGithubStatusURL: viper.GetBool(viperVCSRepoGithubStatusURLDisabled),
		DisableStashSetStatus:  viper.GetBool(viperVCSRepoBitbucketStatusDisabled),
		GithubSecret:           viper.GetString(viperVCSRepoGithubSecret),
		StashPrivateKey:        viper.GetString(viperVCSRepoBitbucketPrivateKey),
		StashConsumerKey:       viper.GetString(viperVCSRepoBitbucketConsumerKey),
	}
	if err := repositoriesmanager.Initialize(rmInitOpts); err != nil {
		log.Warning("Error initializing repositories manager connections: %s", err)
	}

	//Initiliaze hook package
	hook.Init(viper.GetString(viperURLAPI))

	//Intialize notification package
	notification.Init(viper.GetString(viperURLAPI), baseURL)

	// Initialize the auth driver
	var authMode string
	var authOptions interface{}
	switch viper.GetBool(viperAuthLDAPEnable) {
	case true:
		authMode = "ldap"
		authOptions = auth.LDAPConfig{
			Host:         viper.GetString(viperAuthLDAPHost),
			Port:         viper.GetInt(viperAuthLDAPPort),
			Base:         viper.GetString(viperAuthLDAPBase),
			DN:           viper.GetString(viperAuthLDAPDN),
			SSL:          viper.GetBool(viperAuthLDAPSSL),
			UserFullname: viper.GetString(viperAuthLDAPFullname),
		}
	default:
		authMode = "local"
	}

	storeOptions := sessionstore.Options{
		Mode:          viper.GetString(viperCacheMode),
		TTL:           viper.GetInt(viperCacheTTL),
		RedisHost:     viper.GetString(viperCacheRedisHost),
		RedisPassword: viper.GetString(viperCacheRedisPassword),
	}

	var errdriver error
	router.authDriver, errdriver = auth.GetDriver(ctx, authMode, authOptions, storeOptions)
	if errdriver != nil {
		log.Fatalf("Error: %v", errdriver)
	}

	kafkaOptions := event.KafkaConfig{
		Enabled:         viper.GetBool(viperEventsKafkaEnabled),
		BrokerAddresses: viper.GetString(viperEventsKafkaBroker),
		User:            viper.GetString(viperEventsKafkaUser),
		Password:        viper.GetString(viperEventsKafkaPassword),
		Topic:           viper.GetString(viperEventsKafkaTopic),
	}
	if err := event.Initialize(kafkaOptions); err != nil {
		log.Warning("⚠ Error while initializing event system: %s", err)
	} else {
		go event.DequeueEvent(ctx)
	}

	if err := worker.Initialize(ctx, database.GetDBMap); err != nil {
		log.Warning("⚠ Error while initializing workers routine: %s", err)
	}

	go queue.Pipelines(ctx, database.GetDBMap)
	go workflow.Scheduler(ctx, database.GetDBMap)
	go pipeline.AWOLPipelineKiller(ctx, database.GetDBMap)
	go hatchery.Heartbeat(ctx, database.GetDBMap)
	go auditCleanerRoutine(ctx, database.GetDBMap)

	go repositoriesmanager.ReceiveEvents(ctx, database.GetDBMap)

	go stats.StartRoutine(ctx, database.GetDBMap)
	go action.RequirementsCacheLoader(ctx, 5*time.Second, database.GetDBMap)
	go hookRecoverer(ctx, database.GetDBMap)

	go user.PersistentSessionTokenCleaner(ctx, database.GetDBMap)

	if !viper.GetBool(viperVCSPollingDisabled) {
		go poller.Initialize(ctx, 10, database.GetDBMap)
	} else {
		log.Warning("⚠ Repositories polling is disabled")
	}

	if !viper.GetBool(viperSchedulersDisabled) {
		go scheduler.Initialize(ctx, 10, database.GetDBMap)
	} else {
		log.Warning("⚠ Cron Scheduler is disabled")
	}

	s := &http.Server{
		Addr:           ":" + viper.GetString(viperServerHTTPPort),
		Handler:        router.mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	event.Publish(sdk.EventEngine{Message: fmt.Sprintf("started - listen on %s", viper.GetString(viperServerHTTPPort))})

	go func() {
		//TLS is disabled for the moment. We need to serve TLS on HTTP too
		if err := grpc.Init(viper.GetInt(viperServerGRPCPort), false, "", ""); err != nil {
			log.Fatalf("Cannot start grpc cds-server: %s", err)
		}
	}()

	log.Info("Starting HTTP Server on port %s", viper.GetString(viperServerHTTPPort))
	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Cannot start cds-server: %s", err)
	}

	return nil
}

var mainCmd = &cobra.Command{
	Use:   "api",
	Short: "CDS Engine",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()

	},
}

func main() {
	mainCmd.Execute()
}
