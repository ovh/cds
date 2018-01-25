package main

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	cors "github.com/itsjamie/gin-cors"
	"github.com/ovh/tat"
	"github.com/ovh/tat/api/cache"
	"github.com/ovh/tat/api/group"
	"github.com/ovh/tat/api/hook"
	"github.com/ovh/tat/api/message"
	"github.com/ovh/tat/api/store"
	"github.com/ovh/tat/api/topic"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var mainCmd = &cobra.Command{
	Use:   "tat",
	Short: "Run TAT Engine",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("tat")
		viper.AutomaticEnv()

		initLog()

		router := gin.New()
		router.Use(tatRecovery)

		if !viper.GetBool("production") {
			router.Use(gin.Logger())
		}

		if viper.GetString("tat_log_level") != "" {
			switch viper.GetString("tat_log_level") {
			case "debug":
				log.SetLevel(log.DebugLevel)
			case "info":
				log.SetLevel(log.InfoLevel)
			case "error":
				log.SetLevel(log.ErrorLevel)
			}
		}

		// Add a ginrus middleware, which:
		//   - Logs all requests, like a combined access and error log.
		//   - Logs to stdout.
		//   - RFC3339 with UTC time format.
		router.Use(ginrus(log.StandardLogger(), time.RFC3339, true))

		router.Use(cors.Middleware(cors.Config{
			Origins:         "*",
			Methods:         "GET, PUT, POST, DELETE",
			RequestHeaders:  "Origin, Authorization, Content-Type, Accept, Tat_Password, Tat_Username",
			ExposedHeaders:  "Tat_Password, Tat_Username",
			MaxAge:          50 * time.Second,
			Credentials:     true,
			ValidateHeaders: false,
		}))

		if err := store.NewStore(); err != nil {
			log.Fatalf("Error trying to reach mongoDB. Please check your Tat Configuration and access to your MongoDB. Err: %s", err.Error())
		}
		topic.InitDB()
		group.InitDB()
		message.InitDB()

		routerRoot := router.Group("")

		initRoutesGroups(routerRoot, CheckPassword())
		initRoutesMessages(routerRoot, CheckPassword())
		initRoutesPresences(routerRoot, CheckPassword())
		initRoutesTopics(routerRoot, CheckPassword())
		initRoutesUsers(routerRoot, CheckPassword())
		initRoutesStats(routerRoot, CheckPassword())
		initRoutesSystem(routerRoot, CheckPassword())
		hook.InitHooks()
		defer hook.CloseHooks()

		s := &http.Server{
			Addr:           ":" + viper.GetString("listen_port"),
			Handler:        router,
			ReadTimeout:    time.Duration(viper.GetInt("read_timeout")) * time.Second,
			WriteTimeout:   time.Duration(viper.GetInt("write_timeout")) * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		log.Infof("TAT is running on %s", viper.GetString("listen_port"))
		cache.TestInstanceAtStartup()

		if err := s.ListenAndServe(); err != nil {
			log.Info("Error while running ListenAndServe: %s", err.Error())
		}
	},
}

var versionNewLine bool

