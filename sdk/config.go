package sdk

// DefaultValues contains default user values for init DB
type DefaultValues struct {
	DefaultGroupName string
}

// ConfigUser struct.
type ConfigUser struct {
	URLUI  string `json:"url.ui"`
	URLAPI string `json:"url.api"`
}

type APIConfig struct {
	DefaultRunRetentionPolicy    string `json:"default_run_retention_policy"`
	ProjectCreationDisabled      bool   `json:"project_creation_disabled"`
	ProjectInfoCreationDisabled  string `json:"project_info_creation_disabled,omitempty"`
	ProjectVCSManagementDisabled bool   `json:"project_vcs_management_disabled,omitempty"`
}

type TCPServer struct {
	Addr               string `toml:"addr" default:"" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
	Port               int    `toml:"port" default:"8090" json:"port"`
	GlobalTCPRateLimit int64  `toml:"globalTCPRateLimit" default:"2097152" comment:"Rate limit (B/s) for incoming logs" json:"globalTCPRateLimit"`
}

type RedisConf struct {
	Host                  string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
	Password              string `toml:"password" json:"-"`
	DbIndex               int    `toml:"dbindex" default:"0" json:"dbindex"`
	InsecureSkipVerifyTLS bool   `toml:"insecureSkipVerifyTLS" default:"false" json:"insecureSkipVerifyTLS"`
	EnableTLS             bool   `toml:"enableTLS" default:"false" json:"enableTLS"`
}
