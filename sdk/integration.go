package sdk

// This is the buitin integration model
const (
	KafkaIntegrationModel    = "Kafka"
	RabbitMQIntegrationModel = "RabbitMQ"
)

// Here are the default plateform models
var (
	BuiltinIntegrationModels = []*IntegrationModel{
		&KafkaIntegration,
		&RabbitMQIntegration,
	}
	// KafkaIntegration represent a kafka integration
	KafkaIntegration = IntegrationModel{
		Name:       KafkaIntegrationModel,
		Author:     "CDS",
		Identifier: "github.com/ovh/cds/integration/builtin/kafka",
		Icon:       "",
		DefaultConfig: IntegrationConfig{
			"broker url": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"username": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"password": IntegrationConfigValue{
				Type: IntegrationConfigTypePassword,
			},
		},
		Disabled: false,
		Hook:     true,
	}
	// RabbitMQIntegration represent a kafka integration
	RabbitMQIntegration = IntegrationModel{
		Name:       RabbitMQIntegrationModel,
		Author:     "CDS",
		Identifier: "github.com/ovh/cds/integration/builtin/rabbitmq",
		Icon:       "",
		DefaultConfig: IntegrationConfig{
			"uri": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"username": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"password": IntegrationConfigValue{
				Type: IntegrationConfigTypePassword,
			},
		},
		Disabled: false,
		Hook:     true,
	}
)

// IntegrationConfig represent the configuration of a plateform
type IntegrationConfig map[string]IntegrationConfigValue

// Clone return a copy of the config (with a copy of the underlying data structure)
func (config IntegrationConfig) Clone() IntegrationConfig {
	new := make(IntegrationConfig, len(config))
	for k, v := range config {
		new[k] = v
	}
	return new
}

// EncryptSecrets encrypt secrets given a cypher func
func (config IntegrationConfig) EncryptSecrets(encryptFunc func(string) (string, error)) error {
	for k, v := range config {
		if v.Type == IntegrationConfigTypePassword {
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
func (config IntegrationConfig) DecryptSecrets(decryptFunc func(string) (string, error)) error {
	for k, v := range config {
		if v.Type == IntegrationConfigTypePassword {
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
	// IntegrationConfigTypeString represents a string configuration value
	IntegrationConfigTypeString = "string"
	// IntegrationConfigTypeText represents a text configuration value
	IntegrationConfigTypeText = "text"
	// IntegrationConfigTypePassword represents a password configuration value
	IntegrationConfigTypePassword = "password"
)

// IntegrationConfigValue represent a configuration value for a integration
type IntegrationConfigValue struct {
	Value       string `json:"value" yaml:"value"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// IntegrationModel represent a integration model with its default configuration
type IntegrationModel struct {
	ID                      int64                        `json:"id" db:"id" yaml:"-" cli:"-"`
	Name                    string                       `json:"name" db:"name" yaml:"name" cli:"name,key"`
	Author                  string                       `json:"author" db:"author" yaml:"author" cli:"author"`
	Identifier              string                       `json:"identifier" db:"identifier" yaml:"identifier,omitempty"`
	Icon                    string                       `json:"icon" db:"icon" yaml:"icon"`
	DefaultConfig           IntegrationConfig            `json:"default_config" db:"-" yaml:"default_config"`
	DeploymentDefaultConfig IntegrationConfig            `json:"deployment_default_config" db:"-" yaml:"deployment_default_config"`
	PublicConfigurations    map[string]IntegrationConfig `json:"public_configurations,omitempty" db:"-" yaml:"public_configurations"`
	Disabled                bool                         `json:"disabled" db:"disabled" yaml:"disabled"`
	Hook                    bool                         `json:"hook" db:"hook" yaml:"hook" cli:"hooks_supported"`
	Storage                 bool                         `json:"storage" db:"storage" yaml:"storage" cli:"storage supported"`
	Deployment              bool                         `json:"deployment" db:"deployment" yaml:"deployment" cli:"deployment_supported"`
	Compute                 bool                         `json:"compute" db:"compute" yaml:"compute" cli:"compute_supported"`
	Public                  bool                         `json:"public,omitempty" db:"public" yaml:"public,omitempty"`
}

//IsBuiltin checks is the model is builtin or not
func (p IntegrationModel) IsBuiltin() bool {
	for _, m := range BuiltinIntegrationModels {
		if p.Name == m.Name {
			return true
		}
	}
	return false
}

// ProjectIntegration is an instanciation of a integration model
type ProjectIntegration struct {
	ID                 int64             `json:"id" db:"id" yaml:"-"`
	ProjectID          int64             `json:"project_id" db:"project_id" yaml:"-"`
	Name               string            `json:"name" db:"name" cli:"name,key" yaml:"name"`
	IntegrationModelID int64             `json:"integration_model_id" db:"integration_model_id" yaml:"-"`
	Model              IntegrationModel  `json:"model" db:"-" yaml:"model"`
	Config             IntegrationConfig `json:"config" db:"-" yaml:"config"`
	// GRPCPlugin field is used to get all plugins associatied to an integration
	// when we GET /project/{permProjectKey}/integrations/{integrationName}
	GRPCPlugins []GRPCPlugin `json:"integration_plugins,omitempty" db:"-" yaml:"-"`
}

// HideSecrets replaces password with a placeholder
func (pf *ProjectIntegration) HideSecrets() {
	pf.Config.HideSecrets()
	pf.Model.DefaultConfig.HideSecrets()
	for k, cfg := range pf.Model.PublicConfigurations {
		cfg.HideSecrets()
		pf.Model.PublicConfigurations[k] = cfg
	}
	pf.Model.DeploymentDefaultConfig.HideSecrets()
}

// MergeWith set new values from new config and update existing values if not default.
func (config IntegrationConfig) MergeWith(cfg IntegrationConfig) {
	for k, v := range cfg {
		val, has := config[k]
		if !has {
			val.Type = v.Type
		}
		if val.Type != IntegrationConfigTypePassword || (val.Type == IntegrationConfigTypePassword && v.Value != PasswordPlaceholder) {
			val.Value = v.Value
		}
		config[k] = val
	}
}

// HideSecrets replaces password with a placeholder
func (config *IntegrationConfig) HideSecrets() {
	for k, v := range *config {
		if NeedPlaceholder(v.Type) {
			v.Value = PasswordPlaceholder
			(*config)[k] = v
		}
	}
}
