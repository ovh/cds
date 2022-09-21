package sdk

type Organization struct {
	ID   string `json:"id" db:"id" cli:"id"`
	Name string `json:"name" db:"name" cli:"name"`
}
