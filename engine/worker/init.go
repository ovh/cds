package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/facebookgo/httpcontrol"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

const (
	envFlagPrefix           = "cds_"
	flagSingleUse           = "single-use"
	flagAutoUpdate          = "auto-update"
	flagFromGithub          = "from-github"
	flagForceExit           = "force-exit"
	flagBaseDir             = "basedir"
	flagTTL                 = "ttl"
	flagBookedPBJobID       = "booked-pb-job-id"
	flagBookedWorkflowJobID = "booked-workflow-job-id"
	flagBookedJobID         = "booked-job-id"
	flagGRPCAPI             = "grpc-api"
	flagGRPCInsecure        = "grpc-insecure"
	flagGraylogProtocol     = "graylog-protocol"
	flagGraylogHost         = "graylog-host"
	flagGraylogPort         = "graylog-port"
	flagGraylogExtraKey     = "graylog-extra-key"
	flagGraylogExtraValue   = "graylog-extra-value"
	flagLogLevel            = "log-level"
	flagAPI                 = "api"
	flagInsecure            = "insecure"
	flagToken               = "token"
	flagName                = "name"
	flagModel               = "model"
	flagHatchery            = "hatchery"
	flagHatcheryName        = "hatchery-name"
)

func initFlagsRun(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.Bool(flagSingleUse, false, "Exit after executing an action")
	flags.Bool(flagAutoUpdate, false, "Auto update worker binary from CDS API")
	flags.Bool(flagFromGithub, false, "Update binary from latest github release")
	flags.Bool(flagForceExit, false, "If single_use=true, force exit. This is useful if it's spawned by an Hatchery (default: worker wait 30min for being killed by hatchery)")
	flags.String(flagBaseDir, "", "This directory (default TMPDIR os environment var) will contains worker working directory and temporary files")
	flags.Int(flagTTL, 30, "Worker time to live (minutes)")
	flags.Int64(flagBookedPBJobID, 0, "Booked Pipeline Build job id")
	flags.Int64(flagBookedWorkflowJobID, 0, "Booked Workflow job id")
	flags.Int64(flagBookedJobID, 0, "Booked job id")
	flags.String(flagGRPCAPI, "", "CDS GRPC tcp address")
	flags.Bool(flagGRPCInsecure, false, "Disable GRPC TLS encryption")
	flags.String(flagGraylogProtocol, "", "Ex: --graylog-protocol=xxxx-yyyy")
	flags.String(flagGraylogHost, "", "Ex: --graylog-host=xxxx-yyyy")
	flags.String(flagGraylogPort, "", "Ex: --graylog-port=12202")
	flags.String(flagGraylogExtraKey, "", "Ex: --graylog-extra-key=xxxx-yyyy")
	flags.String(flagGraylogExtraValue, "", "Ex: --graylog-extra-value=xxxx-yyyy")
	flags.String(flagLogLevel, "notice", "Log Level: debug, info, notice, warning, critical")
	flags.String(flagAPI, "", "URL of CDS API")
	flags.Bool(flagInsecure, false, `(SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.`)
	flags.String(flagToken, "", "CDS Token")
	flags.String(flagName, "", "Name of worker")
	flags.Int(flagModel, 0, "Model of worker")
	flags.Int(flagHatchery, 0, "Hatchery ID spawing worker")
	flags.String(flagHatcheryName, "", "Hatchery Name spawing worker")
}

// FlagBool replaces viper.GetBool
func FlagBool(cmd *cobra.Command, key string) bool {
	envKey := envFlagPrefix + key
	envKey = strings.Replace(envKey, "-", "_", -1)
	envKey = strings.ToUpper(envKey)

	if os.Getenv(envKey) != "" {
		if os.Getenv(envKey) == "true" || os.Getenv(envKey) == "1" {
			return true
		}
	} else if cmd.Flag(key) != nil && cmd.Flag(key).Value.String() == "true" {
		return true
	}

	return false
}

// FlagString replaces viper.GetString
func FlagString(cmd *cobra.Command, key string) string {
	envKey := envFlagPrefix + key
	envKey = strings.Replace(envKey, "-", "_", -1)
	envKey = strings.ToUpper(envKey)

	if os.Getenv(envKey) != "" {
		return os.Getenv(envKey)
	}

	return cmd.Flag(key).Value.String()
}

// FlagInt replaces viper.GetInt
func FlagInt(cmd *cobra.Command, key string) int {
	envKey := envFlagPrefix + key
	envKey = strings.Replace(envKey, "-", "_", -1)
	envKey = strings.ToUpper(envKey)

	if os.Getenv(envKey) != "" {
		i, _ := strconv.Atoi(os.Getenv(envKey))
		return i
	}

	i, _ := strconv.Atoi(cmd.Flag(key).Value.String())
	return i
}

