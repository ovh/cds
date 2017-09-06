package vsphere

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
)

// getGraylogGrpcEnv fetch the graylog and grpc configuration from viper and return environement variable in a slice
func getGraylogGrpcEnv(model *sdk.Model) []string {
	env := []string{}

	if viper.GetString("worker_graylog_host") != "" {
		env = append(env, fmt.Sprintf("CDS_GRAYLOG_HOST=%s", viper.GetString("worker_graylog_host")))
	}
	if viper.GetString("worker_graylog_port") != "" {
		env = append(env, fmt.Sprintf("export CDS_GRAYLOG_PORT=%s", viper.GetString("worker_graylog_port")))
	}
	if viper.GetString("worker_graylog_extra_key") != "" {
		env = append(env, fmt.Sprintf("export CDS_GRAYLOG_EXTRA_KEY=%s", viper.GetString("worker_graylog_extra_key")))
	}
	if viper.GetString("worker_graylog_extra_value") != "" {
		env = append(env, fmt.Sprintf("export CDS_GRAYLOG_EXTRA_VALUE=%s", viper.GetString("worker_graylog_extra_value")))
	}

	if viper.GetString("grpc_api") != "" && model.Communication == sdk.GRPC {
		env = append(env, fmt.Sprintf("export CDS_GRPC_API=%s", viper.GetString("grpc_api")))
		env = append(env, fmt.Sprintf("export CDS_GRPC_INSECURE=%t", viper.GetBool("grpc_insecure")))
	}

	return env
}
