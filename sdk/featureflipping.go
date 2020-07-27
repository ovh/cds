package sdk

type Feature struct {
	ID   int64  `json:"id" db:"id" cli:"-" yaml:"-"`
	Name string `json:"name" db:"name" cli:"name" yaml:"name"`
	Rule string `json:"rule" db:"rule" cli:"-" yaml:"rule"`
}

type FeatureEnabledResponse struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}
