package exportentities_test

import (
	"testing"

	"github.com/ovh/cds/sdk/exportentities"

	"github.com/stretchr/testify/assert"
)

func TestReadURL(t *testing.T) {
	b, _, _ := exportentities.ReadURL("https://raw.githubusercontent.com/ovh/tat/master/.travis.yml", "yml")
	assert.True(t, len(b) > 0)
}
