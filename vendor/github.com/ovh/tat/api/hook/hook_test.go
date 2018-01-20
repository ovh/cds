package hook

import (
	"testing"

	"github.com/ovh/tat"
	"github.com/stretchr/testify/assert"
)

func TestMatchCriteria(t *testing.T) {

	assert.Equal(t, false, matchCriteria(
		tat.Message{Labels: []tat.Label{{Text: "labelA", Color: "#eeeeee"}}},
		tat.FilterCriteria{Label: "labelB"}),
		"this message should not match")

	assert.Equal(t, true, matchCriteria(
		tat.Message{Labels: []tat.Label{{Text: "labelA", Color: "#eeeeee"}}},
		tat.FilterCriteria{Label: "labelA"}),
		"this message should match")

	assert.Equal(t, false, matchCriteria(
		tat.Message{Labels: []tat.Label{{Text: "labelA", Color: "#eeeeee"}}},
		tat.FilterCriteria{NotLabel: "labelA"}),
		"this message should not match")

	assert.Equal(t, true, matchCriteria(
		tat.Message{Labels: []tat.Label{{Text: "labelA", Color: "#eeeeee"}, {Text: "labelB", Color: "#eeeeee"}}},
		tat.FilterCriteria{AndLabel: "labelA"}),
		"this message should match")

	assert.Equal(t, true, matchCriteria(
		tat.Message{Labels: []tat.Label{{Text: "labelA", Color: "#eeeeee"}, {Text: "labelB", Color: "#eeeeee"}}},
		tat.FilterCriteria{AndLabel: "labelA,labelB"}),
		"this message should match")

	assert.Equal(t, false, matchCriteria(
		tat.Message{Labels: []tat.Label{{Text: "labelA", Color: "#eeeeee"}, {Text: "labelB", Color: "#eeeeee"}}},
		tat.FilterCriteria{AndLabel: "labelA,labelB,labelC"}),
		"this message should not match")

	assert.Equal(t, false, matchCriteria(
		tat.Message{Tags: []string{"tagA"}},
		tat.FilterCriteria{Tag: "tagB"}),
		"this message should not match")

	assert.Equal(t, true, matchCriteria(
		tat.Message{Tags: []string{"tagA"}},
		tat.FilterCriteria{Tag: "tagA"}),
		"this message should match")

	assert.Equal(t, false, matchCriteria(
		tat.Message{Tags: []string{"tagA"}},
		tat.FilterCriteria{NotTag: "tagA"}),
		"this message should not match")

	assert.Equal(t, true, matchCriteria(
		tat.Message{Tags: []string{"tagA", "tagB"}},
		tat.FilterCriteria{AndTag: "tagA"}),
		"this message should match")

	assert.Equal(t, true, matchCriteria(
		tat.Message{Tags: []string{"tagA", "tagB"}},
		tat.FilterCriteria{AndTag: "tagA,tagB"}),
		"this message should match")

	assert.Equal(t, false, matchCriteria(
		tat.Message{Tags: []string{"tagA", "tagB"}},
		tat.FilterCriteria{AndTag: "tagA,tagB,tagC"}),
		"this message should not match")

	assert.Equal(t, true, matchCriteria(
		tat.Message{InReplyOfID: ""},
		tat.FilterCriteria{OnlyMsgRoot: true}),
		"this message should match")

	assert.Equal(t, false, matchCriteria(
		tat.Message{InReplyOfID: "fff"},
		tat.FilterCriteria{OnlyMsgRoot: true}),
		"this message should not match")

	assert.Equal(t, true, matchCriteria(
		tat.Message{Author: tat.Author{Username: "foo"}},
		tat.FilterCriteria{Username: "foo"}),
		"this message should match")

	assert.Equal(t, false, matchCriteria(
		tat.Message{Author: tat.Author{Username: "foo"}},
		tat.FilterCriteria{Username: "bar"}),
		"this message should not match")

	assert.Equal(t, true, matchCriteria(
		tat.Message{Labels: []tat.Label{{Text: "labelA", Color: "#eeeeee"}, {Text: "labelB", Color: "#eeeeee"}}, Tags: []string{"tagA", "tagB", "tagC"}},
		tat.FilterCriteria{AndTag: "tagA,tagB", AndLabel: "labelA,labelB"}),
		"this message should match")
}
