package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// Worker represents instances of CDS workers living to serve.
type Worker struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	LastBeat   time.Time `json:"-"`
	GroupID    int64     `json:"group_id"`
	Model      int64     `json:"model"`
	HatcheryID int64     `json:"hatchery_id"`
	Status     Status    `json:"status"` // Waiting, Building, Disabled, Unknown
	Uptodate   bool      `json:"up_to_date"`
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

// Model represents a worker model (ex: Go 1.5.1 Docker Images)
// with specified capabilities (ex: go, golint and go2xunit binaries)
type Model struct {
	ID           int64         `json:"id" db:"id"`
	Name         string        `json:"name"  db:"name"`
	Type         string        `json:"type"  db:"type"`
	Image        string        `json:"image" db:"image"`
	Capabilities []Requirement `json:"capabilities" db:"-"`
	CreatedBy    User          `json:"created_by" db:"-"`
	OwnerID      int64         `json:"owner_id" db:"owner_id"` //DEPRECATED
	GroupID      int64         `json:"group_id" db:"group_id"`
}

// ModelStatus sums up the number of worker deployed and wanted for a given model
type ModelStatus struct {
	ModelID       int64         `json:"model_id" yaml:"-"`
	ModelName     string        `json:"model_name" yaml:"name"`
	ModelGroupID  int64         `json:"model_group_id" yaml:"model_group_id"`
	CurrentCount  int64         `json:"current_count" yaml:"current"`
	WantedCount   int64         `json:"wanted_count" yaml:"wanted"`
	BuildingCount int64         `json:"building_count" yaml:"building"`
	Requirements  []Requirement `json:"requirements"`
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

	uri := "/worker"
	data, code, errr := Request("GET", uri, nil)
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

// GetWorkerModels retrieves all worker models available to user
func GetWorkerModels() ([]Model, error) {
	uri := fmt.Sprintf("/worker/model")

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

	_, _, err := Request("DELETE", uri, nil)
	if err != nil {
		return err
	}

	return nil
}

// AddCapabilityToWorkerModel adds a capability to given model
func AddCapabilityToWorkerModel(modelID int64, name string, capaType string, value string) error {

	uri := fmt.Sprintf("/worker/model/%d/capability", modelID)

	r := Requirement{
		Name:  name,
		Type:  capaType,
		Value: value,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	_, code, err := Request("POST", uri, data)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// UpdateCapabilityToWorkerModel updates a capability to given model
func UpdateCapabilityToWorkerModel(modelID int64, name string, capaType string, value string) error {

	uri := fmt.Sprintf("/worker/model/%d/capability/%s", modelID, name)

	r := Requirement{
		Name:  name,
		Type:  capaType,
		Value: value,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	_, code, err := Request("PUT", uri, data)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// DeleteWorkerCapability removes a capability from given worker model
func DeleteWorkerCapability(workerModelID int64, capaName string) error {
	uri := fmt.Sprintf("/worker/model/%d/capability/%s", workerModelID, capaName)

	if _, _, err := Request("DELETE", uri, nil); err != nil {
		return err
	}

	return nil
}

// SetWorkerStatus update worker status
func SetWorkerStatus(s Status) error {
	var uri string
	switch s {
	case StatusChecking:
		uri = fmt.Sprintf("/worker/checking")
	case StatusWaiting:
		uri = fmt.Sprintf("/worker/waiting")
	default:
		return fmt.Errorf("Unsupported status : %s", s.String())
	}

	_, code, err := Request("POST", uri, nil)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("cds: api error (%d)", code)
	}

	return nil
}
