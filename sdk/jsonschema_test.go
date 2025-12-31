package sdk_test

import (
	"testing"

	"github.com/ovh/cds/sdk"
)

func TestGetYAMLKeywordsFromJsonSchema(t *testing.T) {
	got := sdk.GetYAMLKeywordsFromJsonSchema()

	// DIsplay the obtained keywords
	for _, v := range got {
		t.Log(v)
	}
}
