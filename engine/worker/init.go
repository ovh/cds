package main

import (
	"os"
	"strconv"

	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/ovh/cds/sdk/log"
)

func initViper() {
	viper.SetEnvPrefix("cds")
	viper.AutomaticEnv()

	log.Initialize(&log.Conf{
		Level:             viper.GetString("log_level"),
		GraylogProtocol:   viper.GetString("graylog_protocol"),
		GraylogHost:       viper.GetString("graylog_host"),
		GraylogPort:       viper.GetString("graylog_port"),
		GraylogExtraKey:   viper.GetString("graylog_extra_key"),
		GraylogExtraValue: viper.GetString("graylog_extra_value"),
	})

	var errN error
	name, errN = os.Hostname()
	if errN != nil {
		log.Error("Cannot retrieve hostname: %s", errN)
		os.Exit(1)
	}

	hatchS := viper.GetString("hatchery")
	var errH error
	hatchery, errH = strconv.ParseInt(hatchS, 10, 64)
	if errH != nil {
		log.Error("WARNING: Invalid hatchery ID (%s)", errH)
		os.Exit(2)
	}

	// could be empty
	hatcheryName = viper.GetString("hatchery_name")

	api = viper.GetString("api")
	if api == "" {
		log.Error("--api not provided, aborting.")
		os.Exit(3)
	}

	key = viper.GetString("key")
	if key == "" {
		log.Error("--key not provided, aborting.")
		os.Exit(4)
	}

	givenName := viper.GetString("name")
	if givenName != "" {
		name = givenName
	}
	status.Name = name

	model = int64(viper.GetInt("model"))
	status.Model = model
}

func initServer() {
	port, err := server()
	if err != nil {
		log.Error("cannot bind port for worker export: %s", err)
		os.Exit(1)
	}
	exportport = port
}

type grpcCreds struct {
	Name, Token string
}

// GetRequestMetadata gets the request metadata as a map from a grpcCreds.
func (c *grpcCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"name":  c.Name,
		"token": c.Token,
	}, nil
}

// RequireTransportSecurity indicates whether the credentials requires transport security.
func (c *grpcCreds) RequireTransportSecurity() bool {
	return !viper.GetBool("grpc_insecure")
}

func initGRPCConn() {
	grpcAddress = viper.GetString("grpc_api")

	if grpcAddress != "" {
		opts := []grpc.DialOption{grpc.WithPerRPCCredentials(
			&grpcCreds{
				Name:  name,
				Token: WorkerID,
			})}

		if viper.GetBool("grpc_insecure") {
			opts = append(opts, grpc.WithInsecure())
		}

		var err error
		grpcConn, err = grpc.Dial(grpcAddress, opts...)
		if err != nil {
			log.Error("Unable to connect to GRPC API %s: %s", grpcAddress, err)
		}
	}
}
