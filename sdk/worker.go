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
	ID                     int64               `json:"id" db:"id" cli:"-"`
	Name                   string              `json:"name"  db:"name" cli:"name"`
	Description            string              `json:"description"  db:"description" cli:"description"`
	Type                   string              `json:"type"  db:"type" cli:"type"`
	ModelVirtualMachine    ModelVirtualMachine `json:"model_virtual_machine,omitempty" db:"-" cli:"-"`
	ModelDocker            ModelDocker         `json:"model_docker,omitempty" db:"-" cli:"-"`
	Communication          string              `json:"communication"  db:"communication" cli:"communication"`
	Disabled               bool                `json:"disabled"  db:"disabled" cli:"disabled"`
	Restricted             bool                `json:"restricted"  db:"restricted" cli:"restricted"`
	RegisteredCapabilities []Requirement       `json:"registered_capabilities"  db:"-" cli:"-"`
	RegisteredOS           string              `json:"registered_os"  db:"-" cli:"-"`
	RegisteredArch         string              `json:"registered_arch"  db:"-" cli:"-"`
	NeedRegistration       bool                `json:"need_registration"  db:"need_registration" cli:"-"`
	LastRegistration       time.Time           `json:"last_registration"  db:"last_registration" cli:"-"`
	UserLastModified       time.Time           `json:"user_last_modified"  db:"user_last_modified" cli:"-"`
	CreatedBy              User                `json:"created_by" db:"-" cli:"-"`
	Provision              int64               `json:"provision" db:"provision" cli:"provision"`
	GroupID                int64               `json:"group_id" db:"group_id" cli:"-"`
	Group                  Group               `json:"group" db:"-" cli:"-"`
	NbSpawnErr             int64               `json:"nb_spawn_err" db:"nb_spawn_err" cli:"nb_spawn_err"`
	LastSpawnErr           string              `json:"last_spawn_err" db:"last_spawn_err" cli:"-"`
	DateLastSpawnErr       *time.Time          `json:"date_last_spawn_err" db:"date_last_spawn_err" cli:"-"`
	IsDeprecated           bool                `json:"is_deprecated" db:"is_deprecated" cli:"deprecated"`
	IsOfficial             bool                `json:"is_official" db:"-" cli:"official"`
}

// ModelVirtualMachine for openstack or vsphere
type ModelVirtualMachine struct {
	Image   string `json:"image,omitempty"`
	Flavor  string `json:"flavor,omitempty"`
	PreCmd  string `json:"pre_cmd,omitempty"`
	Cmd     string `json:"cmd,omitempty"`
	PostCmd string `json:"post_cmd,omitempty"`
}

// ModelDocker for swarm, marathon and kubernetes
type ModelDocker struct {
	Image  string `json:"image,omitempty"`
	Memory int64  `json:"memory,omitempty"`
	Cmd    string `json:"cmd,omitempty"`
}

// WorkerArgs is all the args needed to run a worker
type WorkerArgs struct {
	API                string `json:"api"`
	Token              string `json:"token"`
	Name               string `json:"name"`
	BaseDir            string `json:"base_dir"`
	HTTPInsecure       bool   `json:"http_insecure"`
	Key                string `json:"key"`
	Model              int64  `json:"model"`
	Hatchery           int64  `json:"hatchery"`
	HatcheryName       string `json:"hatchery_name"`
	PipelineBuildJobID int64  `json:"pipeline_build_job_id"`
	WorkflowJobID      int64  `json:"workflow_job_id"`
	TTL                int    `json:"ttl"`
	FromWorkerImage    bool   `json:"from_worker_image"`
	//Graylog params
	GraylogHost       string `json:"graylog_host"`
	GraylogPort       int    `json:"graylog_port"`
	GraylogExtraKey   string `json:"graylog_extra_key"`
	GraylogExtraValue string `json:"graylog_extra_value"`
	//GRPC Params
	GrpcAPI      string `json:"grpc_api"`
	GrpcInsecure bool   `json:"grpc_insecure"`
}
