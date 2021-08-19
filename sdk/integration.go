package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// This is the buitin integration model
const (
	KafkaIntegrationModel         = "Kafka"
	RabbitMQIntegrationModel      = "RabbitMQ"
	OpenstackIntegrationModel     = "Openstack"
	AWSIntegrationModel           = "AWS"
	DefaultStorageIntegrationName = "shared.infra"
	ArtifactManagerModel          = "ArtifactManager"

	ArtifactManagerConfigPlatform              = "platform"
	ArtifactManagerConfigURL                   = "url"
	ArtifactManagerConfigTokenName             = "token.name"
	ArtifactManagerConfigToken                 = "token"
	ArtifactManagerConfigReleaseToken          = "release.token"
	ArtifactManagerConfigCdsRepository         = "cds.repository"
	ArtifactProjectKey                         = "project.key"
	ArtifactManagerConfigPromotionLowMaturity  = "promotion.maturity.low"
	ArtifactManagerConfigPromotionHighMaturity = "promotion.maturity.high"
	ArtifactManagerConfigBuildInfoPath         = "build.info.path"
)

// Here are the default plateform models
var (
	BuiltinIntegrationModels = []*IntegrationModel{
		&KafkaIntegration,
		&RabbitMQIntegration,
		&OpenstackIntegration,
		&AWSIntegration,
		&ArtifactManagerIntegration,
	}
	// KafkaIntegration represents a kafka integration
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
			"topic": IntegrationConfigValue{
				Type:        IntegrationConfigTypeString,
				Description: "This is mandatory only if you want to use Event Integration",
			},
		},
		Disabled: false,
		Hook:     true,
		Event:    true,
	}
	// RabbitMQIntegration represents a kafka integration
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
	// OpenstackIntegration represents an openstack integration
	OpenstackIntegration = IntegrationModel{
		Name:       OpenstackIntegrationModel,
		Author:     "CDS",
		Identifier: "github.com/ovh/cds/integration/builtin/openstack",
		Icon:       "",
		DefaultConfig: IntegrationConfig{
			"address": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"region": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"domain": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"tenant_name": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"username": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"password": IntegrationConfigValue{
				Type: IntegrationConfigTypePassword,
			},
			"storage_container_prefix": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"storage_temporary_url_supported": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
		},
		Storage:  true,
		Disabled: false,
		Hook:     false,
	}
	// ArtifactManagerIntegration represents an artifact manager integration (like artifactory)
	ArtifactManagerIntegration = IntegrationModel{
		Name:       ArtifactManagerModel,
		Author:     "CDS",
		Identifier: "github.com/ovh/cds/integration/builtin/artifact-manager",
		Icon:       "",
		DefaultConfig: IntegrationConfig{
			ArtifactManagerConfigPlatform: IntegrationConfigValue{
				Type:        IntegrationConfigTypeString,
				Description: "Only 'artifactory' is implemented",
			},
			ArtifactManagerConfigURL: IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			ArtifactManagerConfigTokenName: IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			ArtifactManagerConfigToken: IntegrationConfigValue{
				Type: IntegrationConfigTypePassword,
			},
			ArtifactManagerConfigReleaseToken: IntegrationConfigValue{
				Type: IntegrationConfigTypePassword,
			},
			ArtifactProjectKey: IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			ArtifactManagerConfigCdsRepository: IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			ArtifactManagerConfigPromotionLowMaturity: IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			ArtifactManagerConfigPromotionHighMaturity: IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
		},
		AdditionalDefaultConfig: IntegrationConfig{
			ArtifactManagerConfigBuildInfoPath: IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
		},
		ArtifactManager: true,
	}
	// AWSIntegration represents an aws integration
	AWSIntegration = IntegrationModel{
		Name:       AWSIntegrationModel,
		Author:     "CDS",
		Identifier: "github.com/ovh/cds/integration/builtin/aws",
		Icon:       "",
		DefaultConfig: IntegrationConfig{
			"region": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"bucket_name": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"prefix": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"access_key_id": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"secret_access_key": IntegrationConfigValue{
				Type: IntegrationConfigTypePassword,
			},
			"endpoint": IntegrationConfigValue{
				Type: IntegrationConfigTypeString,
			},
			"disable_ssl": IntegrationConfigValue{
				Type: IntegrationConfigTypeBoolean,
			},
			"force_path_style": IntegrationConfigValue{
				Type: IntegrationConfigTypeBoolean,
			},
		},
		Storage:  true,
		Disabled: false,
		Hook:     false,
	}
)

// IntegrationType represents all different type of integrations
type IntegrationType string

const (
	IntegrationTypeEvent      = IntegrationType("event")
	IntegrationTypeCompute    = IntegrationType("compute")
	IntegrationTypeHook       = IntegrationType("hook")
	IntegrationTypeStorage    = IntegrationType("storage")
	IntegrationTypeDeployment = IntegrationType("deployment")
)

// DefaultIfEmptyStorage return sdk.DefaultStorageIntegrationName if integrationName is empty
func DefaultIfEmptyStorage(integrationName string) string {
	if integrationName == "" {
		return DefaultStorageIntegrationName
	}
	return integrationName
}

