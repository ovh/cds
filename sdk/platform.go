package sdk

const (
	KafkaPlatformModel = "Kafka"
)

var (
	KafkaPlatform = PlatformModel{
		Name:       KafkaPlatformModel,
		Author:     "CDS",
		Identifier: "github.com/ovh/cds/platform/builtin/kafka",
		Icon:       "",
		DefaultConfig: PlatformConfig{
			"broker": "",
		},
		Disabled: false,
		Hook:     true,
	}
)

type PlatformConfig map[string]string

type PlatformModel struct {
	ID            int64          `json:"id" db:"id"`
	Name          string         `json:"name" db:"name"`
	Author        string         `json:"author" db:"author"`
	Identifier    string         `json:"identifier" db:"identifier"`
	Icon          string         `json:"icon" db:"icon"`
	DefaultConfig PlatformConfig `json:"default_config" db:"-"`
	Disabled      bool           `json:"disabled" db:"disabled"`
	Hook          bool           `json:"hook" db:"hook"`
	FileStorage   bool           `json:"fileStorage" db:"file_storage"`
	BlockStorage  bool           `json:"blockStorage" db:"block_storage"`
	Deployment    bool           `json:"deployment" db:"deployment"`
	Compute       bool           `json:"compute" db:"compute"`
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
