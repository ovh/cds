package sdk

import (
	"fmt"
)

type WebsocketV2FilterType string

const (
	WebsocketV2FilterTypeGlobal             WebsocketV2FilterType = "global"
	WebsocketV2FilterTypeProject            WebsocketV2FilterType = "project"
	WebsocketV2FilterTypeProjectPurgeReport WebsocketV2FilterType = "project-purge-report"
	WebsocketV2FilterTypeProjectRuns        WebsocketV2FilterType = "project-runs"
	WebsocketV2FilterTypeQueue              WebsocketV2FilterType = "queue"
)

func (f WebsocketV2FilterType) IsValid() bool {
	switch f {
	case WebsocketV2FilterTypeGlobal,
		WebsocketV2FilterTypeProject,
		WebsocketV2FilterTypeProjectRuns,
		WebsocketV2FilterTypeProjectPurgeReport,
		WebsocketV2FilterTypeQueue:
		return true
	}
	return false
}

type WebsocketV2Filters []WebsocketV2Filter

type WebsocketV2Filter struct {
	Type              WebsocketV2FilterType `json:"type"`
	ProjectKey        string                `json:"project_key"`
	ProjectRunsParams string                `json:"project_runs_params"`
	PurgeReportID     string                `json:"purge_report_id"`
}

// Key generates the unique key associated to given filter.
func (f WebsocketV2Filter) Key() string {
	switch f.Type {
	case WebsocketV2FilterTypeProject:
		return fmt.Sprintf("%s-%s", f.Type, f.ProjectKey)
	case WebsocketV2FilterTypeProjectPurgeReport:
		return fmt.Sprintf("%s-%s-%s", f.Type, f.ProjectKey, f.PurgeReportID)
	case WebsocketV2FilterTypeProjectRuns:
		return fmt.Sprintf("%s-%s", f.Type, f.ProjectKey)
	default:
		return string(f.Type)
	}
}

// IsValid return an error if given filter is not valid.
func (f WebsocketV2Filter) IsValid() error {
	if !f.Type.IsValid() {
		return NewErrorFrom(ErrWrongRequest, "invalid or empty given filter type: %s", f.Type)
	}

	switch f.Type {
	case WebsocketV2FilterTypeProject,
		WebsocketV2FilterTypeProjectRuns:
		if f.ProjectKey == "" {
			return NewErrorFrom(ErrWrongRequest, "missing project key")
		}
	case WebsocketV2FilterTypeProjectPurgeReport:
		if f.ProjectKey == "" || f.PurgeReportID == "" {
			return NewErrorFrom(ErrWrongRequest, "missing project key or report id")
		}
	}

	return nil
}

type WebsocketV2Event struct {
	Status string      `json:"status"`
	Error  string      `json:"error"`
	Event  FullEventV2 `json:"event"`
}
