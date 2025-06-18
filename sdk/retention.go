package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sguiheux/jsonschema"
)

type ProjectRunRetention struct {
	ID           string     `json:"id" db:"id"`
	ProjectKey   string     `json:"project_key" db:"project_key"`
	LastModified time.Time  `json:"last_modified" db:"last_modified"`
	Retentions   Retentions `json:"retentions" db:"retention"`
}
type Retentions struct {
	WorkflowRetentions []WorkflowRetentions `json:"retention,omitempty"`
	DefaultRetention   RetentionRule        `json:"default_retention"`
}

type WorkflowRetentions struct {
	Workflow         string                  `json:"workflow" jsonschema_description:"Workflow where the retention rule is applied"`
	Rules            []WorkflowRetentionRule `json:"rules,omitempty"`
	DefaultRetention *RetentionRule          `json:"default_retention,omitempty"`
}

type WorkflowRetentionRule struct {
	RetentionRule
	GitRef string `json:"git_ref" jsonschema_description:"Git reference where the rule is applied"`
}

type RetentionRule struct {
	DurationInDays int64 `json:"duration_in_days,omitempty" jsonschema_description:"The number of days before the run is deleted"`
	Count          int64 `json:"count,omitempty" jsonschema_description:"The maximum numbers of tuns that is kept"`
}

func (r Retentions) Value() (driver.Value, error) {
	j, err := json.Marshal(r)
	return j, WrapError(err, "cannot marshal Retentions")
}

func (r *Retentions) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, r), "cannot unmarshal Retentions")
}

func GetProjectRunRetentionJsonSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{Anonymous: false}
	retentionSchema := reflector.Reflect(&Retentions{})
	return retentionSchema
}
