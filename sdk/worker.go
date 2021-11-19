package sdk

import (
	"time"
)

// Worker represents instances of CDS workers living to serve.
type Worker struct {
	ID           string    `json:"id" cli:"-" db:"id"`
	Name         string    `json:"name" cli:"name,key" db:"name"`
	LastBeat     time.Time `json:"lastbeat" cli:"lastbeat" db:"last_beat"`
	ModelID      *int64    `json:"model_id" cli:"-"  db:"model_id"`
	JobRunID     *int64    `json:"job_run_id" cli:"-"  db:"job_run_id"`
	Status       string    `json:"status" cli:"status" db:"status"` // Waiting, Building, Disabled, Unknown
	HatcheryID   *int64    `json:"hatchery_id,omitempty" cli:"-" db:"hatchery_id"`
	HatcheryName string    `json:"hatchery_name" cli:"-" db:"hatchery_name"` // If the hatchery service was deleted we will keep its name in the worker
	Uptodate     bool      `json:"uptodate" cli:"-" db:"-"`
	ConsumerID   string    `json:"-" cli:"-"  db:"auth_consumer_id"`
	Version      string    `json:"version" cli:"version"  db:"version"`
	OS           string    `json:"os" cli:"os"  db:"os"`
	Arch         string    `json:"arch" cli:"arch"  db:"arch"`
	PrivateKey   []byte    `json:"private_key,omitempty" cli:"-" db:"cypher_private_key" gorpmapping:"encrypted,ID,Name,JobRunID"`
}

// WorkerRegistrationForm represents the arguments needed to register a worker
type WorkerRegistrationForm struct {
	BinaryCapabilities []string
	Version            string
	OS                 string
	Arch               string
}

// SpawnErrorForm represents the arguments needed to add error registration on worker model
type SpawnErrorForm struct {
	Error string
	Logs  []byte
}

// WorkflowNodeJobRunData is returned to worker in answer to postTakeWorkflowJobHandler
type WorkflowNodeJobRunData struct {
	NodeJobRun               WorkflowNodeJobRun
	Secrets                  []Variable
	Features                 map[FeatureName]bool
	Number                   int64
	SubNumber                int64
	SigningKey               string
	GelfServiceAddr          string
	GelfServiceAddrEnableTLS bool
	CDNHttpAddr              string
	ProjectKey               string
	WorkflowName             string
	WorkflowID               int64
	RunID                    int64
	NodeRunName              string
}
