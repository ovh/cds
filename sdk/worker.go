package sdk

import (
	"time"
)

// Worker represents instances of CDS workers living to serve.
type Worker struct {
	ID            string    `json:"id" cli:"-"`
	Name          string    `json:"name" cli:"name,key"`
	LastBeat      time.Time `json:"-" cli:"-"`
	GroupID       int64     `json:"group_id" cli:"-"`
	ModelID       int64     `json:"model_id" cli:"-"`
	ActionBuildID int64     `json:"action_build_id" cli:"-"`
	Model         *Model    `json:"model" cli:"-"`
	HatcheryID    int64     `json:"hatchery_id" cli:"-"`
	HatcheryName  string    `json:"hatchery_name" cli:"-"`
	JobType       string    `json:"job_type" cli:"-"`    // sdk.JobType...
	Status        Status    `json:"status" cli:"status"` // Waiting, Building, Disabled, Unknown
	Uptodate      bool      `json:"up_to_date" cli:"-"`
}

// WorkerRegistrationForm represents the arguments needed to register a worker
type WorkerRegistrationForm struct {
	Name               string
	Token              string
	ModelID            int64
	Hatchery           int64
	HatcheryName       string
	BinaryCapabilities []string
	Version            string
	OS                 string
	Arch               string
}

// WorkerTakeForm contains booked JobID if exists
type WorkerTakeForm struct {
	BookedJobID int64
	Time        time.Time
	OS          string
	Arch        string
	Version     string
}

// Existing worker type
const (
	Docker      = "docker"
	HostProcess = "host"
	Openstack   = "openstack"
	VSphere     = "vsphere"
)

var (
	// AvailableWorkerModelType List of all worker model type
	AvailableWorkerModelType = []string{
		string(Docker),
		string(HostProcess),
		string(Openstack),
		string(VSphere),
	}
)

// Existing worker communication
const (
	HTTP = "http"
	GRPC = "grpc"
)

var (
	// AvailableWorkerModelCommunication List of all worker model communication
	AvailableWorkerModelCommunication = []string{
		string(HTTP),
		string(GRPC),
	}
)

// SpawnErrorForm represents the arguments needed to add error registration on worker model
type SpawnErrorForm struct {
	Error string
}

// Model represents a worker model (ex: Go 1.5.1 Docker Images)
// with specified capabilities (ex: go, golint and go2xunit binaries)
//easyjson:json
type Model struct {
	ID               int64              `json:"id" db:"id" cli:"-"`
	Name             string             `json:"name"  db:"name" cli:"name"`
	Description      string             `json:"description"  db:"description" cli:"description"`
	Type             string             `json:"type"  db:"type" cli:"type"`
	Image            string             `json:"image" db:"image" cli:"-"`
	Capabilities     []Requirement      `json:"capabilities" db:"-" cli:"-"`
	Communication    string             `json:"communication"  db:"communication" cli:"communication"`
	Template         *map[string]string `json:"template"  db:"template" cli:"-"`
	RunScript        string             `json:"run_script"  db:"run_script" cli:"-"`
	Disabled         bool               `json:"disabled"  db:"disabled" cli:"disabled"`
	Restricted       bool               `json:"restricted"  db:"restricted" cli:"restricted"`
	NeedRegistration bool               `json:"need_registration"  db:"need_registration" cli:"-"`
	LastRegistration time.Time          `json:"last_registration"  db:"last_registration" cli:"-"`
	UserLastModified time.Time          `json:"user_last_modified"  db:"user_last_modified" cli:"-"`
	CreatedBy        User               `json:"created_by" db:"-" cli:"-"`
	Provision        int64              `json:"provision" db:"provision" cli:"provision"`
	GroupID          int64              `json:"group_id" db:"group_id" cli:"-"`
	Group            Group              `json:"group" db:"-" cli:"-"`
	NbSpawnErr       int64              `json:"nb_spawn_err" db:"nb_spawn_err" cli:"nb_spawn_err"`
	LastSpawnErr     string             `json:"last_spawn_err" db:"last_spawn_err" cli:"-"`
	DateLastSpawnErr *time.Time         `json:"date_last_spawn_err" db:"date_last_spawn_err" cli:"-"`
	IsDeprecated     bool               `json:"is_deprecated" db:"is_deprecated" cli:"deprecated"`
	IsOfficial       bool               `json:"is_official" db:"-" cli:"official"`
}

// OpenstackModelData type details the "Image" field of Openstack type model
type OpenstackModelData struct {
	Image    string `json:"os"`
	Flavor   string `json:"flavor,omitempty"`
	UserData string `json:"user_data"`
}
