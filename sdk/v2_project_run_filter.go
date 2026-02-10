package sdk

import (
	"time"
)

// ProjectRunFilter représente un filtre de workflow run partagé au niveau d'un projet
type ProjectRunFilter struct {
	ID           string    `json:"id" db:"id" cli:"id"`
	ProjectKey   string    `json:"project_key" db:"project_key" cli:"project_key"`
	Name         string    `json:"name" db:"name" cli:"name"`
	Value        string    `json:"value" db:"value" cli:"value"`
	Sort         string    `json:"sort,omitempty" db:"sort" cli:"sort"`
	Order        int64     `json:"order" db:"order" cli:"order"`
	LastModified time.Time `json:"last_modified" db:"last_modified" cli:"last_modified"`
}

// Check valide la structure d'un ProjectRunFilter
func (f *ProjectRunFilter) Check() error {
	// Validation du nom
	if f.Name == "" {
		return NewErrorFrom(ErrWrongRequest, "filter name is required")
	}
	if len(f.Name) > 100 {
		return NewErrorFrom(ErrWrongRequest, "filter name must be less than 100 characters")
	}
	// Les noms peuvent contenir n'importe quel caractère UTF-8 (emojis, icônes, etc.)
	// Pas de validation regex restrictive

	// Validation de la valeur
	if f.Value == "" {
		return NewErrorFrom(ErrWrongRequest, "filter value is required")
	}

	// Validation du tri (optionnel)
	if f.Sort != "" {
		validSorts := []string{"started:asc", "started:desc", "last_modified:asc", "last_modified:desc"}
		valid := false
		for _, s := range validSorts {
			if f.Sort == s {
				valid = true
				break
			}
		}
		if !valid {
			return NewErrorFrom(ErrWrongRequest, "invalid sort value")
		}
	}

	// Validation de l'ordre
	if f.Order < 0 {
		return NewErrorFrom(ErrWrongRequest, "order must be positive or zero")
	}

	return nil
}
