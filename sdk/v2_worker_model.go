package sdk

import (
	"encoding/json"

	"github.com/xeipuuv/gojsonschema"
)

const (
	WorkerModelTypeOpenstack = "openstack"
	WorkerModelTypeDocker    = "docker"
	WorkerModelTypeVSphere   = "vsphere"
)

type V2WorkerModel struct {
	Name        string          `json:"name" cli:"name" jsonschema:"minLength=1" jsonschema_extras:"order=1" jsonschema_description:"Name of the worker model"`
	Description string          `json:"description,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Description of the worker model"`
	OSArch      string          `json:"osarch" jsonschema_extras:"order=3" jsonschema_description:"OS/Arch of the worker model"`
	Type        string          `json:"type" cli:"type" jsonschema:"enum=docker,enum=openstack,enum=vsphere" jsonschema_extras:"order=4" jsonschema_description:"Type of worker model: docker, openstack, vsphere"`
	Spec        json.RawMessage `json:"spec" jsonschema_allof_type:"type=docker:#/$defs/V2WorkerModelDockerSpec,type=openstack:#/$defs/V2WorkerModelOpenstackSpec,type=vsphere:#/$defs/V2WorkerModelVSphereSpec" jsonschema_extras:"order=5" jsonschema_description:"Specification of the worker model"`

	// Not in json schema
	Commit string `json:"commit,omitempty" jsonschema:"-"`
}

type V2WorkerModelDockerSpec struct {
	Image    string            `json:"image" jsonschema:"minLength=1" jsonschema_extras:"order=1" jsonschema_description:"Docker image name"`
	Username string            `json:"username,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Username to login to the registry"`
	Password string            `json:"password,omitempty" jsonschema_extras:"order=3" jsonschema_description:"User password to login to the registry"`
	Envs     map[string]string `json:"envs,omitempty" jsonschema_extras:"order=4" jsonschema_description:"Additional environment variables to inject into the worker"`
}

type V2WorkerModelOpenstackSpec struct {
	Image  string `json:"image" jsonschema_description:"Name of the openstack image"`
	Flavor string `json:"flavor" jsonschema_description:"Openstack flavor used by the worker model"`
}

type V2WorkerModelVSphereSpec struct {
	Image    string `json:"image" jsonschema_description:"Name of the vsphere template"`
	Username string `json:"username,omitempty" jsonschema_description:"Username to connect to the VM"`
	Password string `json:"password,omitempty" jsonschema_description:"Username password to connect to the VM"`
}

func (wm V2WorkerModel) GetName() string {
	return wm.Name
}

func (wm V2WorkerModel) Lint() []error {
	workerModelSchema := GetWorkerModelJsonSchema()
	workerModelSchemaS, err := workerModelSchema.MarshalJSON()
	if err != nil {
		return []error{NewErrorFrom(err, "unable to load worker model schema")}
	}
	schemaLoader := gojsonschema.NewStringLoader(string(workerModelSchemaS))

	modelJson, err := json.Marshal(wm)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to marshal worker model")}
	}
	documentLoader := gojsonschema.NewStringLoader(string(modelJson))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to validate worker model")}
	}
	if result.Valid() {
		return nil
	}

	errors := make([]error, 0, len(result.Errors()))
	for _, e := range result.Errors() {
		errors = append(errors, NewErrorFrom(ErrInvalidData, e.String()))
	}
	return errors
}
