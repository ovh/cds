package sdk

// Hatchery registration model
type Hatchery struct {
	RatioService *int `json:"ratio_service" db:"-"`
}
