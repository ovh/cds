package sdk

import "encoding/json"

// ApplicationOverview represents the overview of an application
type ApplicationOverview struct {
	Graphs  []ApplicationOverviewGraph      `json:"graphs,omitempty"`
	GitURL  string                          `json:"git_url,omitempty"`
	History map[string][]WorkflowRunSummary `json:"history,omitempty"`
}

// ApplicationOverviewGraph represents data pushed by CDS for metrics
type ApplicationOverviewGraph struct {
	Type  string            `json:"type"`
	Datas []json.RawMessage `json:"datas"`
}
