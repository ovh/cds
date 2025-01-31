package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type SearchResultType string

const (
	ProjectSearchResultType        SearchResultType = "project"
	WorkflowSearchResultType       SearchResultType = "workflow"
	WorkflowLegacySearchResultType SearchResultType = "workflow-legacy"
)

type SearchResults []SearchResult

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

type SearchFilter struct {
	Key     string   `json:"key"`
	Options []string `json:"options"`
	Example string   `json:"example"`
}
