package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	cors "github.com/itsjamie/gin-cors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// VERSION ...
const VERSION = "0.1.0"

var mainCmd = &cobra.Command{
	Use: "cds2xmpp",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("cds")
		viper.AutomaticEnv()

		switch viper.GetString("log_level") {
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "error":
			log.SetLevel(log.WarnLevel)
			gin.SetMode(gin.ReleaseMode)
		default:
			log.SetLevel(log.DebugLevel)
		}

		router := gin.New()
		router.Use(gin.Recovery())

		router.Use(cors.Middleware(cors.Config{
			Origins:         "*",
			Methods:         "GET, PUT, POST, DELETE",
			RequestHeaders:  "Origin, Authorization, Content-Type, Accept",
			MaxAge:          50 * time.Second,
			Credentials:     true,
			ValidateHeaders: false,
		}))

		router.GET("/mon/ping", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"version": VERSION})
		})

		s := &http.Server{
			Addr:           ":" + viper.GetString("listen_port"),
			Handler:        router,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		var err error
		cdsbot, err = getBotClient()
		if err != nil {
			log.Fatalf("Error while initialize client err:%s", err)
		}

		go cdsbot.born()

		log.Infof("Running cds2xmpp on %s", viper.GetString("listen_port"))
		if err := s.ListenAndServe(); err != nil {
			log.Errorf("Error while running ListenAndServe: %s", err.Error())
		}

	},
}

func init() {
	flags := mainCmd.Flags()

	flags.String("log-level", "", "Log Level : debug, info or warn")
	viper.BindPFlag("log_level", flags.Lookup("log-level"))

	flags.String("listen-port", "8085", "Listen Port")
	viper.BindPFlag("listen_port", flags.Lookup("listen-port"))

	flags.String("event-kafka-broker-addresses", "", "Ex: --event-kafka-broker-addresses=host:port,host2:port2")
	viper.BindPFlag("event_kafka_broker_addresses", flags.Lookup("event-kafka-broker-addresses"))

	flags.String("event-kafka-topic", "", "Ex: --kafka-topic=your-kafka-topic")
	viper.BindPFlag("event_kafka_topic", flags.Lookup("event-kafka-topic"))

	flags.String("event-kafka-user", "", "Ex: --kafka-user=your-kafka-user")
	viper.BindPFlag("event_kafka_user", flags.Lookup("event-kafka-user"))

	flags.String("event-kafka-password", "", "Ex: --kafka-password=your-kafka-password")
	viper.BindPFlag("event_kafka_password", flags.Lookup("event-kafka-password"))

	flags.String("event-kafka-group", "", "Ex: --kafka-group=your-kafka-group")
	viper.BindPFlag("event_kafka_group", flags.Lookup("event-kafka-group"))

	flags.String("xmpp-server", "", "XMPP Server")
	viper.BindPFlag("xmpp_server", flags.Lookup("xmpp-server"))

	flags.String("xmpp-bot-jid", "cds@localhost", "XMPP Bot JID")
	viper.BindPFlag("xmpp_bot_jid", flags.Lookup("xmpp-bot-jid"))

	flags.String("xmpp-bot-password", "", "XMPP Bot Password")
	viper.BindPFlag("xmpp_bot_password", flags.Lookup("xmpp-bot-password"))

	flags.String("admin-cds2xmpp", "", "Admin cds2xmpp admina@jabber.yourdomain.net,adminb@jabber.yourdomain.net,")
	viper.BindPFlag("admin_cds2xmpp", flags.Lookup("admin-cds2xmpp"))

	flags.String("admin-conference", "", "CDS Admin conference cds@conference.jabber.yourdomain.net")
	viper.BindPFlag("admin_conference", flags.Lookup("admin-conference"))

	flags.Bool("xmpp-debug", false, "XMPP Debug")
	viper.BindPFlag("xmpp_debug", flags.Lookup("xmpp-debug"))

	flags.Bool("xmpp-notls", true, "XMPP No TLS")
	viper.BindPFlag("xmpp_notls", flags.Lookup("xmpp-notls"))

	flags.Bool("xmpp-starttls", false, "XMPP Start TLS")
	viper.BindPFlag("xmpp_starttls", flags.Lookup("xmpp-starttls"))

	flags.Bool("xmpp-session", true, "XMPP Session")
	viper.BindPFlag("xmpp_session", flags.Lookup("xmpp-session"))

	flags.Bool("force-dot", true, "If destination (except conference) does not contains '.' skip destination")
	viper.BindPFlag("force_dot", flags.Lookup("force-dot"))

	flags.Bool("xmpp-insecure-skip-verify", true, "XMPP InsecureSkipVerify")
	viper.BindPFlag("xmpp_insecure_skip_verify", flags.Lookup("xmpp-insecure-skip-verify"))

	flags.String("xmpp-default-hostname", "", "Default Hostname for user, enter your.jabber.net for @your.jabber.net")
	viper.BindPFlag("xmpp_default_hostname", flags.Lookup("xmpp-default-hostname"))

	flags.Int("xmpp-delay", 5, "Delay between two sent messages")
	viper.BindPFlag("xmpp_delay", flags.Lookup("xmpp-delay"))

	flags.String("more-help", "", "Text added on /cds help")
	viper.BindPFlag("more_help", flags.Lookup("more-help"))
}

func main() {
	mainCmd.Execute()
}
