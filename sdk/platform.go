package sdk

import (
	"runtime"
)

const (
	KafkaPlatformModel = "Kafka"
)

// Here are the default hooks
var (
	BuiltinPlatformModels = []*WorkflowHookModel{
		&WebHookModel,
		&RepositoryWebHookModel,
		&GitPollerModel,
		&SchedulerModel,
		&KafkaHookModel,
	}
)

var (
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

type PlatformModelPluginBinary struct {
	Size       int64  `json:"size"`
	Perm       uint32 `json:"perm"`
	MD5sum     string `json:"md5sum"`
	ObjectPath string `json:"object_path"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
}

type PlatformModelPlugin struct {
	Name     string                               `json:"name"`
	Version  string                               `json:"version"`
	Binaries map[string]PlatformModelPluginBinary `json:"binaries"`
}

func (p *PlatformModelPlugin) Binary() *PlatformModelPluginBinary {
	if p.Binaries == nil {
		return nil
	}

	b, ok := p.Binaries[runtime.GOOS+"-"+runtime.GOARCH]
	if !ok {
		return nil
	}
	return &b
}

func (p *PlatformModelPlugin) AddBinary(os, arch string, b PlatformModelPluginBinary) {
	if p.Binaries == nil {
		p.Binaries = map[string]PlatformModelPluginBinary{}
	}

	p.Binaries[os+"-"+arch] = b
}

// PlatformModel represent a platform model with its default configuration
type PlatformModel struct {
	ID                  int64                `json:"id" db:"id" yaml:"-" cli:"-"`
	Name                string               `json:"name" db:"name" yaml:"name" cli:"name,key"`
	Author              string               `json:"author" db:"author" yaml:"author" cli:"author"`
	Identifier          string               `json:"identifier" db:"identifier" yaml:"identifier,omitempty"`
	Icon                string               `json:"icon" db:"icon" yaml:"icon"`
	DefaultConfig       PlatformConfig       `json:"default_config" db:"-" yaml:"default_config"`
	Disabled            bool                 `json:"disabled" db:"disabled" yaml:"disabled"`
	Hook                bool                 `json:"hook" db:"hook" yaml:"hook" cli:"hooks_supported"`
	FileStorage         bool                 `json:"file_storage" db:"file_storage" yaml:"file_storage" cli:"file_storage supported"`
	BlockStorage        bool                 `json:"block_storage" db:"block_storage" yaml:"block_storage" cli:"block_storage supported"`
	Deployment          bool                 `json:"deployment" db:"deployment" yaml:"deployment" cli:"deployment_supported"`
	Compute             bool                 `json:"compute" db:"compute" yaml:"compute" cli:"compute_supported"`
	PlatformModelPlugin *PlatformModelPlugin `json:"platform_model_plugin,omitempty" db:"-" yaml:"-"`
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
	ID              int64          `json:"id" db:"id"`
	ProjectID       int64          `json:"project_id" db:"project_id"`
	Name            string         `json:"name" db:"name"`
	PlatformModelID int64          `json:"platform_model_id" db:"platform_model_id"`
	Model           PlatformModel  `json:"model" db:"-"`
	Config          PlatformConfig `json:"config" db:"-"`
}
