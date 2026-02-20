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
	Name        string          `json:"name" cli:"name" jsonschema:"minLength=1,example=my-worker-model" jsonschema_extras:"order=1" jsonschema_description:"Name of the worker model"`
	Description string          `json:"description,omitempty" jsonschema:"example=Worker model for building Go applications" jsonschema_extras:"order=2" jsonschema_description:"Description of the worker model"`
	OSArch      string          `json:"osarch" jsonschema:"example=linux/amd64" jsonschema_extras:"order=3" jsonschema_description:"OS/Arch of the worker model"`
	Type        string          `json:"type" cli:"type" jsonschema:"enum=docker,enum=openstack,enum=vsphere,example=docker" jsonschema_extras:"order=4" jsonschema_description:"Type of worker model: docker, openstack, vsphere"`
	Spec        json.RawMessage `json:"spec" jsonschema_allof_type:"type=docker:#/$defs/V2WorkerModelDockerSpec,type=openstack:#/$defs/V2WorkerModelOpenstackSpec,type=vsphere:#/$defs/V2WorkerModelVSphereSpec" jsonschema_extras:"order=5" jsonschema_description:"Specification of the worker model"`
}

type V2WorkerModelDockerSpec struct {
	Image    string            `json:"image" jsonschema:"minLength=1,example=golang:1.21" jsonschema_extras:"order=1" jsonschema_description:"Docker image name"`
	Username string            `json:"username,omitempty" jsonschema:"example=myuser" jsonschema_extras:"order=2" jsonschema_description:"Username to login to the registry"`
	Password string            `json:"password,omitempty" jsonschema:"example=${{ secrets.DOCKER_PASSWORD }}" jsonschema_extras:"order=3" jsonschema_description:"User password to login to the registry"`
	Envs     map[string]string `json:"envs,omitempty" jsonschema_extras:"order=4" jsonschema_description:"Additional environment variables to inject into the worker"`
}

type V2WorkerModelOpenstackSpec struct {
	Image  string `json:"image" jsonschema:"example=Ubuntu 22.04" jsonschema_description:"Name of the openstack image"`
	Flavor string `json:"flavor,omitempty" jsonschema:"example=b2-30" jsonschema_description:"Default flavor to use"`
}

type V2WorkerModelVSphereSpec struct {
	Image    string `json:"image" jsonschema:"example=my-vsphere-template" jsonschema_description:"Name of the vsphere template"`
	Flavor   string `json:"flavor,omitempty" jsonschema:"example=large" jsonschema_description:"Flavor to use for CPU/RAM sizing"`
	Username string `json:"username,omitempty" jsonschema:"example=admin" jsonschema_description:"Username to connect to the VM"`
	Password string `json:"password,omitempty" jsonschema:"example=${{ secrets.VSPHERE_PASSWORD }}" jsonschema_description:"Username password to connect to the VM"`
}

func (wm V2WorkerModel) GetName() string {
	return wm.Name
}

func (wm V2WorkerModel) Lint() []error {
	workerModelSchema := GetWorkerModelJsonSchema()
	workerModelSchemaS, err := workerModelSchema.MarshalJSON()
	if err != nil {
		return []error{NewErrorFrom(err, "worker model %s: unable to load worker model schema", wm.Name)}
	}
	schemaLoader := gojsonschema.NewStringLoader(string(workerModelSchemaS))

	modelJson, err := json.Marshal(wm)
	if err != nil {
		return []error{NewErrorFrom(err, "worker model %s: unable to marshal worker model", wm.Name)}
	}
	documentLoader := gojsonschema.NewStringLoader(string(modelJson))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return []error{NewErrorFrom(ErrInvalidData, "worker model %s: unable to validate worker model: %v", wm.Name, err.Error())}
	}
	if result.Valid() {
		return nil
	}

	errors := make([]error, 0, len(result.Errors()))
	for _, e := range result.Errors() {
		errors = append(errors, NewErrorFrom(ErrInvalidData, "worker model %s: yaml validation failed: %s", wm.Name, e.String()))
	}
	return errors
}
