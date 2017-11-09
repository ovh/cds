package vsphere

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

// getGraylogGrpcEnv fetch the graylog and grpc configuration from viper and return environement variable in a slice
func (h *HatcheryVSphere) getGraylogGrpcEnv(model sdk.Model) []string {
	env := []string{}

	if h.Configuration().Provision.WorkerLogsOptions.Graylog.Host != "" {
		env = append(env, fmt.Sprintf("CDS_GRAYLOG_HOST=%s", h.Configuration().Provision.WorkerLogsOptions.Graylog.Host))
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.Port > 0 {
		env = append(env, fmt.Sprintf("export CDS_GRAYLOG_PORT=%d", h.Configuration().Provision.WorkerLogsOptions.Graylog.Port))
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey != "" {
		env = append(env, fmt.Sprintf("export CDS_GRAYLOG_EXTRA_KEY=%s", h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey))
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue != "" {
		env = append(env, fmt.Sprintf("export CDS_GRAYLOG_EXTRA_VALUE=%s", h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue))
	}

	if h.Configuration().API.GRPC.URL != "" && model.Communication == sdk.GRPC {
		env = append(env, fmt.Sprintf("export CDS_GRPC_API=%s", h.Configuration().API.GRPC.URL))
		env = append(env, fmt.Sprintf("export CDS_GRPC_INSECURE=%t", h.Configuration().API.GRPC.Insecure))
	}

	return env
}
