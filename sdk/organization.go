package sdk

type Organization struct {
	ID   string `json:"id" db:"id" cli:"id"`
	Name string `json:"name" db:"name" cli:"name"`
}

type OrganizationRegion struct {
	ID             string `json:"id" db:"id"`
	OrganizationID string `json:"organization_id" db:"organization_id"`
	RegionID       string `json:"region_id" db:"region_id"`
}
