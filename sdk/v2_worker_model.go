package sdk

import (
	"encoding/json"
	"fmt"
	"github.com/xeipuuv/gojsonschema"
)

type V2WorkerModel struct {
	Name        string          `json:"name" cli:"name" jsonschema:"required"`
	From        string          `json:"from"`
	Description string          `json:"description,omitempty"`
	Type        string          `json:"type" cli:"type" jsonschema:"required"`
	Spec        json.RawMessage `json:"spec" jsonschema:"required" jsonschema_allof_type:"type=docker:#/$defs/V2WorkerModelDockerSpec,type=openstack:#/$defs/V2WorkerModelOpenstackSpec,type=vsphere:#/$defs/V2WorkerModelVSphereSpec"`
}

type V2WorkerModelDockerSpec struct {
	Image    string            `json:"image" jsonschema:"required"`
	Registry string            `json:"registry,omitempty" jsonschema:"required"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Cmd      string            `json:"cmd,omitempty" jsonschema:"required"`
	Shell    string            `json:"shell,omitempty" jsonschema:"required"`
	Envs     map[string]string `json:"envs,omitempty"`
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
