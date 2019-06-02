package elasticsearch

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (es *ES) GetStatus(name string) sdk.MonitoringStatusLine {
	if es.client == nil {
		return sdk.MonitoringStatusLine{
			Component: componentName,
			Value:     "disconnected",
			Status:    sdk.MonitoringStatusWarn,
		}
	} else {
		info, code, err := es.client.Ping(es.Endpoint).Do(context.Background())
		log.Debug("Metrics> ElasticSearch> Ping code: %v", code)
		line := sdk.MonitoringStatusLine{Component: componentName, Type: componentName}

		if err != nil {
			line.Status = sdk.MonitoringStatusWarn
			line.Value = fmt.Sprintf("no ping (%v)", err)
		} else if code >= 400 {
			line.Status = sdk.MonitoringStatusWarn
			line.Value = fmt.Sprintf("ping error (code:%d, err: %v)", code, err)
		} else {
			line.Status = sdk.MonitoringStatusOK
			line.Value = fmt.Sprintf("Version: %s", info.Version.Number)
		}

		return line
	}
}
