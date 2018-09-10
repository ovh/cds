package sdk

import "encoding/json"

// ApplicationOverview represents the overview of an application
type ApplicationOverview struct {
	Graphs  []ApplicationOverviewGraph `json:"graphs"`
	GitURL  string                     `json:"git_url"`
	History map[string][]WorkflowRun   `json:"history"`
}

// ApplicationOverviewGraph represents data pushed by CDS for metrics
type ApplicationOverviewGraph struct {
	Type  string            `json:"type"`
	Datas []json.RawMessage `json:"datas"`
}
