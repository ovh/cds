package sdk

import (
	"bytes"
	"html/template"
	"time"
)

// Worker represents instances of CDS workers living to serve.
type Worker struct {
	ID            string    `json:"id" cli:"-"`
	Name          string    `json:"name" cli:"name,key"`
	LastBeat      time.Time `json:"lastbeat" cli:"lastbeat"`
	GroupID       int64     `json:"group_id" cli:"-"`
	ModelID       int64     `json:"model_id" cli:"-"`
	ActionBuildID int64     `json:"action_build_id" cli:"-"`
	Model         *Model    `json:"model" cli:"-"`
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

// Existing worker model type
const (
	Docker      = "docker"
	HostProcess = "host"
	Openstack   = "openstack"
	VSphere     = "vsphere"
)

// WorkerModelValidate returns if given strings are valid worker model type.
func WorkerModelValidate(modelType string) bool {
	for _, s := range AvailableWorkerModelType {
		if s == modelType {
			return true
		}
	}
	return false
}

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
	Logs  []byte
}

// Model represents a worker model (ex: Go 1.5.1 Docker Images)
// with specified capabilities (ex: go, golint and go2xunit binaries)
//easyjson:json
type Model struct {
	ID                     int64               `json:"id" db:"id" cli:"-"`
	Name                   string              `json:"name" db:"name" cli:"name,key"`
	Description            string              `json:"description"  db:"description" cli:"description"`
	Type                   string              `json:"type"  db:"type" cli:"type"`
	Image                  string              `json:"image" db:"image" cli:"image"` // TODO: DELETE after migration done
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
	CheckRegistration      bool                `json:"check_registration"  db:"check_registration" cli:"-"`
	UserLastModified       time.Time           `json:"user_last_modified"  db:"user_last_modified" cli:"-"`
	CreatedBy              User                `json:"created_by" db:"-" cli:"-"`
	Provision              int64               `json:"provision" db:"provision" cli:"provision"`
	GroupID                int64               `json:"group_id" db:"group_id" cli:"-"`
	Group                  Group               `json:"group" db:"-" cli:"-"`
	NbSpawnErr             int64               `json:"nb_spawn_err" db:"nb_spawn_err" cli:"nb_spawn_err"`
	LastSpawnErr           string              `json:"last_spawn_err" db:"-" cli:"-"`
	LastSpawnErrLogs       *string             `json:"last_spawn_err_log" db:"-" cli:"-"`
	DateLastSpawnErr       *time.Time          `json:"date_last_spawn_err" db:"date_last_spawn_err" cli:"-"`
	IsDeprecated           bool                `json:"is_deprecated" db:"is_deprecated" cli:"deprecated"`
	IsOfficial             bool                `json:"is_official" db:"-" cli:"official"`
	PatternName            string              `json:"pattern_name,omitempty" db:"-" cli:"-"`
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
	Image  string            `json:"image,omitempty"`
	Memory int64             `json:"memory,omitempty"`
	Envs   map[string]string `json:"envs,omitempty"`
	Shell  string            `json:"shell,omitempty"`
	Cmd    string            `json:"cmd,omitempty"`
}

// ModelPattern represent patterns for users and admin when creating a worker model
type ModelPattern struct {
	ID    int64     `json:"id" db:"id"`
	Name  string    `json:"name" db:"name"`
	Type  string    `json:"type" db:"type"`
	Model ModelCmds `json:"model" db:"-"`
}

// ModelCmds is the struct to represent a pattern
type ModelCmds struct {
	Envs    map[string]string `json:"envs,omitempty"`
	Shell   string            `json:"shell,omitempty"`
	PreCmd  string            `json:"pre_cmd,omitempty"`
	Cmd     string            `json:"cmd,omitempty"`
	PostCmd string            `json:"post_cmd,omitempty"`
}

// WorkerArgs is all the args needed to run a worker
type WorkerArgs struct {
	API             string `json:"api"`
	Token           string `json:"token"`
	Name            string `json:"name"`
	BaseDir         string `json:"base_dir"`
	HTTPInsecure    bool   `json:"http_insecure"`
	Model           int64  `json:"model"`
	HatcheryName    string `json:"hatchery_name"`
	WorkflowJobID   int64  `json:"workflow_job_id"`
	TTL             int    `json:"ttl"`
	FromWorkerImage bool   `json:"from_worker_image"`
	//Graylog params
	GraylogHost       string `json:"graylog_host"`
	GraylogPort       int    `json:"graylog_port"`
	GraylogExtraKey   string `json:"graylog_extra_key"`
	GraylogExtraValue string `json:"graylog_extra_value"`
	//GRPC Params
	GrpcAPI      string `json:"grpc_api"`
	GrpcInsecure bool   `json:"grpc_insecure"`
}

// TemplateEnvs return envs interpolated with worker arguments
func TemplateEnvs(args WorkerArgs, envs map[string]string) (map[string]string, error) {
	for name, value := range envs {
		tmpl, errt := template.New("env").Parse(value)
		if errt != nil {
			return envs, errt
		}
		var buffer bytes.Buffer
		if errTmpl := tmpl.Execute(&buffer, args); errTmpl != nil {
			return envs, errTmpl
		}
		envs[name] = buffer.String()
	}

	return envs, nil
}

// WorkflowNodeJobRunData is returned to worker in answer to postTakeWorkflowJobHandler
type WorkflowNodeJobRunData struct {
	NodeJobRun WorkflowNodeJobRun
	Secrets    []Variable
	Number     int64
	SubNumber  int64
}
