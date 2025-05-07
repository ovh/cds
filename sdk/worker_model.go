package sdk

import (
	"database/sql/driver"
	"encoding/json"
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

// Model represents a worker model (ex: Go 1.5.1 Docker Images)
// with specified capabilities (ex: go, golint and go2xunit binaries)
type Model struct {
	ID                  int64               `json:"id" db:"id" cli:"-"`
	Name                string              `json:"name" db:"name" cli:"name,key" action_metadata:"model-name"`
	Description         string              `json:"description" db:"description" cli:"description"`
	Type                string              `json:"type" db:"type" cli:"type"`
	Disabled            bool                `json:"disabled" db:"disabled" cli:"disabled"`
	Restricted          bool                `json:"restricted" db:"restricted" cli:"restricted"`
	RegisteredOS        *string             `json:"registered_os" db:"registered_os" cli:"-"`
	RegisteredArch      *string             `json:"registered_arch" db:"registered_arch" cli:"-"`
	NeedRegistration    bool                `json:"need_registration" db:"need_registration" cli:"-"`
	LastRegistration    time.Time           `json:"last_registration" db:"last_registration" cli:"-"`
	CheckRegistration   bool                `json:"check_registration" db:"check_registration" cli:"-"`
	UserLastModified    time.Time           `json:"user_last_modified" db:"user_last_modified" cli:"-"`
	Author              Author              `json:"created_by" db:"created_by" cli:"-"`
	GroupID             int64               `json:"group_id" db:"group_id" cli:"-"`
	NbSpawnErr          int64               `json:"nb_spawn_err" db:"nb_spawn_err" cli:"nb_spawn_err"`
	LastSpawnErr        *string             `json:"last_spawn_err" db:"last_spawn_err" cli:"-"`
	LastSpawnErrLogs    *string             `json:"last_spawn_err_log" db:"last_spawn_err_log" cli:"-"`
	DateLastSpawnErr    *time.Time          `json:"date_last_spawn_err" db:"date_last_spawn_err" cli:"-"`
	IsDeprecated        bool                `json:"is_deprecated" db:"is_deprecated" cli:"deprecated"`
	ModelVirtualMachine ModelVirtualMachine `json:"model_virtual_machine,omitempty" db:"model_virtual_machine" cli:"-"`
	ModelDocker         ModelDocker         `json:"model_docker,omitempty" db:"model_docker" cli:"-"`
	// aggregates
	Editable               bool          `json:"editable,omitempty" db:"-"`
	Group                  *Group        `json:"group" db:"-" cli:"-"`
	RegisteredCapabilities []Requirement `json:"registered_capabilities" db:"-" cli:"-"`
	IsOfficial             bool          `json:"is_official" db:"-" cli:"official"`
	PatternName            string        `json:"pattern_name,omitempty" db:"-" cli:"-"`
}

type Models []Model

type WorkerModelSecret struct {
	ID            string    `json:"id" db:"id"`
	Created       time.Time `json:"created" cli:"created" db:"created"`
	WorkerModelID int64     `json:"worker_model_id" db:"worker_model_id"`
	Name          string    `json:"name" db:"name"`
	Value         string    `json:"value" db:"cipher_value" gorpmapping:"encrypted,WorkerModelID,Name"`
}

type WorkerModelSecrets []WorkerModelSecret

func (w WorkerModelSecrets) ToMap() map[string]string {
	res := make(map[string]string, len(w))
	for i := range w {
		res[w[i].Name] = w[i].Value
	}
	return res
}

// Author struct contains info about model author.
type Author struct {
	Username string `json:"username" cli:"-"`
	Fullname string `json:"fullname" cli:"-"`
	Email    string `json:"email" cli:"-"`
}

// Value returns driver.Value from author.
func (a Author) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, WrapError(err, "cannot marshal Author")
}

// Scan author.
func (a *Author) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, a), "cannot unmarshal Author")
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
	if !NamePatternRegex.MatchString(m.Name) {
		return WrapError(ErrInvalidWorkerModelNamePattern, "worker model name %s does not respect pattern %s", m.Name, NamePattern)
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
		if m.ModelVirtualMachine.User == "" || m.ModelVirtualMachine.Password == "" {
			return WrapError(ErrWrongRequest, "missing vm user and password")
		}
	default:
		return NewErrorFrom(ErrWrongRequest, "invalid worker model type")
	}
	return nil
}

// Path returns full path of the model that contains group and model names.
func (m Model) Path() string {
	return ComputeWorkerModelPath(m.Group.Name, m.Name)
}

// ComputeWorkerModelPath returns path for a worker model with given group name and model name.
func ComputeWorkerModelPath(groupName, modelName string) string {
	if groupName == SharedInfraGroupName {
		return modelName
	}
	return fmt.Sprintf("%s/%s", groupName, modelName)
}

// ModelVirtualMachine for openstack or vsphere.
type ModelVirtualMachine struct {
	Image    string `json:"image,omitempty"`
	Flavor   string `json:"flavor,omitempty"`
	PreCmd   string `json:"pre_cmd,omitempty"`
	Cmd      string `json:"cmd,omitempty"`
	PostCmd  string `json:"post_cmd,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

// Value returns driver.Value from model virtual machine.
func (m ModelVirtualMachine) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal ModelVirtualMachine")
}

// Scan model virtual machine.
func (m *ModelVirtualMachine) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, m), "cannot unmarshal ModelVirtualMachine")
}

// ModelDocker for swarm and kubernetes.
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

// Value returns driver.Value from model docker.
func (m ModelDocker) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal ModelDocker")
}

// Scan model docker.
func (m *ModelDocker) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, m), "cannot unmarshal ModelDocker")
}

// ModelPattern represent patterns for users and admin when creating a worker model
type ModelPattern struct {
	ID    int64     `json:"id" db:"id"`
	Name  string    `json:"name" db:"name"`
	Type  string    `json:"type" db:"type"`
	Model ModelCmds `json:"model" db:"model"`
}

// ModelCmds is the struct to represent a pattern
type ModelCmds struct {
	Envs    map[string]string `json:"envs,omitempty"`
	Shell   string            `json:"shell,omitempty"`
	PreCmd  string            `json:"pre_cmd,omitempty"`
	Cmd     string            `json:"cmd,omitempty"`
	PostCmd string            `json:"post_cmd,omitempty"`
}

// Value returns driver.Value from model cmds.
func (m ModelCmds) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal ModelCmds")
}

// Scan model cmds.
func (m *ModelCmds) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, m), "cannot unmarshal ModelCmds")
}

// ModelsToGroupIDs returns group ids of given worker models.
func ModelsToGroupIDs(ms []*Model) []int64 {
	ids := make([]int64, len(ms))
	for i := range ms {
		ids[i] = ms[i].GroupID
	}
	return ids
}
