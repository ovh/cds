package exportentities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadURL(t *testing.T) {
	b, _, _ := ReadURL("https://raw.githubusercontent.com/ovh/tat/master/.travis.yml", "yml")
	assert.True(t, len(b) > 0)
}
