package main

import (
	"os"
	"strconv"

	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

func initViper(w *currentWorker) {
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
	w.status.Name, errN = os.Hostname()
	if errN != nil {
		log.Error("Cannot retrieve hostname: %s", errN)
		os.Exit(1)
	}
	givenName := viper.GetString("name")
	if givenName != "" {
		w.status.Name = givenName
	}

	hatchS := viper.GetString("hatchery")
	var errH error
	w.hatchery.id, errH = strconv.ParseInt(hatchS, 10, 64)
	if errH != nil {
		log.Error("WARNING: Invalid hatchery ID (%s)", errH)
		os.Exit(2)
	}

	// could be empty
	w.hatchery.name = viper.GetString("hatchery_name")
	w.apiEndpoint = viper.GetString("api")
	if w.apiEndpoint == "" {
		log.Error("--api not provided, aborting.")
		os.Exit(3)
	}

	w.token = viper.GetString("key")
	if w.token == "" {
		log.Error("--key not provided, aborting.")
		os.Exit(4)
	}

	w.modelID = int64(viper.GetInt("model"))
	w.status.Model = w.modelID

	w.basedir = viper.GetString("basedir")
	if w.basedir == "" {
		w.basedir = os.TempDir()
	}
	w.bookedJobID = viper.GetInt64("booked_job_id")

	w.client = cdsclient.NewWorker(w.apiEndpoint)
}

func (w *currentWorker) initServer(c context.Context) {
	port, err := w.serve(c)
	if err != nil {
		log.Error("cannot bind port for worker export: %s", err)
		os.Exit(1)
	}
	w.exportPort = port
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

func (w *currentWorker) initGRPCConn() {
	w.grpc.address = viper.GetString("grpc_api")

	if w.grpc.address != "" {
		opts := []grpc.DialOption{grpc.WithPerRPCCredentials(
			&grpcCreds{
				Name:  w.status.Name,
				Token: w.id,
			})}

		if viper.GetBool("grpc_insecure") {
			opts = append(opts, grpc.WithInsecure())
		}

		var err error
		w.grpc.conn, err = grpc.Dial(w.grpc.address, opts...)
		if err != nil {
			log.Error("Unable to connect to GRPC API %s: %s", w.grpc.address, err)
		}
	}
}
