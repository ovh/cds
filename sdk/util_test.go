package sdk_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func TestStringFirstN(t *testing.T) {
	hash := "63cea1cffc5c9872b4741e5bb31ec85c0634a681"
	hashShort := sdk.StringFirstN(hash, 7)
	assert.Equal(t, "63cea1c", hashShort)
	assert.Equal(t, "63cea1c", sdk.StringFirstN(hash, 7))
	assert.Equal(t, "63ce", sdk.StringFirstN("63ce", 7))
	assert.Equal(t, "", sdk.StringFirstN("", 7))
}
