package sdk

const (
	KafkaPlatformModel = "Kafka"
)

// Here are the default hooks
var (
	BuiltinHookModels = []*WorkflowHookModel{
		&WebHookModel,
		&RepositoryWebHookModel,
		&GitPollerModel,
		&SchedulerModel,
		&KafkaHookModel,
	}

	BuiltinPlatformModels = []*PlatformModel{
		&KafkaPlatform,
	}
	// KafkaPlatform represent a kafka platform
	KafkaPlatform = PlatformModel{
		Name:       KafkaPlatformModel,
		Author:     "CDS",
		Identifier: "github.com/ovh/cds/platform/builtin/kafka",
		Icon:       "",
		DefaultConfig: PlatformConfig{
			"broker url": PlatformConfigValue{
				Type: PlatformConfigTypeString,
			},
			"username": PlatformConfigValue{
				Type: PlatformConfigTypeString,
			},
			"password": PlatformConfigValue{
				Type: PlatformConfigTypePassword,
			},
		},
		Disabled: false,
		Hook:     true,
	}
)

// PlatformConfig represent the configuration of a plateform
type PlatformConfig map[string]PlatformConfigValue

const (
	// PlatformConfigTypeString represents a string configuration value
	PlatformConfigTypeString = "string"
	// PlatformConfigTypeString represents a password configuration value
	PlatformConfigTypePassword = "password"
)

// PlatformConfigValue represent a configuration value for a platform
type PlatformConfigValue struct {
	Value string `json:"value" yaml:"value"`
	Type  string `json:"type" yaml:"type"`
}

// PlatformModel represent a platform model with its default configuration
type PlatformModel struct {
	ID                      int64          `json:"id" db:"id" yaml:"-" cli:"-"`
	Name                    string         `json:"name" db:"name" yaml:"name" cli:"name,key"`
	Author                  string         `json:"author" db:"author" yaml:"author" cli:"author"`
	Identifier              string         `json:"identifier" db:"identifier" yaml:"identifier,omitempty"`
	Icon                    string         `json:"icon" db:"icon" yaml:"icon"`
	DefaultConfig           PlatformConfig `json:"default_config" db:"-" yaml:"default_config"`
	Disabled                bool           `json:"disabled" db:"disabled" yaml:"disabled"`
	Hook                    bool           `json:"hook" db:"hook" yaml:"hook" cli:"hooks_supported"`
	FileStorage             bool           `json:"file_storage" db:"file_storage" yaml:"file_storage" cli:"file_storage supported"`
	BlockStorage            bool           `json:"block_storage" db:"block_storage" yaml:"block_storage" cli:"block_storage supported"`
	Deployment              bool           `json:"deployment" db:"deployment" yaml:"deployment" cli:"deployment_supported"`
	DeploymentDefaultConfig PlatformConfig `json:"deployment_default_config" db:"-" yaml:"deployment_default_config"`
	Compute                 bool           `json:"compute" db:"compute" yaml:"compute" cli:"compute_supported"`
	PluginID                *int64         `json:"-" db:"grpc_plugin_id" yaml:"-"`
	PluginName              string         `json:"plugin_name,omitempty" db:"-" yaml:"plugin,omitempty"`
}

//IsBuiltin checks is the model is builtin or not
func (p PlatformModel) IsBuiltin() bool {
	for _, m := range BuiltinPlatformModels {
		if p.Name == m.Name {
			return true
		}
	}
	return false
}

// ProjectPlatform is an instanciation of a platform model
type ProjectPlatform struct {
	ID              int64          `json:"id" db:"id" yaml:"-"`
	ProjectID       int64          `json:"project_id" db:"project_id" yaml:"-"`
	Name            string         `json:"name" db:"name" cli:"name,key" yaml:"name"`
	PlatformModelID int64          `json:"platform_model_id" db:"platform_model_id" yaml:"-"`
	Model           PlatformModel  `json:"model" db:"-" yaml:"model"`
	Config          PlatformConfig `json:"config" db:"-" yaml:"config"`
}

// MergeWith merge two config
func (config *PlatformConfig) MergeWith(cfg PlatformConfig) {
	for k, v := range cfg {
		(*config)[k] = v
	}
}
