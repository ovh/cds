package sdk

import "time"

const (
	MetricKeyVulnerability = "Vulnerability"
)

// Metric represent a CDS metric
type Metric struct {
	ProjectKey    string      `json:"project_key"`
	ApplicationID int64       `json:"application_id"`
	WorkflowID    int64       `json:"workflow_id"`
	Key           string      `json:"key"`
	Value         interface{} `json:"value"`
	Date          time.Time   `json:"timestamp"`
}

// MetricRequest represents the request to retrieve metrics
type MetricRequest struct {
	ProjectKey    string `json:"project_key"`
	ApplicationID int64  `json:"application_id"`
	WorkflowID    int64  `json:"workflow_id"`
	Key           string `json:"key"`
}
