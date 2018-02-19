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

// Platform is an instanciation of a platform model
type Platform struct {
	ID              int64
	ProjectID       int64
	Name            string
	PlatformModelID int64
	Model           PlatformModel
	Config          PlatformConfig
}
