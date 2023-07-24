package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

const (
	envFlagPrefix = "cds_"
	flagConfig    = "config"

	// TODO: the flag below will be removed
	flagBaseDir             = "basedir"
	flagBookedWorkflowJobID = "booked-workflow-job-id"
	flagRunJobID            = "run-job-id"
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
	flags.String(flagConfig, "", "base64 encoded json configuration")

	// TODO: the flag below will be removed
	flags.String(flagBaseDir, "", "This directory (default TMPDIR os environment var) will contains worker working directory and temporary files")
	flags.Int64(flagBookedWorkflowJobID, 0, "Booked Workflow job id")
	flags.String(flagRunJobID, "", "Run job ID")
	flags.String(flagGraylogProtocol, "", "Ex: --graylog-protocol=xxxx-yyyy")
	flags.String(flagGraylogHost, "", "Ex: --graylog-host=xxxx-yyyy")
	flags.String(flagGraylogPort, "", "Ex: --graylog-port=12202")
	flags.String(flagGraylogExtraKey, "", "Ex: --graylog-extra-key=xxxx-yyyy")
	flags.String(flagGraylogExtraValue, "", "Ex: --graylog-extra-value=xxxx-yyyy")
	flags.String(flagLogLevel, "notice", "Log Level: debug, info, notice, warning, error")
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

func initFromConfig(ctx context.Context, cfg *workerruntime.WorkerConfig, w *internal.CurrentWorker) error {
	cfg.Log.GraylogFieldCDSVersion = sdk.VERSION
	cfg.Log.GraylogFieldCDSOS = sdk.GOOS
	cfg.Log.GraylogFieldCDSArch = sdk.GOARCH
	cfg.Log.GraylogFieldCDSServiceType = "worker"

	cdslog.Initialize(ctx, &cfg.Log)

	fs := afero.NewOsFs()
	if cfg.Basedir == "" {
		cfg.Basedir = os.TempDir()
	}
	log.Debug(ctx, "creating basedir %s", cfg.Basedir)
	if err := fs.MkdirAll(cfg.Basedir, os.FileMode(0755)); err != nil {
		return errors.Errorf("unable to setup worker basedir %q: %+v", cfg.Basedir, err)
	}
	os.Setenv("BASEDIR", cfg.Basedir)
	os.Setenv("HATCHERY_NAME", cfg.HatcheryName)
	os.Setenv("HATCHERY_WORKER", cfg.Name)
	if cfg.Region != "" {
		os.Setenv("HATCHERY_REGION", cfg.Region)
	}
	if cfg.Model != "" {
		os.Setenv("HATCHERY_MODEL", cfg.Model)
	}
	for k, v := range cfg.InjectEnvVars {
		if v == "" {
			continue
		}
		os.Setenv(k, v)
	}

	return w.Init(cfg, afero.NewBasePathFs(fs, cfg.Basedir))
}

func initFromFlags(cmd *cobra.Command) (*workerruntime.WorkerConfig, error) {
	if base64config := FlagString(cmd, flagConfig); base64config != "" {
		btes, err := base64.StdEncoding.DecodeString(base64config)
		if err != nil {
			return nil, errors.Errorf("unable to decode config: %v", err)
		}
		var cfg workerruntime.WorkerConfig
		if err := sdk.JSONUnmarshal(btes, &cfg); err != nil {
			return nil, errors.Errorf("unable to parse config: %v", err)
		}
		return &cfg, nil
	}

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

	hatcheryName := FlagString(cmd, flagHatcheryName)
	apiEndpoint := FlagString(cmd, flagAPI)
	if apiEndpoint == "" {
		return nil, errors.New("--api not provided, aborting.")
	}

	token := FlagString(cmd, flagToken)
	if token == "" {
		return nil, errors.New("--token not provided, aborting.")
	}

	basedir, err := filepath.EvalSymlinks(basedir)
	if err != nil {
		return nil, errors.Errorf("symlink error: %v", err)
	}

	return &workerruntime.WorkerConfig{
		Name:                givenName,
		Basedir:             basedir,
		HatcheryName:        hatcheryName,
		APIToken:            token,
		APIEndpoint:         apiEndpoint,
		APIEndpointInsecure: FlagBool(cmd, flagInsecure),
		Model:               FlagString(cmd, flagModel),
		Log: cdslog.Conf{
			Level:             FlagString(cmd, flagLogLevel),
			GraylogProtocol:   FlagString(cmd, flagGraylogProtocol),
			GraylogHost:       FlagString(cmd, flagGraylogHost),
			GraylogPort:       FlagString(cmd, flagGraylogPort),
			GraylogExtraKey:   FlagString(cmd, flagGraylogExtraKey),
			GraylogExtraValue: FlagString(cmd, flagGraylogExtraValue),
		},
	}, nil
}
