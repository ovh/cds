package sdk

// These are type of plugins
const (
	GRPCPluginDeploymentIntegration = "integration-deploy_application"
	GRPCPluginAction                = "action"
)

// GRPCPlugin is the type representing a plugin over GRPC
type GRPCPlugin struct {
	ID                 int64              `json:"id" yaml:"id" cli:"id" db:"id"`
	Name               string             `json:"name" yaml:"name" cli:"name,key" db:"name"`
	Type               string             `json:"type" yaml:"type" cli:"type" db:"type"`
	Author             string             `json:"author" yaml:"author" cli:"author" db:"author"`
	Description        string             `json:"description" yaml:"description" cli:"description" db:"description"`
	Parameters         []Parameter        `json:"parameters,omitempty" yaml:"parameters,omitempty" cli:"parameters" db:"-"`
	Binaries           []GRPCPluginBinary `json:"binaries" yaml:"binaries" cli:"-" db:"-"`
	IntegrationModelID *int64             `json:"-" db:"integration_model_id" yaml:"-" cli:"-"`
	Integration        string             `json:"integration" db:"-" yaml:"integration" cli:"integration"`
}

func (p *GRPCPlugin) Validate() error {
	if p.Name == "" || p.Type == "" || p.Author == "" || p.Description == "" {
		return NewErrorFrom(ErrPluginInvalid, "Invalid plugin: name, type, author and description are mandatory")
	}
	return nil
}

// GetBinary returns the binary for a specific os and arch
func (p GRPCPlugin) GetBinary(os, arch string) *GRPCPluginBinary {
	for _, b := range p.Binaries {
		if b.OS == os && b.Arch == arch {
			return &b
		}
	}
	return nil
}

// GRPCPluginBinary represents a binary file (for a specific os and arch) serving a GRPCPlugin
type GRPCPluginBinary struct {
	OS               string          `json:"os,omitempty" yaml:"os"`
	Arch             string          `json:"arch,omitempty" yaml:"arch"`
	Name             string          `json:"name,omitempty" yaml:"-"`
	ObjectPath       string          `json:"object_path,omitempty" yaml:"-"`
	Size             int64           `json:"size,omitempty" yaml:"-"`
	Perm             uint32          `json:"perm,omitempty" yaml:"-"`
	MD5sum           string          `json:"md5sum,omitempty" yaml:"-"`
	SHA512sum        string          `json:"sha512sum,omitempty" yaml:"-"`
	TempURL          string          `json:"temp_url,omitempty" yaml:"-"`
	TempURLSecretKey string          `json:"-" yaml:"-"`
	Entrypoints      []string        `json:"entrypoints,omitempty" yaml:"entrypoints"`
	Cmd              string          `json:"cmd,omitempty" yaml:"cmd"`
	Args             []string        `json:"args,omitempty" yaml:"args"`
	Requirements     RequirementList `json:"requirements,omitempty" yaml:"requirements"`
	FileContent      []byte          `json:"file_content,omitempty" yaml:"-"` //only used for upload
	PluginName       string          `json:"plugin_name,omitempty" yaml:"-"`
}

// GetName is a part of the objectstore.Object interface implementation
func (b GRPCPluginBinary) GetName() string {
	return b.Name
}

// GetPath is a part of the objectstore.Object interface implementation
func (b GRPCPluginBinary) GetPath() string {
	return b.Name + "-" + b.OS + "-" + b.Arch
}
