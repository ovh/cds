package sdk

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
)

// StaticFiles define a files needed to be save for serving static files
type StaticFiles struct {
	ID                   int64     `json:"id" db:"id" cli:"id"`
	Name                 string    `json:"name" db:"name" cli:"name"`
	WorkflowID           int64     `json:"workflow_id" db:"-"`
	NodeRunID            int64     `json:"workflow_node_run_id" db:"workflow_node_run_id"`
	NodeJobRunID         int64     `json:"workflow_node_run_job_id,omitempty" db:"-"`
	EntryPoint           string    `json:"entrypoint" db:"entrypoint"`
	StaticKey            string    `json:"static_key" db:"static_key"`
	PublicURL            string    `json:"public_url" db:"public_url" cli:"public_url"`
	Created              time.Time `json:"created" db:"created" cli:"created"`
	ProjectIntegrationID *int64    `json:"project_integration_id" db:"project_integration_id"`

	TempURL   string `json:"temp_url,omitempty" db:"-"`
	SecretKey string `json:"secret_key,omitempty" db:"-"`
}

//GetName returns the name the artifact
func (staticfile *StaticFiles) GetName() string {
	return base64.RawURLEncoding.EncodeToString([]byte(staticfile.Name))
}

//GetPath returns the path of the artifact
func (staticfile *StaticFiles) GetPath() string {
	var container string
	if staticfile.StaticKey != "" {
		container = fmt.Sprintf("%d-%s-%s", staticfile.WorkflowID, url.PathEscape(staticfile.StaticKey), url.PathEscape(staticfile.Name))
	} else {
		container = fmt.Sprintf("%d-%s", staticfile.NodeJobRunID, url.PathEscape(staticfile.Name))
	}
	return base64.RawURLEncoding.EncodeToString([]byte(container))
}

// Equal returns true if  StaticFiles are equal to another one
func (staticfile StaticFiles) Equal(currStaticfile StaticFiles) bool {
	return currStaticfile.NodeRunID == staticfile.NodeRunID &&
		currStaticfile.Name == staticfile.Name &&
		currStaticfile.EntryPoint == staticfile.EntryPoint &&
		currStaticfile.StaticKey == staticfile.StaticKey
}
