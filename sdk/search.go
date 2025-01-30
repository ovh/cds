package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type SearchResultType string

const (
	ProjectSearchResultType SearchResultType = "project"
)

type SearchResults []SearchResult

func (s *SearchResults) AppendProjects(ps ...Project) {
	for i := range ps {
		var found bool
		for _, r := range *s {
			if r.Type == ProjectSearchResultType && ps[i].Key == r.ID {
				found = true
				break
			}
		}
		if !found {
			*s = append(*s, SearchResult{
				Type: ProjectSearchResultType,
				ID:   ps[i].Key,
			})
		}
	}
}

type SearchResult struct {
	Type     SearchResultType     `json:"type"`
	ID       string               `json:"id"`
	Label    string               `json:"label"`
	Variants SearchResultVariants `json:"variants,omitempty"`
}

type SearchResultVariants []string

func (v SearchResultVariants) Value() (driver.Value, error) {
	names, err := json.Marshal(v)
	return names, WrapError(err, "cannot marshal SearchResultVariants")
}

func (v *SearchResultVariants) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, v), "cannot unmarshal SearchResultVariants")
}
