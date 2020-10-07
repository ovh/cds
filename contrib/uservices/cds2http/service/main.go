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
	Use: "cds2http",
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

		go do()
		log.Infof("Running cds2http on %s", viper.GetString("listen_port"))
		if err := s.ListenAndServe(); err != nil {
			log.Errorf("Error while running ListenAndServe: %s", err.Error())
		}
	},
}

func init() {
	flags := mainCmd.Flags()

	flags.String("log-level", "", "Log Level : debug, info or warn")
	viper.BindPFlag("log_level", flags.Lookup("log-level")) // nolint

	flags.String("listen-port", "8085", "Listen Port")
	viper.BindPFlag("listen_port", flags.Lookup("listen-port")) // nolint

	flags.String("event-kafka-broker-addresses", "", "Ex: --event-kafka-broker-addresses=host:port,host2:port2")
	viper.BindPFlag("event_kafka_broker_addresses", flags.Lookup("event-kafka-broker-addresses")) // nolint

	flags.String("event-kafka-topic", "", "Ex: --kafka-topic=your-kafka-topic")
	viper.BindPFlag("event_kafka_topic", flags.Lookup("event-kafka-topic")) // nolint

	flags.String("event-kafka-version", "", "Ex: --kafka-version=your-kafka-version")
	viper.BindPFlag("event_kafka_version", flags.Lookup("event-kafka-version")) // nolint

	flags.String("event-kafka-user", "", "Ex: --kafka-user=your-kafka-user")
	viper.BindPFlag("event_kafka_user", flags.Lookup("event-kafka-user")) // nolint

	flags.String("event-kafka-password", "", "Ex: --kafka-password=your-kafka-password")
	viper.BindPFlag("event_kafka_password", flags.Lookup("event-kafka-password")) // nolint

	flags.String("event-kafka-group", "", "Ex: --kafka-group=your-kafka-group")
	viper.BindPFlag("event_kafka_group", flags.Lookup("event-kafka-group")) // nolint

	flags.String("event-remote-url", "", "Ex: --event-remote-url=your-remote-url")
	viper.BindPFlag("event_remote_url", flags.Lookup("event-remote-url")) // nolint

	flags.Bool("force-dot", true, "If destination (except conference) does not contains '.' skip destination")
	viper.BindPFlag("force_dot", flags.Lookup("force-dot")) // nolint
}

func main() {
	mainCmd.Execute()
}
