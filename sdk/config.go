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
