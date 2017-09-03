package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

var mainCmd = &cobra.Command{
	Use:   "api",
	Short: "CDS Engine",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()

		//Check the first config key
		if viper.GetString(viperURLAPI) == "" {
			sdk.Exit("Your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration.")
		}

		logLevel := viper.GetString("log_level")
		if logLevel == "" {
			logLevel = viper.GetString("log.level")
		}
		log.Initialize(&log.Conf{Level: logLevel})
		log.Info("Starting CDS server...")

		startupTime = time.Now()

		//Initliaze context
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)

		// Gracefully shutdown sql connections
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
		defer func() {
			signal.Stop(c)
			cancel()
		}()
		go func() {
			select {
			case <-c:
				log.Warning("Cleanup SQL connections")
				database.Close()
				cancel()
				event.Publish(sdk.EventEngine{Message: "shutdown"})
				event.Close()
				os.Exit(0)

			case <-ctx.Done():
			}
		}()

		//Initialize secret driver
		secret.Init(viper.GetString(viperServerSecretKey))

		//Initialize mail package
		mail.Init(viper.GetString(viperSMTPUser),
			viper.GetString(viperSMTPPassword),
			viper.GetString(viperSMTPFrom),
			viper.GetString(viperSMTPHost),
			viper.GetString(viperSMTPPort),
			viper.GetBool(viperSMTPTLS),
			viper.GetBool(viperSMTPDisable))

		//Initialize artifacts storage
		var objectstoreKind objectstore.Kind
		switch viper.GetString(viperArtifactMode) {
		case "openstack", "swift":
			objectstoreKind = objectstore.Openstack
		case "filesystem", "local":
			objectstoreKind = objectstore.Filesystem
		default:
			log.Fatalf("Unsupported objecstore mode : %s", viper.GetString(viperArtifactMode))
		}

		cfg := objectstore.Config{
			Kind: objectstoreKind,
			Options: objectstore.ConfigOptions{
				Openstack: objectstore.ConfigOptionsOpenstack{
					Address:         viper.GetString(viperArtifactOSURL),
					Username:        viper.GetString(viperArtifactOSUsername),
					Password:        viper.GetString(viperArtifactOSPassword),
					Tenant:          viper.GetString(viperArtifactOSTenant),
					Region:          viper.GetString(viperArtifactOSRegion),
					ContainerPrefix: viper.GetString(viperArtifactOSContainerPrefix),
				},
				Filesystem: objectstore.ConfigOptionsFilesystem{
					Basedir: viper.GetString(viperArtifactLocalBasedir),
				},
			},
		}

		if err := objectstore.Initialize(ctx, cfg); err != nil {
			log.Fatalf("Cannot initialize storage: %s", err)
		}

		//Intialize database
		if _, err := database.Init(
			viper.GetString(viperDBUser),
			viper.GetString(viperDBPassword),
			viper.GetString(viperDBName),
			viper.GetString(viperDBHost),
			viper.GetString(viperDBPort),
			viper.GetString(viperDBSSLMode),
			viper.GetInt(viperDBTimeout),
			viper.GetInt(viperDBMaxConn),
		); err != nil {
			log.Error("Cannot connect to database: %s", err)
			os.Exit(3)
		}

		defaultValues := sdk.DefaultValues{
			DefaultGroupName: viper.GetString(viperAuthDefaultGroup),
			SharedInfraToken: viper.GetString(viperAuthSharedInfraToken),
		}
		if err := bootstrap.InitiliazeDB(defaultValues, database.GetDBMap); err != nil {
			log.Error("Cannot setup databases: %s", err)
		}

		if err := workflow.CreateBuiltinWorkflowHookModels(database.GetDBMap()); err != nil {
			log.Error("Cannot setup builtin workflow hook models")
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
	},
}

func main() {
	mainCmd.Execute()
}
