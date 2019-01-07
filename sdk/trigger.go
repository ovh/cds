package sdk

// Prerequisite defines a expected value to one triggering pipeline parameter
type Prerequisite struct {
	Parameter     string `json:"parameter"`
	ExpectedValue string `json:"expected_value"`
}

// PipelineTrigger represent a pipeline trigger
type PipelineTrigger struct {
	ID int64 `json:"id"`

	SrcProject     Project     `json:"src_project" yaml:"-"`
	SrcApplication Application `json:"src_application" yaml:"-"`
	SrcPipeline    Pipeline    `json:"src_pipeline" yaml:"-"`
	SrcEnvironment Environment `json:"src_environment" yaml:"-"`

	DestProject     Project     `json:"dest_project" yaml:"-"`
	DestApplication Application `json:"dest_application" yaml:"-"`
	DestPipeline    Pipeline    `json:"dest_pipeline" yaml:"-"`
	DestEnvironment Environment `json:"dest_environment" yaml:"-"`

	Manual        bool           `json:"manual"`
	Parameters    []Parameter    `json:"parameters"`
	Prerequisites []Prerequisite `json:"prerequisites"`
	LastModified  int64          `json:"last_modified"`
}
