package sanity

// Warning unique identifiers
const (
	_ = iota
	MultipleWorkerModelWarning
	NoWorkerModelMatchRequirement
	InvalidVariableFormat
	ProjectVariableDoesNotExist
	ApplicationVariableDoesNotExist
	EnvironmentVariableDoesNotExist
	CannotUseEnvironmentVariable
	MultipleHostnameRequirement
	IncompatibleBinaryAndModelRequirements
	IncompatibleServiceAndModelRequirements
	IncompatibleMemoryAndModelRequirements
	GitURLWithoutLinkedRepository
	GitURLWithoutKey
	EnvironmentVariableUsedInApplicationDoesNotExist
	InvalidVariableFormatUsedInApplication
	MissingEnvironment
)

var messageAmericanEnglish = map[int64]string{
	MultipleWorkerModelWarning:                       `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}} has multiple Worker Model as requirement. It will never start building.`,
	NoWorkerModelMatchRequirement:                    `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: No worker model matches all required binaries`,
	InvalidVariableFormat:                            `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Invalid variable format '{{index . "VarName"}}'`,
	ProjectVariableDoesNotExist:                      `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Project variable '{{index . "VarName"}}' used but doesn't exist`,
	ApplicationVariableDoesNotExist:                  `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Application variable '{{index . "VarName"}}' used but doesn't exist in application '{{index . "AppName"}}'`,
	EnvironmentVariableDoesNotExist:                  `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Environment variable {{index . "VarName"}} used but doesn't exist in all environments`,
	CannotUseEnvironmentVariable:                     `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Cannot use environment variable '{{index . "VarName"}} in a pipeline of type 'Build'`,
	MultipleHostnameRequirement:                      `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}} has multiple Hostname requirements. It will never start building.`,
	IncompatibleBinaryAndModelRequirements:           `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Model {{index . "ModelName"}} does not have the binary '{{index . "BinaryRequirement"}}' capability`,
	IncompatibleServiceAndModelRequirements:          `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Model {{index . "ModelName"}} cannot be linked to service '{{index . "ServiceRequirement"}}'`,
	IncompatibleMemoryAndModelRequirements:           `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Model {{index . "ModelName"}} cannot handle memory requirement`,
	GitURLWithoutLinkedRepository:                    `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}} is used but one one more applications are linked to any repository. Git clone will failed`,
	GitURLWithoutKey:                                 `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}} is used but no ssh key were found. Git clone will failed`,
	MissingEnvironment:                               `Application {{index . "ApplicationName"}}: At least one environment with one variable should be defined`,
	EnvironmentVariableUsedInApplicationDoesNotExist: `Application {{index . "ApplicationName"}}: Environment variable {{index . "VarName"}} used but doesn't exist in all environments`,
	InvalidVariableFormatUsedInApplication:           `Application {{index . "ApplicationName"}}: Invalid variable format '{{index . "VarName"}}'`}
