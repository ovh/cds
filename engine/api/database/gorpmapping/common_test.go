package gorpmapping_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/stretchr/testify/assert"
)

func TestIDsToQueryString(t *testing.T) {
	assert.Equal(t, "1,2,3", gorpmapping.IDsToQueryString([]int64{1, 2, 3}))
}

func TestToQueryString(t *testing.T) {
	assert.Equal(t, "1,2,3", gorpmapping.ToQueryString([]int64{1, 2, 3}))
}
