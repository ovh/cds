package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/afero"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	envFlagPrefix           = "cds_"
	flagBaseDir             = "basedir"
	flagBookedWorkflowJobID = "booked-workflow-job-id"
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
	flagHatcheryName        = "hatchery-name"
)

func initFlagsRun(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.String(flagBaseDir, "", "This directory (default TMPDIR os environment var) will contains worker working directory and temporary files")
	flags.Int64(flagBookedWorkflowJobID, 0, "Booked Workflow job id")
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
	flags.String(flagModel, "", "Model of worker")
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

func initFromFlags(cmd *cobra.Command, w *internal.CurrentWorker) {
	var errN error
	var hostname string
	hostname, errN = os.Hostname()
	if errN != nil {
		// no log, no need to exit here
		// we recheck os.Hostname cmd below, when log are initialized
		fmt.Printf("Cannot retrieve hostname: %s\n", errN)
	}

	givenName := hostname
	if FlagString(cmd, flagName) != "" {
		givenName = FlagString(cmd, flagName)
	}

	basedir := FlagString(cmd, flagBaseDir)
	if basedir == "" {
		basedir = os.TempDir()
	}

	log.Initialize(context.Background(), &log.Conf{
		Level:                      FlagString(cmd, flagLogLevel),
		GraylogProtocol:            FlagString(cmd, flagGraylogProtocol),
		GraylogHost:                FlagString(cmd, flagGraylogHost),
		GraylogPort:                FlagString(cmd, flagGraylogPort),
		GraylogExtraKey:            FlagString(cmd, flagGraylogExtraKey),
		GraylogExtraValue:          FlagString(cmd, flagGraylogExtraValue),
		GraylogFieldCDSVersion:     sdk.VERSION,
		GraylogFieldCDSOS:          sdk.GOOS,
		GraylogFieldCDSArch:        sdk.GOARCH,
		GraylogFieldCDSServiceName: givenName,
		GraylogFieldCDSServiceType: "worker",
	})

	hatcheryName := FlagString(cmd, flagHatcheryName)
	apiEndpoint := FlagString(cmd, flagAPI)
	if apiEndpoint == "" {
		log.Error(context.TODO(), "--api not provided, aborting.")
		os.Exit(3)
	}

	token := FlagString(cmd, flagToken)
	if token == "" {
		log.Error(context.TODO(), "--token not provided, aborting.")
		os.Exit(4)
	}

	basedir, err := filepath.EvalSymlinks(basedir)
	if err != nil {
		log.Error(context.Background(), "symlink error: %v", err)
		os.Exit(6)
	}

	fs := afero.NewOsFs()
	log.Debug("creating basedir %s", basedir)
	if err := fs.MkdirAll(basedir, os.FileMode(0755)); err != nil {
		log.Error(context.TODO(), "basedir error: %v", err)
		os.Exit(5)
	}

	if err := w.Init(givenName,
		hatcheryName,
		apiEndpoint,
		token,
		FlagString(cmd, flagModel),
		FlagBool(cmd, flagInsecure),
		afero.NewBasePathFs(fs, basedir)); err != nil {
		log.Error(context.TODO(), "Cannot init worker: %v", err)
		os.Exit(1)
	}
}
