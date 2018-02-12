package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// Worker represents instances of CDS workers living to serve.
type Worker struct {
	ID            string    `json:"id" cli:"-"`
	Name          string    `json:"name" cli:"name"`
	LastBeat      time.Time `json:"-" cli:"lastbeat"`
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
}

// OpenstackModelData type details the "Image" field of Openstack type model
type OpenstackModelData struct {
	Image    string `json:"os"`
	Flavor   string `json:"flavor,omitempty"`
	UserData string `json:"user_data"`
}

// GetWorkers retrieves from engine all worker the user has access to
func GetWorkers(models ...string) ([]Worker, error) {
	if len(models) != 0 {
		return nil, fmt.Errorf("not implemented")
	}

	data, _, errr := Request("GET", "/worker", nil)
	if errr != nil {
		return nil, errr
	}

	var workers []Worker
	if err := json.Unmarshal(data, &workers); err != nil {
		return nil, err
	}

	return workers, nil
}

// DisableWorker order the engine to disable given worker, not allowing it to take builds
func DisableWorker(workerID string) error {
	_, _, err := Request("POST", fmt.Sprintf("/worker/%s/disable", workerID), nil)
	return err
}

// AddWorkerModel registers a new worker model available
func AddWorkerModel(name string, t string, img string, groupID int64) (*Model, error) {
	uri := fmt.Sprintf("/worker/model")

	m := Model{
		Name:    name,
		Type:    t,
		Image:   img,
		GroupID: groupID,
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	data, code, err := Request("POST", uri, data)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// GetWorkerModel retrieves a specific worker model
func GetWorkerModel(name string) (*Model, error) {
	uri := fmt.Sprintf("/worker/model?name=%s", name)

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	var m Model
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	return &m, nil
}

// UpdateWorkerModel updates all characteristics of a worker model
func UpdateWorkerModel(id int64, name string, t string, value string) error {
	uri := fmt.Sprintf("/worker/model/%d", id)

	data, err := json.Marshal(Model{ID: id, Name: name, Type: t, Image: value})
	if err != nil {
		return err
	}

	_, code, err := Request("PUT", uri, data)
	if err != nil {
		return err
	}
	if code > 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// GetWorkerModels retrieves all worker models available to user (enabled or not)
func GetWorkerModels() ([]Model, error) {
	return getWorkerModels(true)
}

func getWorkerModels(withDisabled bool) ([]Model, error) {
	var uri string
	if withDisabled {
		uri = fmt.Sprintf("/worker/model")
	} else {
		uri = fmt.Sprintf("/worker/model/enabled")
	}

	data, _, errr := Request("GET", uri, nil)
	if errr != nil {
		return nil, errr
	}

	var models []Model
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, err
	}

	return models, nil
}

// DeleteWorkerModel deletes a worker model and all its capabilities
func DeleteWorkerModel(workerModelID int64) error {
	uri := fmt.Sprintf("/worker/model/%d", workerModelID)

	if _, _, err := Request("DELETE", uri, nil); err != nil {
		return err
	}

	return nil
}
