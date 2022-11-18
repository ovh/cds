package sdk

type FeatureName string

const (
	FeatureMFARequired  FeatureName = "mfa_required"
	FeaturePurgeName    FeatureName = "workflow-retention-policy"
	FeaturePurgeMaxRuns FeatureName = "workflow-retention-maxruns"
	FeatureTracing      FeatureName = "tracing"
	FeatureWorkflowV3   FeatureName = "workflow-v3"
	FeatureRegion       FeatureName = "region"
	FeatureCDSAsCode    FeatureName = "all-as-code"
)

type Feature struct {
	ID   int64       `json:"id" db:"id" cli:"-" yaml:"-"`
	Name FeatureName `json:"name" db:"name" cli:"name" yaml:"name"`
	Rule string      `json:"rule" db:"rule" cli:"-" yaml:"rule"`
}

type FeatureEnabledResponse struct {
	Name    FeatureName `json:"name"`
	Enabled bool        `json:"enabled"`
	Exists  bool        `json:"exists"`
}
