package sdk

import (
	"time"
)

// Stats aggregates all CDS stats
type Stats struct {
	History []Week
}

// Week exposes what happened in a week timeframe
type Week struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`

	Builds               int64 `json:"builds_completed"`
	UnitTests            int64 `json:"unit_tests"`
	MaxBuildingWorkers   int64 `json:"max_building_worker"`
	MaxBuildingPipelines int64 `json:"max_building_pipeline"`

	Users    int64 `json:"period_total_users"`
	NewUsers int64 `json:"new_users"`

	Projects    int64 `json:"period_total_projects"`
	NewProjects int64 `json:"new_projects"`

	Applications    int64 `json:"period_total_applications"`
	NewApplications int64 `json:"new_applications"`

	Pipelines struct {
		Build   int64 `json:"build"`
		Testing int64 `json:"testing"`
		Deploy  int64 `json:"deploy"`
	} `json:"period_total_pipelines"`
	NewPipelines int64 `json:"new_pipelines"`

	RunnedPipelines struct {
		Build   int64 `json:"build"`
		Testing int64 `json:"testing"`
		Deploy  int64 `json:"deploy"`
	} `json:"runned_pipelines"`
}
