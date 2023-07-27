package sdk

import (
	"time"
)

// V2Worker represents instances of CDS workers living to serve.
type V2Worker struct {
	ID           string    `json:"id" cli:"-" db:"id"`
	Name         string    `json:"name" cli:"name,key" db:"name"`
	LastBeat     time.Time `json:"last_beat" cli:"last_beat" db:"last_beat"`
	ModelName    string    `json:"model_name" cli:"-" db:"model_name"`
	JobRunID     string    `json:"job_run_id" cli:"-" db:"job_run_id"`
	Status       string    `json:"status" cli:"status" db:"status"` // Waiting, Building, Disabled, Unknown
	HatcheryID   string    `json:"hatchery_id,omitempty" cli:"-" db:"hatchery_id"`
	HatcheryName string    `json:"hatchery_name" cli:"-" db:"hatchery_name"` // If the hatchery service was deleted we will keep its name in the worker
	ConsumerID   string    `json:"-" cli:"-" db:"auth_consumer_id"`
	Version      string    `json:"version" cli:"version" db:"version"`
	OS           string    `json:"os" cli:"os" db:"os"`
	Arch         string    `json:"arch" cli:"arch" db:"arch"`
	PrivateKey   []byte    `json:"cypher_private_key,omitempty" cli:"-" db:"cypher_private_key" gorpmapping:"encrypted,ID,JobRunID,HatcheryID,ConsumerID"`
}

type V2TakeJobResponse struct {
	RunJob        V2WorkflowRunJob       `json:"run_job"`
	AsCodeActions map[string]V2Action    `json:"actions"`
	SigningKey    string                 `json:"signing_key"`
	Contexts      WorkflowRunJobsContext `json:"contexts"`
}