// FlagInt64 replaces viper.GetInt64
func FlagInt64(cmd *cobra.Command, key string) int64 {
	envKey := envFlagPrefix + key
	envKey = strings.Replace(envKey, "-", "_", -1)
	envKey = strings.ToUpper(envKey)

	if os.Getenv(envKey) != "" {
		i, _ := strconv.ParseInt(os.Getenv(envKey), 10, 64)
		return i
	}

	i, _ := strconv.ParseInt(cmd.Flag(key).Value.String(), 10, 64)
	return i
}

func initFlags(cmd *cobra.Command, w *currentWorker) {
	var errN error
	var hostname string
	hostname, errN = os.Hostname()
	if errN != nil {
		// no log, no need to exit here
		// we recheck os.Hostname cmd below, when log are initialized
		fmt.Printf("Cannot retrieve hostname: %s\n", errN)
	}

	w.status.Name = hostname
	givenName := FlagString(cmd, flagName)
	if givenName != "" {
		w.status.Name = givenName
	}

	log.Initialize(&log.Conf{
		Level:                  FlagString(cmd, flagLogLevel),
		GraylogProtocol:        FlagString(cmd, flagGraylogProtocol),
		GraylogHost:            FlagString(cmd, flagGraylogHost),
		GraylogPort:            FlagString(cmd, flagGraylogPort),
		GraylogExtraKey:        FlagString(cmd, flagGraylogExtraKey),
		GraylogExtraValue:      FlagString(cmd, flagGraylogExtraValue),
		GraylogFieldCDSVersion: sdk.VERSION,
		GraylogFieldCDSName:    w.status.Name,
	})

	// recheck hostname and send log if error
	if hostname == "" {
		if _, err := os.Hostname(); err != nil {
			log.Error("Cannot retrieve hostname: %v", err)
			os.Exit(1)
		}
	}

	hatchS := FlagString(cmd, flagHatchery)
	var errH error
	w.hatchery.id, errH = strconv.ParseInt(hatchS, 10, 64)
	if errH != nil {
		log.Error("WARNING: Invalid hatchery ID (%v)", errH)
		os.Exit(2)
	}

	// could be empty
	w.hatchery.name = FlagString(cmd, flagHatcheryName)
	w.apiEndpoint = FlagString(cmd, flagAPI)
	if w.apiEndpoint == "" {
		log.Error("--api not provided, aborting.")
		os.Exit(3)
	}

	w.token = FlagString(cmd, flagToken)
	if w.token == "" {
		log.Error("--token not provided, aborting.")
		os.Exit(4)
	}

	w.model = sdk.Model{ID: int64(FlagInt(cmd, flagModel))}

	w.basedir = FlagString(cmd, flagBaseDir)
	if w.basedir == "" {
		w.basedir = os.TempDir()
	}
	w.bookedPBJobID = FlagInt64(cmd, flagBookedPBJobID)
	w.bookedWJobID = FlagInt64(cmd, flagBookedWorkflowJobID)

	w.client = cdsclient.NewWorker(w.apiEndpoint, w.status.Name, &http.Client{
		Timeout: time.Second * 10,
		Transport: &httpcontrol.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: FlagBool(cmd, flagInsecure)},
		},
	})

	w.autoUpdate = FlagBool(cmd, flagAutoUpdate)
	w.singleUse = FlagBool(cmd, flagSingleUse)
	w.grpc.address = FlagString(cmd, flagGRPCAPI)
	w.grpc.insecure = FlagBool(cmd, flagGRPCInsecure)
}

func (w *currentWorker) initServer(c context.Context) {
	port, err := w.serve(c)
	if err != nil {
		log.Error("cannot bind port for worker export: %v", err)
		os.Exit(1)
	}
	w.exportPort = port
}

type grpcCreds struct {
	Insecure    bool
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
	return !c.Insecure
}

func (w *currentWorker) initGRPCConn() {
	if w.grpc.address != "" {
		opts := []grpc.DialOption{grpc.WithPerRPCCredentials(
			&grpcCreds{
				Insecure: w.grpc.insecure,
				Name:     w.status.Name,
				Token:    w.id,
			})}

		opts = append(opts, grpc.WithInsecure())

		var err error
		w.grpc.conn, err = grpc.Dial(w.grpc.address, opts...)
		if err != nil {
			log.Error("Unable to connect to GRPC API %s: %v", w.grpc.address, err)
		}
	}
}
