package sdk

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchQueryLikePatterns(t *testing.T) {
	for _, tc := range []struct {
		name     string
		query    string
		expected []string
	}{
		{"empty query gives no pattern", "", []string{}},
		{"blank query gives no pattern", "   ", []string{}},
		{"one word", "awesome", []string{"%awesome%"}},
		{"one pattern per word", "awesome main", []string{"%awesome%", "%main%"}},
		{"extra spaces are ignored", "  awesome   main  ", []string{"%awesome%", "%main%"}},
		{"words are lowered", "AwEsOme MAIN", []string{"%awesome%", "%main%"}},
		{"percent is escaped", "50%", []string{`%50\%%`}},
		{"underscore is escaped", "my_run", []string{`%my\_run%`}},
		{"backslash is escaped", `a\b`, []string{`%a\\b%`}},
		{"repeated words are deduplicated", "main main MAIN", []string{"%main%"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, SearchQueryLikePatterns(tc.query))
		})
	}
}

func TestSearchQueryLikePatternsIsBounded(t *testing.T) {
	var words []string
	for i := 0; i < SearchQueryMaxWords*10; i++ {
		words = append(words, fmt.Sprintf("word%d", i))
	}
	patterns := SearchQueryLikePatterns(strings.Join(words, " "))
	require.Len(t, patterns, SearchQueryMaxWords)
	require.Equal(t, "%word0%", patterns[0])
	require.Equal(t, fmt.Sprintf("%%word%d%%", SearchQueryMaxWords-1), patterns[SearchQueryMaxWords-1])
}
