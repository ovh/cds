package sdk

import (
	"fmt"
	"time"
)

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
	NbSpawnErr             int64               `json:"nb_spawn_err" db:"nb_spawn_err" cli:"nb_spawn_err"`
	LastSpawnErr           string              `json:"last_spawn_err" db:"-" cli:"-"`
	LastSpawnErrLogs       *string             `json:"last_spawn_err_log" db:"-" cli:"-"`
	DateLastSpawnErr       *time.Time          `json:"date_last_spawn_err" db:"date_last_spawn_err" cli:"-"`
	IsDeprecated           bool                `json:"is_deprecated" db:"is_deprecated" cli:"deprecated"`
	IsOfficial             bool                `json:"is_official" db:"-" cli:"official"`
	PatternName            string              `json:"pattern_name,omitempty" db:"-" cli:"-"`
	// aggregates
	Editable bool   `json:"editable,omitempty" db:"-"`
	Group    *Group `json:"group" db:"-" cli:"-"`
}

// Update workflow template field from new data.
func (m *Model) Update(data Model) {
	m.Name = data.Name
	m.Description = data.Description
	m.Disabled = data.Disabled
	m.Restricted = data.Restricted
	m.IsDeprecated = data.IsDeprecated
	m.IsOfficial = data.IsOfficial
	m.GroupID = data.GroupID
	m.Type = data.Type
	m.Provision = data.Provision
	m.ModelDocker = ModelDocker{}
	m.ModelVirtualMachine = ModelVirtualMachine{}
	switch m.Type {
	case Docker:
		m.ModelDocker = data.ModelDocker
	default:
		m.ModelVirtualMachine = data.ModelVirtualMachine
	}
	m.Restricted = data.Restricted
}

// IsValid returns error if the model is not valid.
func (m Model) IsValid() error {
	if m.Name == "" {
		return WrapError(ErrWrongRequest, "invalid worker model name")
	}
	if m.GroupID == 0 {
		return WrapError(ErrWrongRequest, "missing worker model group data")
	}
	return nil
}

func (m Model) IsValidType() error {
	switch m.Type {
	case Docker:
		if m.ModelDocker.Image == "" {
			return NewErrorFrom(ErrWrongRequest, "invalid worker model image")
		}
		if m.PatternName == "" && (m.ModelDocker.Cmd == "" || m.ModelDocker.Shell == "") {
			return WrapError(ErrWrongRequest, "invalid worker model command or shell command")
		}
	case Openstack:
		if m.ModelVirtualMachine.Image == "" {
			return WrapError(ErrWrongRequest, "invalid worker model image")
		}
		if m.ModelVirtualMachine.Flavor == "" {
			return WrapError(ErrWrongRequest, "invalid worker model flavor")
		}
		if m.PatternName == "" && m.ModelVirtualMachine.Cmd == "" {
			return WrapError(ErrWrongRequest, "invalid worker model command")
		}
	case VSphere:
		if m.ModelVirtualMachine.Image == "" {
			return WrapError(ErrWrongRequest, "invalid worker model image")
		}
		if m.PatternName == "" && m.ModelVirtualMachine.Cmd == "" {
			return WrapError(ErrWrongRequest, "invalid worker model command")
		}
	default:
		return NewErrorFrom(ErrWrongRequest, "invalid worker model type")
	}
	return nil
}

// GetPath returns path for model.
func (m Model) GetPath(groupName string) string {
	if groupName == SharedInfraGroupName {
		return m.Name
	}
	return fmt.Sprintf("%s/%s", groupName, m.Name)
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
	Image    string            `json:"image,omitempty"`
	Private  bool              `json:"private,omitempty"`
	Registry string            `json:"registry,omitempty"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Memory   int64             `json:"memory,omitempty"`
	Envs     map[string]string `json:"envs,omitempty"`
	Shell    string            `json:"shell,omitempty"`
	Cmd      string            `json:"cmd,omitempty"`
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

// ModelsToGroupIDs returns group ids of given worker models.
func ModelsToGroupIDs(ms []*Model) []int64 {
	ids := make([]int64, len(ms))
	for i := range ms {
		ids[i] = ms[i].GroupID
	}
	return ids
}
