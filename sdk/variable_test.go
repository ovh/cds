package sdk_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func TestVariablesMerge(t *testing.T) {
	left := []sdk.Variable{{
		Name:  "A",
		Value: "abc",
	}, {
		Name:  "B",
		Value: "def",
	}, {
		Name:  "C",
		Value: "ghi",
	}}

	right := []sdk.Variable{{
		Name:  "B",
		Value: "123",
	}, {
		Name:  "D",
		Value: "456",
	}}

	res := sdk.VariablesMerge(left, right)
	assert.Equal(t, 4, len(res))
	sort.Slice(res, func(i, j int) bool { return res[i].Name < res[j].Name })

	assert.Equal(t, "A", res[0].Name)
	assert.Equal(t, "abc", res[0].Value)
	assert.Equal(t, "B", res[1].Name)
	assert.Equal(t, "123", res[1].Value)
	assert.Equal(t, "C", res[2].Name)
	assert.Equal(t, "ghi", res[2].Value)
	assert.Equal(t, "D", res[3].Name)
	assert.Equal(t, "456", res[3].Value)
}
