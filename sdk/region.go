package sdk

type Region struct {
	ID   string `json:"id" db:"id" cli:"id"`
	Name string `json:"name" db:"name" cli:"name"`
}
