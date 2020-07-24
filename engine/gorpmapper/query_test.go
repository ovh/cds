package gorpmapper_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/gorpmapper"
)

func TestToQueryString(t *testing.T) {
	assert.Equal(t, "1,2,3", gorpmapper.ToQueryString([]int64{1, 2, 3}))
}
