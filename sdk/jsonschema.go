package sdk

import (
	"github.com/invopop/jsonschema"
)

func GetWorkerModelJsonSchema() *jsonschema.Schema {
	wmSchema := jsonschema.Reflect(&V2WorkerModel{})
	wmDocker := jsonschema.Reflect(&V2WorkerModelDockerSpec{})
	wmOpenstack := jsonschema.Reflect(&V2WorkerModelOpenstackSpec{})
	wmVSphere := jsonschema.Reflect(&V2WorkerModelVSphereSpec{})

	if wmSchema.Definitions == nil {
		wmSchema.Definitions = make(map[string]*jsonschema.Schema)
	}
	wmSchema.Definitions["V2WorkerModelVSphereSpec"] = wmVSphere
	wmSchema.Definitions["V2WorkerModelOpenstackSpec"] = wmOpenstack
	wmSchema.Definitions["V2WorkerModelDockerSpec"] = wmDocker
	return wmSchema
}

func GetWorkerModelTemplateJsonSchema() *jsonschema.Schema {
	wmtSchema := jsonschema.Reflect(&WorkerModelTemplate{})
	wmtDocker := jsonschema.Reflect(&WorkerModelTemplateDocker{})
	wmtVM := jsonschema.Reflect(&WorkerModelTemplateVM{})

	if wmtSchema.Definitions == nil {
		wmtSchema.Definitions = make(map[string]*jsonschema.Schema)
	}
	wmtSchema.Definitions["WorkerModelTemplateDocker"] = wmtDocker
	wmtSchema.Definitions["WorkerModelTemplateVM"] = wmtVM
	return wmtSchema
}
