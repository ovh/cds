package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type SearchResultType string

const (
	ProjectSearchResultType        SearchResultType = "project"
	WorkflowSearchResultType       SearchResultType = "workflow"
	WorkflowLegacySearchResultType SearchResultType = "workflow-legacy"
)

type SearchResults []SearchResult

type SearchResult struct {
	Type        SearchResultType     `json:"type"`
	ID          string               `json:"id"`
	Label       string               `json:"label"`
	Description string               `json:"description,omitempty"`
	Variants    SearchResultVariants `json:"variants,omitempty"`
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

var searchQueryLikeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// SearchQueryMaxWords bounds the number of words kept from a free text search. Every word costs one
// more LIKE comparison per scanned row, so an oversized query must not be able to multiply the cost
// of a search. Extra words are dropped.
const SearchQueryMaxWords = 20

// SearchQueryWords are the words a free text search is made of, lowered, without duplicates and
// bounded. They are what has to be matched, whether by a database or in memory, so both read them
// from here.
func SearchQueryWords(query string) []string {
	words := make([]string, 0, SearchQueryMaxWords)
	seen := make(map[string]struct{})
	for _, w := range strings.Fields(strings.ToLower(query)) {
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		words = append(words, w)
		if len(words) == SearchQueryMaxWords {
			break
		}
	}
	return words
}

// SearchQueryLikePatterns turns a free text search into one lowered SQL LIKE pattern per word.
// All the returned patterns have to match, so words can be given in any order, and the LIKE
// wildcards are escaped to be matched literally. An empty query returns no pattern at all.
func SearchQueryLikePatterns(query string) []string {
	words := SearchQueryWords(query)
	patterns := make([]string, 0, len(words))
	for _, w := range words {
		patterns = append(patterns, "%"+searchQueryLikeEscaper.Replace(w)+"%")
	}
	return patterns
}
