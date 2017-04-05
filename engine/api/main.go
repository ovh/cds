package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	"github.com/ovh/cds/engine/api/group"
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
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var startupTime time.Time
var baseURL string
var localCLientAuthMode = auth.LocalClientBasicAuthMode

var mainCmd = &cobra.Command{
	Use:   "api",
	Short: "CDS Engine",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()

		//Check the first config key
		if viper.GetString(viperURLAPI) == "" {
			sdk.Exit("Your CDS configuration seems to not be set. Please use environment variables, file or etcd to set your configuration.")
		}

		logLevel := viper.GetString("log_level")
		if logLevel == "" {
			logLevel = viper.GetString("log.level")
		}
		log.Initialize(&log.Conf{Level: logLevel})
		log.Info("Starting CDS server...")

		startupTime = time.Now()

		//Initialize secret driver
		secretBackend := viper.GetString(viperServerSecretBackend)
		secretBackendOptions := viper.GetStringSlice(viperServerSecretBackendOption)
		secretBackendOptionsMap := map[string]string{}
		for _, o := range secretBackendOptions {
			if !strings.Contains(o, "=") {
				log.Warning("Malformated options : %s", o)
				continue
			}
			t := strings.Split(o, "=")
			secretBackendOptionsMap[t[0]] = t[1]
		}
		if err := secret.Init(viper.GetString(viperDBSecret), viper.GetString(viperServerSecretKey), secretBackend, secretBackendOptionsMap); err != nil {
			log.Critical("Cannot initialize secret manager: %s", err)
		}
		if secret.SecretUsername != "" {
			database.SecretDBUser = secret.SecretUsername
		}
		if secret.SecretPassword != "" {
			database.SecretDBPassword = secret.SecretPassword
		}

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
					Address:  viper.GetString(viperArtifactOSURL),
					Username: viper.GetString(viperArtifactOSUsername),
					Password: viper.GetString(viperArtifactOSPassword),
					Tenant:   viper.GetString(viperArtifactOSTenant),
					Region:   viper.GetString(viperArtifactOSRegion),
				},
				Filesystem: objectstore.ConfigOptionsFilesystem{
					Basedir: viper.GetString(viperArtifactLocalBasedir),
				},
			},
		}

		if err := objectstore.Initialize(cfg); err != nil {
			log.Fatalf("Cannot initialize storage: %s", err)
		}

		//Intialize database
		db, err := database.Init(
			viper.GetString(viperDBUser),
			viper.GetString(viperDBPassword),
			viper.GetString(viperDBName),
			viper.GetString(viperDBHost),
			viper.GetString(viperDBPort),
			viper.GetString(viperDBSSLMode),
			viper.GetInt(viperDBTimeout),
			viper.GetInt(viperDBMaxConn),
		)
		if err != nil {
			log.Warning("Cannot connect to database: %s", err)
			os.Exit(3)
		}

		if err = bootstrap.InitiliazeDB(database.GetDBMap); err != nil {
			log.Critical("Cannot setup databases: %s", err)
		}

		// Gracefully shutdown sql connections
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		signal.Notify(c, syscall.SIGTERM)
		signal.Notify(c, syscall.SIGKILL)
		go func() {
			<-c
			log.Warning("Cleanup SQL connections")
			db.Close()
			event.Publish(sdk.EventEngine{Message: "shutdown"})
			event.Close()
			os.Exit(0)
		}()

		router = &Router{
			mux: mux.NewRouter(),
		}
		router.init()

		baseURL = viper.GetString(viperURLUI)

		//Intialize repositories manager
		rmInitOpts := repositoriesmanager.InitializeOpts{
			SecretClient:           secret.Client,
			KeysDirectory:          viper.GetString(viperKeysDirectory),
			UIBaseURL:              baseURL,
			APIBaseURL:             viper.GetString(viperURLAPI),
			DisableGithubSetStatus: viper.GetBool(viperVCSRepoGithubStatusDisabled),
			DisableGithubStatusURL: viper.GetBool(viperVCSRepoGithubStatusURLDisabled),
			DisableStashSetStatus:  viper.GetBool(viperVCSRepoBitbucketStatusDisabled),
			GithubSecret:           viper.GetString(viperVCSRepoGithubSecret),
			StashPrivateKey:        viper.GetString(viperVCSRepoBitbucketPrivateKey),
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
			if viper.GetString(viperAuthMode) == "basic" {
				log.Info("Authentitication mode: Basic")
				localCLientAuthMode = auth.LocalClientBasicAuthMode
			} else {
				log.Info("Authentitication mode: Session")
				localCLientAuthMode = auth.LocalClientSessionMode
			}
		}

		storeOptions := sessionstore.Options{
			Mode:          viper.GetString(viperCacheMode),
			TTL:           viper.GetInt(viperCacheTTL),
			RedisHost:     viper.GetString(viperCacheRedisHost),
			RedisPassword: viper.GetString(viperCacheRedisPassword),
		}

		router.authDriver, _ = auth.GetDriver(authMode, authOptions, storeOptions)

		cache.Initialize(viper.GetString(viperCacheMode), viper.GetString(viperCacheRedisHost), viper.GetString(viperCacheRedisPassword), viper.GetInt(viperCacheTTL))

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
			go event.DequeueEvent()
		}

		if err := worker.Initialize(); err != nil {
			log.Warning("⚠ Error while initializing workers routine: %s", err)
		}

		if err := group.Initialize(database.DBMap(db), viper.GetString(viperAuthDefaultGroup)); err != nil {
			log.Critical("Cannot initialize groups: %s", err)
		}

		go queue.Pipelines()
		go pipeline.AWOLPipelineKiller(database.GetDBMap)
		go hatchery.Heartbeat(database.GetDBMap)
		go auditCleanerRoutine(database.GetDBMap)

		go repositoriesmanager.ReceiveEvents()

		go stats.StartRoutine()
		go action.RequirementsCacheLoader(5, database.GetDBMap)
		go hookRecoverer(database.GetDBMap)

		if !viper.GetBool(viperVCSRepoCacheLoaderDisabled) {
			go repositoriesmanager.RepositoriesCacheLoader(30)
		} else {
			log.Warning("⚠ Repositories cache loader is disabled")
		}

		if !viper.GetBool(viperVCSPollingDisabled) {
			go poller.Initialize(database.GetDBMap, 10)
		} else {
			log.Warning("⚠ Repositories polling is disabled")
		}

		if !viper.GetBool(viperSchedulersDisabled) {
			go scheduler.Initialize(database.GetDBMap, 10)
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
