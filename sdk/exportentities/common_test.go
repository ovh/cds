package exportentities

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadURL(t *testing.T) {
	b, f, err := ReadURL("https://raw.githubusercontent.com/ovh/cds/master/contrib/actions/cds-docker-package.hcl", "hcl")
	fmt.Println(len(b), f, err)
	assert.True(t, len(b) > 0)

	b, f, err = ReadURL("https://raw.githubusercontent.com/ovh/tat/master/.travis.yml", "yml")
	fmt.Println(len(b), f, err)
	assert.True(t, len(b) > 0)
}
