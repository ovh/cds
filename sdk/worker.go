package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// Worker represents instances of CDS workers living to serve.
type Worker struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	LastBeat     time.Time `json:"-"`
	GroupID      int64     `json:"group_id"`
	Model        int64     `json:"model"`
	HatcheryID   int64     `json:"hatchery_id"`
	HatcheryName string    `json:"hatchery_name"`
	Status       Status    `json:"status"` // Waiting, Building, Disabled, Unknown
	Uptodate     bool      `json:"up_to_date"`
}

// Existing worker type
const (
	Docker      = "docker"
	HostProcess = "host"
	Openstack   = "openstack"
)

var (
	// AvailableWorkerModelType List of all worker model type
	AvailableWorkerModelType = []string{
		string(Docker),
		string(HostProcess),
		string(Openstack),
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
type Model struct {
	ID               int64              `json:"id" db:"id"`
	Name             string             `json:"name"  db:"name"`
	Type             string             `json:"type"  db:"type"`
	Image            string             `json:"image" db:"image"`
	Capabilities     []Requirement      `json:"capabilities" db:"-"`
	Communication    string             `json:"communication"  db:"communication"`
	Template         *map[string]string `json:"template"  db:"template"`
	RunScript        string             `json:"run_script"  db:"run_script"`
	Disabled         bool               `json:"disabled"  db:"disabled"`
	NeedRegistration bool               `json:"need_registration"  db:"need_registration"`
	LastRegistration time.Time          `json:"last_registration"  db:"last_registration"`
	UserLastModified time.Time          `json:"user_last_modified"  db:"user_last_modified"`
	CreatedBy        User               `json:"created_by" db:"-"`
	Provision        int64              `json:"provision" db:"provision"`
	GroupID          int64              `json:"group_id" db:"group_id"`
	Group            Group              `json:"group" db:"-"`
	NbSpawnErr       int64              `json:"nb_spawn_err" db:"nb_spawn_err"`
	LastSpawnErr     string             `json:"last_spawn_err" db:"last_spawn_err"`
	DateLastSpawnErr *time.Time         `json:"date_last_spawn_err" db:"date_last_spawn_err"`
}

// OpenstackModelData type details the "Image" field of Openstack type model
type OpenstackModelData struct {
	Image    string `json:"os"`
	Flavor   string `json:"flavor"`
	UserData string `json:"user_data"`
}

// GetWorkers retrieves from engine all worker the user has access to
func GetWorkers(models ...string) ([]Worker, error) {
	if len(models) != 0 {
		return nil, fmt.Errorf("not implemented")
	}

	data, code, errr := Request("GET", "/worker", nil)
	if errr != nil {
		return nil, errr
	}

	if code != 200 {
		return nil, fmt.Errorf("API error (%d)", code)
	}

	var workers []Worker
	if err := json.Unmarshal(data, &workers); err != nil {
		return nil, err
	}

	return workers, nil
}

// DisableWorker order the engine to disable given worker, not allowing it to take builds
func DisableWorker(workerID string) error {
	uri := fmt.Sprintf("/worker/%s/disable", workerID)

	_, code, err := Request("POST", uri, nil)
	if err != nil {
		return err
	}

	if code != 200 {
		return fmt.Errorf("API error (%d)", code)
	}

	return nil
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

// SpawnErrorWorkerModel updates registration infos on worker model
func SpawnErrorWorkerModel(id int64, info string) error {
	uri := fmt.Sprintf("/worker/model/error/%d", id)

	data, err := json.Marshal(SpawnErrorForm{Error: info})
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

// GetWorkerModelsEnabled retrieves all worker models enabled and available to user
func GetWorkerModelsEnabled() ([]Model, error) {
	return getWorkerModels(false)
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
