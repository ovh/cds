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
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/mail"
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
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var startupTime time.Time
var b *Broker
var baseURL string
var localCLientAuthMode = auth.LocalClientBasicAuthMode

var mainCmd = &cobra.Command{
	Use:   "api",
	Short: "CDS Engine",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("cds")
		viper.AutomaticEnv()

		log.Initialize()
		log.Notice("Starting CDS server...\n")

		startupTime = time.Now()

		//Initialize secret driver
		secretBackend := viper.GetString("secret_backend")
		secretBackendOptions := viper.GetStringSlice("secret_backend_option")
		secretBackendOptionsMap := map[string]string{}
		for _, o := range secretBackendOptions {
			if !strings.Contains(o, "=") {
				log.Warning("Malformated options : %s", o)
				continue
			}
			t := strings.Split(o, "=")
			secretBackendOptionsMap[t[0]] = t[1]
		}
		if err := secret.Init(secretBackend, secretBackendOptionsMap); err != nil {
			log.Critical("Cannot initialize secret manager: %s\n", err)
		}

		if secret.SecretUsername != "" {
			database.SecretDBUser = secret.SecretUsername
		}
		if secret.SecretPassword != "" {
			database.SecretDBPassword = secret.SecretPassword
		}

		if err := mail.CheckMailConfiguration(); err != nil {
			log.Fatalf("SMTP configuration error: %s\n", err)
		}

		var objectstoreKind objectstore.Kind
		switch viper.GetString("artifact_mode") {
		case "openstack", "swift":
			objectstoreKind = objectstore.Openstack
		case "filesystem":
			objectstoreKind = objectstore.Filesystem
		default:
			log.Fatalf("Unsupported objectore mode : %s", viper.GetString("artifact_mode"))
		}

		cfg := objectstore.Config{
			Kind: objectstoreKind,
			Options: objectstore.ConfigOptions{
				Openstack: objectstore.ConfigOptionsOpenstack{
					Address:  viper.GetString("artifact_address"),
					Username: viper.GetString("artifact_user"),
					Password: viper.GetString("artifact_password"),
					Tenant:   viper.GetString("artifact_tenant"),
					Region:   viper.GetString("artifact_region"),
				},
				Filesystem: objectstore.ConfigOptionsFilesystem{
					Basedir: viper.GetString("artifact_basedir"),
				},
			},
		}

		if err := objectstore.Initialize(cfg); err != nil {
			log.Fatalf("Cannot initialize storage: %s\n", err)
		}

		db, err := database.Init()
		if err != nil {
			log.Warning("Cannot connect to database: %s\n", err)
		}
		if db != nil {
			if viper.GetBool("db_logging") {
				log.UseDatabaseLogger(db)
			}

			if err = bootstrap.InitiliazeDB(database.GetDBMap); err != nil {
				log.Critical("Cannot setup databases: %s\n", err)
			}

			// Gracefully shutdown sql connections
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			signal.Notify(c, syscall.SIGTERM)
			signal.Notify(c, syscall.SIGKILL)
			go func() {
				<-c
				log.Warning("Cleanup SQL connections\n")
				db.Close()
				event.Publish(sdk.EventEngine{Message: "shutdown"})
				event.Close()
				os.Exit(0)
			}()
		}

		// Make a new Broker instance
		b = &Broker{
			make(map[chan string]bool),
			make(chan (chan string)),
			make(chan (chan string)),
			make(chan string),
		}

		b.Start()

		router = &Router{
			mux: mux.NewRouter(),
		}
		router.init()
		baseURL = viper.GetString("base_url")

		if err := group.Initialize(database.DBMap(db), viper.GetString("default_group")); err != nil {
			log.Critical("Cannot initialize groups: %s\n", err)
		}

		//Intialize repositories manager
		rmInitOpts := repositoriesmanager.InitializeOpts{
			SecretClient:           secret.Client,
			KeysDirectory:          viper.GetString("keys_directory"),
			UIBaseURL:              baseURL,
			APIBaseURL:             viper.GetString("api_url"),
			DisableGithubSetStatus: viper.GetBool("no_github_status"),
			DisableGithubStatusURL: viper.GetBool("no_github_status_url"),
			DisableStashSetStatus:  viper.GetBool("no_stash_status"),
		}
		if err := repositoriesmanager.Initialize(rmInitOpts); err != nil {
			log.Warning("Error initializing repositories manager connections: %s\n", err)
		}

		// Initialize the auth driver
		var authMode string
		var authOptions interface{}
		switch viper.GetBool("ldap_enable") {
		case true:
			authMode = "ldap"
			authOptions = auth.LDAPConfig{
				Host:         viper.GetString("ldap_host"),
				Port:         viper.GetInt("ldap_port"),
				Base:         viper.GetString("ldap_base"),
				DN:           viper.GetString("ldap_dn"),
				SSL:          viper.GetBool("ldap_ssl"),
				UserFullname: viper.GetString("ldap_user_fullname"),
			}
		default:
			authMode = "local"
		}

		storeOptions := sessionstore.Options{
			Mode:          viper.GetString("cache"),
			TTL:           viper.GetInt("session_ttl"),
			RedisHost:     viper.GetString("redis_host"),
			RedisPassword: viper.GetString("redis_password"),
		}

		router.authDriver, _ = auth.GetDriver(authMode, authOptions, storeOptions)

		cache.Initialize(viper.GetString("cache"), viper.GetString("redis_host"), viper.GetString("redis_password"), viper.GetInt("cache_ttl"))

		kafkaOptions := event.KafkaConfig{
			Enabled:         viper.GetBool("event_kafka_enabled"),
			BrokerAddresses: viper.GetString("event_kafka_broker_addresses"),
			User:            viper.GetString("event_kafka_user"),
			Password:        viper.GetString("event_kafka_password"),
			Topic:           viper.GetString("event_kafka_topic"),
		}
		if err := event.Initialize(kafkaOptions); err != nil {
			log.Warning("⚠ Error while initializing event system: %s", err)
		} else {
			go event.DequeueEvent()
		}

		if err := worker.Initialize(); err != nil {
			log.Warning("⚠ Error while initializing workers routine: %s", err)
		}

		go queue.Pipelines()
		go pipeline.AWOLPipelineKiller(database.GetDBMap)
		go hatchery.Heartbeat(database.GetDBMap)
		go log.RemovalRoutine(database.DB)
		go auditCleanerRoutine(database.GetDBMap)

		go repositoriesmanager.ReceiveEvents()

		go stats.StartRoutine()
		go action.RequirementsCacheLoader(5, database.GetDBMap)
		go hookRecoverer(database.GetDBMap)

		if !viper.GetBool("no_repo_cache_loader") {
			go repositoriesmanager.RepositoriesCacheLoader(30)
		} else {
			log.Warning("⚠ Repositories cache loader is disabled")
		}

		if !viper.GetBool("no_repo_polling") {
			go poller.Initialize(database.GetDBMap, 10)
		} else {
			log.Warning("⚠ Repositories polling is disabled")
		}

		if !viper.GetBool("no_scheduler") {
			go scheduler.Initialize(database.GetDBMap, 10)
		} else {
			log.Warning("⚠ Cron Scheduler is disabled")
		}

		s := &http.Server{
			Addr:           ":" + viper.GetString("listen_port"),
			Handler:        router.mux,
			ReadTimeout:    10 * time.Minute,
			WriteTimeout:   10 * time.Minute,
			MaxHeaderBytes: 1 << 20,
		}

		log.Notice("Listening on :%s\n", viper.GetString("listen_port"))
		event.Publish(sdk.EventEngine{Message: fmt.Sprintf("started - listen on %s", viper.GetString("listen_port"))})
		if err := s.ListenAndServe(); err != nil {
			log.Fatalf("Cannot start cds-server: %s\n", err)
		}
	},
}

func main() {
	mainCmd.Execute()
}