// The version command prints this service.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version.",
	Long:  "The version of Tat Engine.",
	Run: func(cmd *cobra.Command, args []string) {
		if versionNewLine {
			fmt.Println(tat.Version)
		} else {
			fmt.Print(tat.Version)
		}
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&versionNewLine, "versionNewLine", "", true, "New line after version number")
	mainCmd.AddCommand(versionCmd)

	flags := mainCmd.Flags()

	flags.Bool("production", false, "Production mode")
	viper.BindPFlag("production", flags.Lookup("production"))

	flags.Bool("no-smtp", false, "No SMTP mode")
	viper.BindPFlag("no_smtp", flags.Lookup("no-smtp"))

	flags.String("tat-log-level", "", "Tat Log Level: debug, info or warn")
	viper.BindPFlag("tat_log_level", flags.Lookup("tat-log-level"))

	flags.String("listen-port", "8080", "Tat Engine Listen Port")
	viper.BindPFlag("listen_port", flags.Lookup("listen-port"))

	flags.String("exposed-scheme", "http", "Tat URI Scheme http or https exposed to client")
	viper.BindPFlag("exposed_scheme", flags.Lookup("exposed-scheme"))

	flags.String("exposed-host", "localhost", "Tat Engine Hostname exposed to client")
	viper.BindPFlag("exposed_host", flags.Lookup("exposed-host"))

	flags.String("exposed-port", "8080", "Tat Engine Port exposed to client")
	viper.BindPFlag("exposed_port", flags.Lookup("exposed-port"))

	flags.String("exposed-path", "", "Tat Engine Path exposed to client, ex: host:port/tat/engine /tat/engine is exposed path")
	viper.BindPFlag("exposed_path", flags.Lookup("exposed-path"))

	flags.String("db-addr", "127.0.0.1:27017", "Address of the mongodb server")
	viper.BindPFlag("db_addr", flags.Lookup("db-addr"))

	flags.String("db-user", "", "User to authenticate with the mongodb server. If \"false\", db-user is not used")
	viper.BindPFlag("db_user", flags.Lookup("db-user"))

	flags.String("db-password", "", "Password to authenticate with the mongodb server. If \"false\", db-password is not used")
	viper.BindPFlag("db_password", flags.Lookup("db-password"))

	flags.String("db-rs-tags", "", "Link hostname with tag on mongodb replica set - Optional: hostnameA:tagName:value,hostnameB:tagName:value. If \"false\", db-rs-tags is not used")
	viper.BindPFlag("db_rs_tags", flags.Lookup("db-rs-tags"))

	flags.String("smtp-host", "", "SMTP Host")
	viper.BindPFlag("smtp_host", flags.Lookup("smtp-host"))

	flags.String("smtp-port", "", "SMTP Port")
	viper.BindPFlag("smtp_port", flags.Lookup("smtp-port"))

	flags.Bool("smtp-tls", false, "SMTP TLS")
	viper.BindPFlag("smtp_tls", flags.Lookup("smtp-tls"))

	flags.String("smtp-user", "", "SMTP Username")
	viper.BindPFlag("smtp_user", flags.Lookup("smtp-user"))

	flags.String("smtp-password", "", "SMTP Password")
	viper.BindPFlag("smtp_password", flags.Lookup("smtp-password"))

	flags.String("smtp-from", "", "SMTP From")
	viper.BindPFlag("smtp_from", flags.Lookup("smtp-from"))

	flags.String("allowed-domains", "", "Users have to use theses emails domains. Empty: no-restriction. Ex: --allowed-domains=domainA.org,domainA.com")
	viper.BindPFlag("allowed_domains", flags.Lookup("allowed-domains"))

	flags.String("default-group", "", "Default Group for new user")
	viper.BindPFlag("default_group", flags.Lookup("default-group"))

	flags.Bool("username-from-email", false, "Username are extracted from first part of email. first.lastame@domainA.org -> username: first.lastname")
	viper.BindPFlag("username_from_email", flags.Lookup("username-from-email"))

	flags.String("header-trust-username", "", "Header Trust Username: for example, if X-Remote-User and X-Remote-User received in header -> auto accept user without testing tat_password. Use it with precaution")
	viper.BindPFlag("header_trust_username", flags.Lookup("header-trust-username"))

	flags.String("trusted-usernames-emails-fullnames", "", "Tuples trusted username / email / fullname. Example: username:email:Firstname1_Fullname1,username2:email2:Firstname2_Fullname2")
	viper.BindPFlag("trusted_usernames_emails_fullnames", flags.Lookup("trusted-usernames-emails-fullnames"))

	flags.String("default-domain", "", "Default domains for mail for trusted username")
	viper.BindPFlag("default_domain", flags.Lookup("default-domain"))

	flags.Int("read-timeout", 50, "Read Timeout in seconds")
	viper.BindPFlag("read_timeout", flags.Lookup("read-timeout"))

	flags.Int("write-timeout", 50, "Write Timeout in seconds")
	viper.BindPFlag("write_timeout", flags.Lookup("write-timeout"))

	flags.Int("db-socket-timeout", 40, "Session DB Socket Timeout in seconds")
	viper.BindPFlag("db_socket_timeout", flags.Lookup("db-socket-timeout"))

	flags.String("redis-hosts", "", "Optional - Used for Cache - Redis hosts (comma separated for cluster)")
	viper.BindPFlag("redis_hosts", flags.Lookup("redis-hosts"))

	flags.String("redis-master", "", "Redis master name")
	viper.BindPFlag("redis_master", flags.Lookup("redis-master"))

	flags.String("redis-sentinels", "", "Redis sentinels (comma separated)")
	viper.BindPFlag("redis_sentinels", flags.Lookup("redis-sentinels"))

	flags.String("redis-password", "", "Optional - Used for Cache - Redis password")
	viper.BindPFlag("redis_password", flags.Lookup("redis-password"))

	flags.Int("db-ensure-safe-db-write", -1, "Min # of servers to ack before write success")
	viper.BindPFlag("db_ensure_safe_db_write", flags.Lookup("db-ensure-safe-db-write"))

	flags.String("webhooks-allowed-path", "", "Empty: no-restriction. Ex: --webhooks-allowed-path=https://urlA/,https://urlB/")
	viper.BindPFlag("webhooks_allowed_path", flags.Lookup("webhooks-allowed-path"))

	flags.Int("hooks-max-errors", 10, "Max errors before Tat set a hook as disabled")
	viper.BindPFlag("hooks_max_errors", flags.Lookup("hooks-max-errors"))

	flags.Bool("webhooks-enabled", true, "True for enabling webhook")
	viper.BindPFlag("webhooks_enabled", flags.Lookup("webhooks-enabled"))

	flags.String("tat2xmpp-url", "", "Empty: no-restriction. Ex: --tat2xmpp-url=https://urlA/,https://urlB/")
	viper.BindPFlag("tat2xmpp_url", flags.Lookup("tat2xmpp-url"))

	flags.String("tat2xmpp-key", "", "Ex: --tat2xmpp-key=KeyTat2XMPP")
	viper.BindPFlag("tat2xmpp_key", flags.Lookup("tat2xmpp-key"))

	flags.String("tat2xmpp-username", "tat.system.xmpp", "Empty: no-restriction. Ex: --tat2xmpp-username=tat.system.xmpp")
	viper.BindPFlag("tat2xmpp_username", flags.Lookup("tat2xmpp-username"))

	flags.String("kafka-broker-addresses", "", "Ex: --kafka-broker-addresses=host:port,host2:port2")
	viper.BindPFlag("kafka_broker_addresses", flags.Lookup("kafka-broker-addresses"))

	flags.String("kafka-client-id", "", "Ex: --kafka-client-id=xxxx-yyyy")
	viper.BindPFlag("kafka_client_id", flags.Lookup("kafka-client-id"))

	flags.String("graylog-protocol", "", "Ex: --graylog-protocol=xxxx-yyyy")
	viper.BindPFlag("graylog_protocol", flags.Lookup("graylog-protocol"))

	flags.String("graylog-server-name", "", "Ex: --graylog-server-name=xxxx-yyyy")
	viper.BindPFlag("graylog_server_name", flags.Lookup("graylog-server-name"))

	flags.String("graylog-host", "", "Ex: --graylog-host=xxxx-yyyy")
	viper.BindPFlag("graylog_host", flags.Lookup("graylog-host"))

	flags.String("graylog-port", "", "Ex: --graylog-port=12202")
	viper.BindPFlag("graylog_port", flags.Lookup("graylog-port"))

	flags.String("graylog-extra-key", "", "Ex: --graylog-extra-key=xxxx-yyyy")
	viper.BindPFlag("graylog_extra_key", flags.Lookup("graylog-extra-key"))

	flags.String("graylog-extra-value", "", "Ex: --graylog-extra-value=xxxx-yyyy")
	viper.BindPFlag("graylog_extra_value", flags.Lookup("graylog-extra-value"))

	flags.String("log-field-app-id", "", "Ex: --log_field_app_id=xxxx-yyyy")
	viper.BindPFlag("log_field_app_id", flags.Lookup("log-field-app-id"))
}

func main() {
	mainCmd.Execute()
}
