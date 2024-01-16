package sdk

import "time"

type ProjectSecret struct {
	ID           string    `json:"id" db:"id"`
	ProjectKey   string    `json:"project_key" db:"project_key"`
	Name         string    `json:"name" db:"name" cli:"name"`
	LastModified time.Time `json:"last_modified" db:"last_modified" cli:"last_modified"`
	Value        string    `json:"value" db:"encrypted_value" gorpmapping:"encrypted,ID,ProjectKey,Name"`
}
