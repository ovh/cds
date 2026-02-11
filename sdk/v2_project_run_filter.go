package sdk

import (
	"time"
)

// ProjectRunFilter represents a workflow run filter shared at project level
type ProjectRunFilter struct {
	ID           string    `json:"id" db:"id" cli:"id"`
	ProjectKey   string    `json:"project_key" db:"project_key" cli:"project_key"`
	Name         string    `json:"name" db:"name" cli:"name"`
	Value        string    `json:"value" db:"value" cli:"value"`
	Sort         string    `json:"sort,omitempty" db:"sort" cli:"sort"`
	Order        int64     `json:"order" db:"order" cli:"order"`
	LastModified time.Time `json:"last_modified" db:"last_modified" cli:"last_modified"`
}

// Check validates the structure of a ProjectRunFilter
func (f *ProjectRunFilter) Check() error {
	// Name validation
	if f.Name == "" {
		return NewErrorFrom(ErrWrongRequest, "filter name is required")
	}
	if len(f.Name) > 100 {
		return NewErrorFrom(ErrWrongRequest, "filter name must be less than 100 characters")
	}
	// Names can contain any UTF-8 character (emojis, icons, etc.)
	// No restrictive regex validation

	// Value validation
	if f.Value == "" {
		return NewErrorFrom(ErrWrongRequest, "filter value is required")
	}

	// Sort validation (optional)
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

	// Order validation
	if f.Order < 0 {
		return NewErrorFrom(ErrWrongRequest, "order must be positive or zero")
	}

	return nil
}
