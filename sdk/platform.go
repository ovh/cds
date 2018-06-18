package sdk

// This is the buitin platform model
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

// Clone return a copy of the config (with a copy of the underlying data structure)
func (config PlatformConfig) Clone() PlatformConfig {
	new := make(PlatformConfig, len(config))
	for k, v := range config {
		new[k] = v
	}
	return new
}

// EncryptSecrets encrypt secrets given a cypher func
func (config PlatformConfig) EncryptSecrets(encryptFunc func(string) (string, error)) error {
	for k, v := range config {
		if v.Type == PlatformConfigTypePassword {
			s, errS := encryptFunc(v.Value)
			if errS != nil {
				return WrapError(errS, "EncryptSecrets> Cannot encrypt password")
			}
			v.Value = s
			config[k] = v
		}
	}
	return nil
}

// DecryptSecrets decrypt secrets given a cypher func
func (config PlatformConfig) DecryptSecrets(decryptFunc func(string) (string, error)) error {
	for k, v := range config {
		if v.Type == PlatformConfigTypePassword {
			s, errS := decryptFunc(v.Value)
			if errS != nil {
				return WrapError(errS, "DecryptSecrets> Cannot descrypt password")
			}
			v.Value = s
			config[k] = v
		}
	}
	return nil
}

const (
	// PlatformConfigTypeString represents a string configuration value
	PlatformConfigTypeString = "string"
	// PlatformConfigTypePassword represents a password configuration value
	PlatformConfigTypePassword = "password"
)

// PlatformConfigValue represent a configuration value for a platform
type PlatformConfigValue struct {
	Value string `json:"value" yaml:"value"`
	Type  string `json:"type" yaml:"type"`
}

// PlatformModel represent a platform model with its default configuration
type PlatformModel struct {
	ID                      int64                     `json:"id" db:"id" yaml:"-" cli:"-"`
	Name                    string                    `json:"name" db:"name" yaml:"name" cli:"name,key"`
	Author                  string                    `json:"author" db:"author" yaml:"author" cli:"author"`
	Identifier              string                    `json:"identifier" db:"identifier" yaml:"identifier,omitempty"`
	Icon                    string                    `json:"icon" db:"icon" yaml:"icon"`
	DefaultConfig           PlatformConfig            `json:"default_config" db:"-" yaml:"default_config"`
	Disabled                bool                      `json:"disabled" db:"disabled" yaml:"disabled"`
	Hook                    bool                      `json:"hook" db:"hook" yaml:"hook" cli:"hooks_supported"`
	FileStorage             bool                      `json:"file_storage" db:"file_storage" yaml:"file_storage" cli:"file_storage supported"`
	BlockStorage            bool                      `json:"block_storage" db:"block_storage" yaml:"block_storage" cli:"block_storage supported"`
	Deployment              bool                      `json:"deployment" db:"deployment" yaml:"deployment" cli:"deployment_supported"`
	DeploymentDefaultConfig PlatformConfig            `json:"deployment_default_config" db:"-" yaml:"deployment_default_config"`
	Compute                 bool                      `json:"compute" db:"compute" yaml:"compute" cli:"compute_supported"`
	PluginID                *int64                    `json:"-" db:"grpc_plugin_id" yaml:"-"`
	PluginName              string                    `json:"plugin_name,omitempty" db:"-" yaml:"plugin,omitempty"`
	Public                  bool                      `json:"public,omitempty" db:"public" yaml:"public,omitempty"`
	PublicConfigurations    map[string]PlatformConfig `json:"public_configurations,omitempty" db:"-" yaml:"public_configurations"`
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

// HideSecrets replaces password with a placeholder
func (pf *ProjectPlatform) HideSecrets() {
	pf.Config.HideSecrets()
	pf.Model.DefaultConfig.HideSecrets()
	for k, cfg := range pf.Model.PublicConfigurations {
		cfg.HideSecrets()
		pf.Model.PublicConfigurations[k] = cfg
	}
	pf.Model.DeploymentDefaultConfig.HideSecrets()
}

// MergeWith merge two config
func (config PlatformConfig) MergeWith(cfg PlatformConfig) {
	for k, v := range cfg {
		val, has := config[k]
		if !has {
			val.Type = v.Type
		}
		if val.Type == PlatformConfigTypePassword && v.Value != PasswordPlaceholder {
			val.Value = v.Value
		}
		config[k] = val
	}
}

// HideSecrets replaces password with a placeholder
func (config *PlatformConfig) HideSecrets() {
	for k, v := range *config {
		if NeedPlaceholder(v.Type) {
			v.Value = PasswordPlaceholder
			(*config)[k] = v
		}
	}
}
