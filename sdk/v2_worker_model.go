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
	Name        string          `json:"name" cli:"name" jsonschema:"required,minLength=1" jsonschema_extras:"order=1"`
	From        string          `json:"from" jsonschema_extras:"order=3,disabled=true"`
	Description string          `json:"description,omitempty" jsonschema_extras:"order=2"`
	Type        string          `json:"type" cli:"type" jsonschema:"required,enum=docker,enum=openstack,enum=vsphere" jsonschema_extras:"order=4"`
	Spec        json.RawMessage `json:"spec" jsonschema:"required" jsonschema_allof_type:"type=docker:#/$defs/V2WorkerModelDockerSpec,type=openstack:#/$defs/V2WorkerModelOpenstackSpec,type=vsphere:#/$defs/V2WorkerModelVSphereSpec" jsonschema_extras:"order=5"`
}

type V2WorkerModelDockerSpec struct {
	Image    string            `json:"image" jsonschema:"required,minLength=1" jsonschema_extras:"order=1"`
	Registry string            `json:"registry,omitempty" jsonschema:"required,minLength=1" jsonschema_extras:"order=2"`
	Username string            `json:"username,omitempty" jsonschema_extras:"order=3"`
	Password string            `json:"password,omitempty" jsonschema_extras:"order=4"`
	Cmd      string            `json:"cmd,omitempty" jsonschema:"required,minLength=1" jsonschema_extras:"order=6"`
	Shell    string            `json:"shell,omitempty" jsonschema:"required,minLength=1" jsonschema_extras:"order=5"`
	Envs     map[string]string `json:"envs,omitempty" jsonschema_extras:"order=7"`
}

type V2WorkerModelOpenstackSpec struct {
	Image   string `json:"image" jsonschema:"required"`
	Cmd     string `json:"cmd,omitempty" jsonschema:"required"`
	Flavor  string `json:"flavor,omitempty" jsonschema:"required"`
	PreCmd  string `json:"pre_cmd,omitempty"`
	PostCmd string `json:"post_cmd,omitempty" jsonschema:"required"`
}

type V2WorkerModelVSphereSpec struct {
	Image    string `json:"image" jsonschema:"required"`
	Username string `json:"username,omitempty" jsonschema:"required"`
	Password string `json:"password,omitempty" jsonschema:"required"`
	Cmd      string `json:"cmd,omitempty" jsonschema:"required"`
	PreCmd   string `json:"pre_cmd,omitempty"`
	PostCmd  string `json:"post_cmd,omitempty" jsonschema:"required"`
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
