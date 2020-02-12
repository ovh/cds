package sdk

// UserTimelineFilter represents user_timeline table
type UserTimelineFilter struct {
	AuthenticatedUserID string         `json:"-" db:"authentified_user_id"`
	Filter              TimelineFilter `json:"filter" db:"-"`
}

// TimelineFilter represents a user filter for the cds timeline
type TimelineFilter struct {
	Projects []ProjectFilter `json:"projects"`
}

// ProjectFilter represents filter on a project
type ProjectFilter struct {
	Key           string   `json:"key"`
	WorkflowNames []string `json:"workflow_names"`
}
