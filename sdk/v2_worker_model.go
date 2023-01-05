package sdk

import (
	"encoding/json"
	"fmt"
	"github.com/xeipuuv/gojsonschema"
)

const (
	WorkerModelTypeOpenstack = "openstack"
	WorkerModelTypeDocker    = "docker"
	WorkerModelTypeVSphere   = "vsphere"
)

type V2WorkerModel struct {
	Name        string          `json:"name" cli:"name" jsonschema:"minLength=1" jsonschema_extras:"order=1" jsonschema_description:"Name of the worker model"`
	From        string          `json:"from,omitempty" jsonschema_extras:"order=3,disabled=true" jsonschema_description:"Name of the worker model template"`
	Description string          `json:"description,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Description of the worker model"`
	Type        string          `json:"type" cli:"type" jsonschema:"enum=docker,enum=openstack,enum=vsphere" jsonschema_extras:"order=4" jsonschema_description:"Type of worker model: docker, openstack, vsphere"`
	Spec        json.RawMessage `json:"spec" jsonschema_allof_type:"type=docker:#/$defs/V2WorkerModelDockerSpec,type=openstack:#/$defs/V2WorkerModelOpenstackSpec,type=vsphere:#/$defs/V2WorkerModelVSphereSpec" jsonschema_extras:"order=5" jsonschema_description:"Specification of the worker model"`
}

type V2WorkerModelDockerSpec struct {
	Image    string            `json:"image" jsonschema:"minLength=1" jsonschema_extras:"order=1" jsonschema_description:"Docker image name"`
	Registry string            `json:"registry,omitempty" jsonschema:"minLength=1" jsonschema_extras:"order=2" jsonschema_description:"The docker image registry"`
	Username string            `json:"username,omitempty" jsonschema_extras:"order=3" jsonschema_description:"Username to login to the registry"`
	Password string            `json:"password,omitempty" jsonschema_extras:"order=4" jsonschema_description:"User password to login to the registry"`
	Cmd      string            `json:"cmd" jsonschema:"minLength=1" jsonschema_extras:"order=6" jsonschema_description:"Command used by CDS to run the worker"`
	Shell    string            `json:"shell" jsonschema:"minLength=1" jsonschema_extras:"order=5" jsonschema_description:"Shell used to run the command"`
	Envs     map[string]string `json:"envs,omitempty" jsonschema_extras:"order=7" jsonschema_description:"Additional environment variables to inject into the worker"`
}

type V2WorkerModelOpenstackSpec struct {
	Image   string `json:"image" jsonschema_description:"Name of the openstack image"`
	Cmd     string `json:"cmd" jsonschema_description:"Command used by CDS to run the worker"`
	Flavor  string `json:"flavor" jsonschema_description:"Openstack flavor used by the worker model"`
	PreCmd  string `json:"pre_cmd,omitempty" jsonschema_description:"Pre command to execute just before running the CDS worker"`
	PostCmd string `json:"post_cmd" jsonschema_description:"Post command to shutdown the VM"`
}

type V2WorkerModelVSphereSpec struct {
	Image    string `json:"image" jsonschema_description:"Name of the vsphere template"`
	Username string `json:"username,omitempty" jsonschema_description:"Username to connect to the VM"`
	Password string `json:"password,omitempty" jsonschema_description:"Username password to connect to the VM"`
	Cmd      string `json:"cmd" jsonschema_description:"Command used by CDD to run the worker"`
	PreCmd   string `json:"pre_cmd,omitempty" jsonschema_description:"Pre command to execute run just before running the CDS worker"`
	PostCmd  string `json:"post_cmd" jsonschema_description:"Post command to shutdown the VM"`
}

func (wm V2WorkerModel) GetName() string {
	return wm.Name
}

func (wm V2WorkerModel) Lint() []error {
	multipleError := MultiError{}

	workerModelSchema := GetWorkerModelJsonSchema()
	workerModelSchemaS, err := workerModelSchema.MarshalJSON()
	if err != nil {
		multipleError.Append(WrapError(err, "unable to load worker model schema"))
		return multipleError
	}
	schemaLoader := gojsonschema.NewStringLoader(string(workerModelSchemaS))

	modelJson, err := json.Marshal(wm)
	if err != nil {
		multipleError.Append(WithStack(err))
		return multipleError
	}
	documentLoader := gojsonschema.NewStringLoader(string(modelJson))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		multipleError.Append(WithStack(err))
		return multipleError
	}
	if result.Valid() {
		return nil
	}
	for _, e := range result.Errors() {
		multipleError.Append(fmt.Errorf("%v", e))
	}
	return multipleError
}
