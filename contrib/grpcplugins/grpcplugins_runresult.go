package grpcplugins

import (
	"github.com/ovh/cds/sdk"
)

func ComputeRunResultHelmDetail(chartName, appVersion, chartVersion string) sdk.V2WorkflowRunResultDetail {
	return sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultHelmDetail{
			Name:         chartName,
			AppVersion:   appVersion,
			ChartVersion: chartVersion,
		},
	}
}

func ComputeRunResultPythonDetail(packageName string, version string, extension string) sdk.V2WorkflowRunResultDetail {
	return sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultPythonDetail{
			Name:      packageName,
			Version:   version,
			Extension: extension,
		},
	}
}
