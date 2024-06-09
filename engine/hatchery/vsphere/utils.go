package vsphere

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

// getGraylogEnv fetch the graylog aconfiguration from viper and return environment variable in a slice
func (h *HatcheryVSphere) getGraylogEnv(model sdk.Model) []string {
	env := []string{}

	if h.Config.Provision.WorkerLogsOptions.Graylog.Host != "" {
		env = append(env, fmt.Sprintf("CDS_GRAYLOG_HOST=%s", h.Config.Provision.WorkerLogsOptions.Graylog.Host))
	}
	if h.Config.Provision.WorkerLogsOptions.Graylog.Port > 0 {
		env = append(env, fmt.Sprintf("export CDS_GRAYLOG_PORT=%d", h.Config.Provision.WorkerLogsOptions.Graylog.Port))
	}
	if h.Config.Provision.WorkerLogsOptions.Graylog.ExtraKey != "" {
		env = append(env, fmt.Sprintf("export CDS_GRAYLOG_EXTRA_KEY=%s", h.Config.Provision.WorkerLogsOptions.Graylog.ExtraKey))
	}
	if h.Config.Provision.WorkerLogsOptions.Graylog.ExtraValue != "" {
		env = append(env, fmt.Sprintf("export CDS_GRAYLOG_EXTRA_VALUE=%s", h.Config.Provision.WorkerLogsOptions.Graylog.ExtraValue))
	}

	return env
}
