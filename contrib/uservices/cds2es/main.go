package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/itsjamie/gin-cors"
	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
)

// VERSION ...
const VERSION = "0.1.0"

func main() {
	configPath := flag.String("f", "", "path to the config file")
	flag.Parse()

	if *configPath == "" {
		fmt.Println("Usage: cds2es -f configFile.yml")
		os.Exit(1)
	}

	configData, err := ioutil.ReadFile(*configPath)
	if err != nil {
		fmt.Printf("Unable to read configuration: %s\n", err)
		os.Exit(2)
	}
	var conf Configuration
	if err := toml.Unmarshal(configData, &conf); err != nil {
		fmt.Printf("Unable to unmarshal configuration: %s", err)
		os.Exit(3)
	}

	switch conf.Debug.LogLevel {
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

	c := make(chan sdk.Event)
	go consume(conf, c)
	go sendToES(conf, c)

	log.Infof("Running cds2es on %s", viper.GetString("listen_port"))
	if err := s.ListenAndServe(); err != nil {
		log.Errorf("Error while running ListenAndServe: %s", err.Error())
	}
}