// IntegrationConfig represent the configuration of an integration
type IntegrationConfig map[string]IntegrationConfigValue

func (config IntegrationConfig) Blur() {
	for k, v := range config {
		if v.Type == IntegrationConfigTypePassword {
			config[k] = IntegrationConfigValue{
				Type:        v.Type,
				Description: v.Description,
				Value:       PasswordPlaceholder,
			}
		}
	}
}

// Clone return a copy of the config (with a copy of the underlying data structure)
func (config IntegrationConfig) Clone() IntegrationConfig {
	new := make(IntegrationConfig, len(config))
	for k, v := range config {
		new[k] = v
	}
	return new
}

// Value returns driver.Value from IntegrationConfig.
func (config IntegrationConfig) Value() (driver.Value, error) {
	j, err := json.Marshal(config)
	return j, WrapError(err, "cannot marshal IntegrationConfig")
}

// Scan IntegrationConfig.
func (config *IntegrationConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, config), "cannot unmarshal IntegrationConfig")
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
	// IntegrationConfigTypeBoolean represents a password configuration value
	IntegrationConfigTypeBoolean = "boolean"

	IntegrationVariablePrefixDeployment      = "deployment"
	IntegrationVariablePrefixArtifactManager = "artifact_manager"
)

// IntegrationConfigValue represent a configuration value for a integration
type IntegrationConfigValue struct {
	Value       string `json:"value" yaml:"value"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type IntegrationConfigMap map[string]IntegrationConfig

func (config IntegrationConfigMap) Clone() IntegrationConfigMap {
	new := make(IntegrationConfigMap, len(config))
	for k, v := range config {
		new[k] = v.Clone()
	}
	return new
}

func GetIntegrationVariablePrefix(model IntegrationModel) string {
	if model.Deployment {
		return IntegrationVariablePrefixDeployment
	}
	if model.ArtifactManager {
		return IntegrationVariablePrefixArtifactManager
	}
	return ""
}

func AllowIntegrationInVariable(model IntegrationModel) bool {
	return model.ArtifactManager || model.Deployment
}

func (config IntegrationConfigMap) Blur() {
	for _, v := range config {
		v.Blur()
	}
}

// Value returns driver.Value from IntegrationConfig.
func (config IntegrationConfigMap) Value() (driver.Value, error) {
	j, err := json.Marshal(config)
	return j, WrapError(err, "cannot marshal IntegrationConfigMap")
}

// Scan IntegrationConfig.
func (config *IntegrationConfigMap) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, config), "cannot unmarshal IntegrationConfigMap")
}

// IntegrationModel represent a integration model with its default configuration
type IntegrationModel struct {
	ID                      int64                `json:"id" db:"id" yaml:"-" cli:"-"`
	Name                    string               `json:"name" db:"name" yaml:"name" cli:"name,key"`
	Author                  string               `json:"author" db:"author" yaml:"author" cli:"author"`
	Identifier              string               `json:"identifier" db:"identifier" yaml:"identifier,omitempty"`
	Icon                    string               `json:"icon" db:"icon" yaml:"icon"`
	DefaultConfig           IntegrationConfig    `json:"default_config" db:"default_config" yaml:"default_config"`
	AdditionalDefaultConfig IntegrationConfig    `json:"additional_default_config" db:"additional_default_config" yaml:"additional_default_config"`
	PublicConfigurations    IntegrationConfigMap `json:"public_configurations,omitempty" db:"cipher_public_configurations" yaml:"public_configurations"`
	Disabled                bool                 `json:"disabled" db:"disabled" yaml:"disabled"`
	Hook                    bool                 `json:"hook" db:"hook" yaml:"hook" cli:"hooks_supported"`
	Storage                 bool                 `json:"storage" db:"storage" yaml:"storage" cli:"storage supported"`
	Deployment              bool                 `json:"deployment" db:"deployment" yaml:"deployment" cli:"deployment_supported"`
	Compute                 bool                 `json:"compute" db:"compute" yaml:"compute" cli:"compute_supported"`
	Event                   bool                 `json:"event" db:"event" yaml:"event" cli:"event_supported"`
	ArtifactManager         bool                 `json:"artifact_manager" db:"artifact_manager" yaml:"artifact_manager" cli:"artifact_manager_supported"`
	Public                  bool                 `json:"public,omitempty" db:"public" yaml:"public,omitempty"`
}

func (p *IntegrationModel) Blur() {
	p.PublicConfigurations.Blur()
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
	Config             IntegrationConfig `json:"config" db:"cipher_config" yaml:"config" gorpmapping:"encrypted,ProjectID,IntegrationModelID"`
	// GRPCPlugin field is used to get all plugins associatied to an integration
	// when we GET /project/{permProjectKey}/integrations/{integrationName}
	GRPCPlugins []GRPCPlugin `json:"integration_plugins,omitempty" db:"-" yaml:"-"`
}

// Blur replaces password with a placeholder
func (pf *ProjectIntegration) Blur() {
	pf.Config.Blur()
	pf.Model.DefaultConfig.Blur()
	pf.Model.PublicConfigurations.Blur()
	pf.Model.AdditionalDefaultConfig.Blur()
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

type FileInfo struct {
	Size int64
	Md5  string
	Type string
}
