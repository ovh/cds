package sdk

import "time"

const (
	MetricKeyVulnerability = "Vulnerability"
	MetricKeyUnitTest      = "UnitTest"
)

// Metric represent a CDS metric
type Metric struct {
	ProjectKey    string             `json:"project_key"`
	ApplicationID int64              `json:"application_id"`
	WorkflowID    int64              `json:"workflow_id"`
	Key           string             `json:"key"`
	Value         map[string]float64 `json:"value"`
	Date          time.Time          `json:"timestamp"`
	Num           int64              `json:"run"`
}

// MetricRequest represents the request to retrieve metrics
type MetricRequest struct {
	ProjectKey    string `json:"project_key"`
	ApplicationID int64  `json:"application_id"`
	WorkflowID    int64  `json:"workflow_id"`
	Key           string `json:"key"`
}
